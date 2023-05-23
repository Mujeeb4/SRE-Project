// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/charset"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/lfs"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	asymkey_service "code.gitea.io/gitea/services/asymkey"

	stdcharset "golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// IdentityOptions for a person's identity like an author or committer
type IdentityOptions struct {
	Name  string
	Email string
}

// CommitDateOptions store dates for GIT_AUTHOR_DATE and GIT_COMMITTER_DATE
type CommitDateOptions struct {
	Author    time.Time
	Committer time.Time
}

type ChangeRepoFile struct {
	Operation    string
	TreePath     string
	FromTreePath string
	Content      string
	SHA          string
}

// UpdateRepoFilesOptions holds the repository files update options
type ChangeRepoFilesOptions struct {
	LastCommitID string
	OldBranch    string
	NewBranch    string
	Message      string
	Files        []*ChangeRepoFile
	Author       *IdentityOptions
	Committer    *IdentityOptions
	Dates        *CommitDateOptions
	Signoff      bool
}

type RepoFileOptions struct {
	treePath     string
	fromTreePath string
	encoding     string
	bom          bool
	executable   bool
}

func detectEncodingAndBOM(entry *git.TreeEntry, repo *repo_model.Repository) (string, bool) {
	reader, err := entry.Blob().DataAsync()
	if err != nil {
		// return default
		return "UTF-8", false
	}
	defer reader.Close()
	buf := make([]byte, 1024)
	n, err := util.ReadAtMost(reader, buf)
	if err != nil {
		// return default
		return "UTF-8", false
	}
	buf = buf[:n]

	if setting.LFS.StartServer {
		pointer, _ := lfs.ReadPointerFromBuffer(buf)
		if pointer.IsValid() {
			meta, err := git_model.GetLFSMetaObjectByOid(db.DefaultContext, repo.ID, pointer.Oid)
			if err != nil && err != git_model.ErrLFSObjectNotExist {
				// return default
				return "UTF-8", false
			}
			if meta != nil {
				dataRc, err := lfs.ReadMetaObject(pointer)
				if err != nil {
					// return default
					return "UTF-8", false
				}
				defer dataRc.Close()
				buf = make([]byte, 1024)
				n, err = util.ReadAtMost(dataRc, buf)
				if err != nil {
					// return default
					return "UTF-8", false
				}
				buf = buf[:n]
			}
		}
	}

	encoding, err := charset.DetectEncoding(buf)
	if err != nil {
		// just default to utf-8 and no bom
		return "UTF-8", false
	}
	if encoding == "UTF-8" {
		return encoding, bytes.Equal(buf[0:3], charset.UTF8BOM)
	}
	charsetEncoding, _ := stdcharset.Lookup(encoding)
	if charsetEncoding == nil {
		return "UTF-8", false
	}

	result, n, err := transform.String(charsetEncoding.NewDecoder(), string(buf))
	if err != nil {
		// return default
		return "UTF-8", false
	}

	if n > 2 {
		return encoding, bytes.Equal([]byte(result)[0:3], charset.UTF8BOM)
	}

	return encoding, false
}

// ChangeRepoFiles adds, updates or removes multiple files in the given repository
func ChangeRepoFiles(ctx context.Context, repo *repo_model.Repository, doer *user_model.User, opts *ChangeRepoFilesOptions) ([]*structs.FileResponse, error) {
	// If no branch name is set, assume default branch
	if opts.OldBranch == "" {
		opts.OldBranch = repo.DefaultBranch
	}
	if opts.NewBranch == "" {
		opts.NewBranch = opts.OldBranch
	}

	gitRepo, closer, err := git.RepositoryFromContextOrOpen(ctx, repo.RepoPath())
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	// oldBranch must exist for this operation
	if _, err := gitRepo.GetBranch(opts.OldBranch); err != nil && !repo.IsEmpty {
		return nil, err
	}

	treePaths := []string{}
	for _, file := range opts.Files {
		treePaths = append(treePaths, file.TreePath)
	}

	// A NewBranch can be specified for the file to be created/updated in a new branch.
	// Check to make sure the branch does not already exist, otherwise we can't proceed.
	// If we aren't branching to a new branch, make sure user can commit to the given branch
	if opts.NewBranch != opts.OldBranch {
		existingBranch, err := gitRepo.GetBranch(opts.NewBranch)
		if existingBranch != nil {
			return nil, models.ErrBranchAlreadyExists{
				BranchName: opts.NewBranch,
			}
		}
		if err != nil && !git.IsErrBranchNotExist(err) {
			return nil, err
		}
	} else if err := VerifyBranchProtection(ctx, repo, doer, opts.OldBranch, treePaths); err != nil {
		return nil, err
	}

	message := strings.TrimSpace(opts.Message)

	author, committer := GetAuthorAndCommitterUsers(opts.Author, opts.Committer, doer)

	t, err := NewTemporaryUploadRepository(ctx, repo)
	if err != nil {
		log.Error("%v", err)
	}
	defer t.Close()

	hasOldBranch := true
	err = t.Clone(opts.OldBranch)
	if err != nil {
		if !git.IsErrBranchNotExist(err) || !repo.IsEmpty {
			return nil, err
		}
		if err := t.Init(); err != nil {
			return nil, err
		}
	}

	filesOptions := map[string]*RepoFileOptions{}
	for _, file := range opts.Files {
		switch file.Operation {
		case "create":
			if err != nil {
				hasOldBranch = false
				opts.LastCommitID = ""
			}
		case "delete", "update":
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("Invalid file operation: %s %s", file.Operation, file.TreePath)
		}

		// If FromTreePath is not set, set it to the opts.TreePath
		if file.TreePath != "" && file.FromTreePath == "" {
			file.FromTreePath = file.TreePath
		}

		// Check that the path given in opts.treePath is valid (not a git path)
		treePath := CleanUploadFileName(file.TreePath)
		if treePath == "" {
			return nil, models.ErrFilenameInvalid{
				Path: file.TreePath,
			}
		}
		// If there is a fromTreePath (we are copying it), also clean it up
		fromTreePath := CleanUploadFileName(file.FromTreePath)
		if fromTreePath == "" && file.FromTreePath != "" {
			return nil, models.ErrFilenameInvalid{
				Path: file.FromTreePath,
			}
		}

		filesOptions[file.TreePath] = &RepoFileOptions{
			treePath:     treePath,
			fromTreePath: fromTreePath,
			encoding:     "UTF-8",
			bom:          false,
			executable:   false,
		}
	}

	if hasOldBranch {
		if err := t.SetDefaultIndex(); err != nil {
			return nil, err
		}
	}

	for _, file := range opts.Files {
		if file.Operation == "delete" {
			// Get the files in the index
			filesInIndex, err := t.LsFiles(file.TreePath)
			if err != nil {
				return nil, fmt.Errorf("DeleteRepoFile: %w", err)
			}

			// Find the file we want to delete in the index
			inFilelist := false
			for _, indexFile := range filesInIndex {
				if indexFile == file.TreePath {
					inFilelist = true
					break
				}
			}
			if !inFilelist {
				return nil, models.ErrRepoFileDoesNotExist{
					Path: file.TreePath,
				}
			}
		}
	}

	if hasOldBranch {
		// Get the commit of the original branch
		commit, err := t.GetBranchCommit(opts.OldBranch)
		if err != nil {
			return nil, err // Couldn't get a commit for the branch
		}

		// Assigned LastCommitID in opts if it hasn't been set
		if opts.LastCommitID == "" {
			opts.LastCommitID = commit.ID.String()
		} else {
			lastCommitID, err := t.gitRepo.ConvertToSHA1(opts.LastCommitID)
			if err != nil {
				return nil, fmt.Errorf("ConvertToSHA1: Invalid last commit ID: %w", err)
			}
			opts.LastCommitID = lastCommitID.String()

		}

		for _, file := range opts.Files {
			fileOptions := filesOptions[file.TreePath]

			if file.Operation == "update" || file.Operation == "delete" {
				fromEntry, err := commit.GetTreeEntryByPath(fileOptions.fromTreePath)
				if err != nil {
					return nil, err
				}
				if file.SHA != "" {
					// If a SHA was given and the SHA given doesn't match the SHA of the fromTreePath, throw error
					if file.SHA != fromEntry.ID.String() {
						return nil, models.ErrSHADoesNotMatch{
							Path:       fileOptions.treePath,
							GivenSHA:   file.SHA,
							CurrentSHA: fromEntry.ID.String(),
						}
					}
				} else if opts.LastCommitID != "" {
					// If a lastCommitID was given and it doesn't match the commitID of the head of the branch throw
					// an error, but only if we aren't creating a new branch.
					if commit.ID.String() != opts.LastCommitID && opts.OldBranch == opts.NewBranch {
						if changed, err := commit.FileChangedSinceCommit(fileOptions.treePath, opts.LastCommitID); err != nil {
							return nil, err
						} else if changed {
							return nil, models.ErrCommitIDDoesNotMatch{
								GivenCommitID:   opts.LastCommitID,
								CurrentCommitID: opts.LastCommitID,
							}
						}
						// The file wasn't modified, so we are good to delete it
					}
				} else {
					// When updating a file, a lastCommitID or SHA needs to be given to make sure other commits
					// haven't been made. We throw an error if one wasn't provided.
					return nil, models.ErrSHAOrCommitIDNotProvided{}
				}
				filesOptions[file.TreePath].encoding, filesOptions[file.TreePath].bom = detectEncodingAndBOM(fromEntry, repo)
				filesOptions[file.TreePath].executable = fromEntry.IsExecutable()
			}
			if file.Operation == "create" || file.Operation == "update" {
				// For the path where this file will be created/updated, we need to make
				// sure no parts of the path are existing files or links except for the last
				// item in the path which is the file name, and that shouldn't exist IF it is
				// a new file OR is being moved to a new path.
				treePathParts := strings.Split(fileOptions.treePath, "/")
				subTreePath := ""
				for index, part := range treePathParts {
					subTreePath = path.Join(subTreePath, part)
					entry, err := commit.GetTreeEntryByPath(subTreePath)
					if err != nil {
						if git.IsErrNotExist(err) {
							// Means there is no item with that name, so we're good
							break
						}
						return nil, err
					}
					if index < len(treePathParts)-1 {
						if !entry.IsDir() {
							return nil, models.ErrFilePathInvalid{
								Message: fmt.Sprintf("a file exists where you’re trying to create a subdirectory [path: %s]", subTreePath),
								Path:    subTreePath,
								Name:    part,
								Type:    git.EntryModeBlob,
							}
						}
					} else if entry.IsLink() {
						return nil, models.ErrFilePathInvalid{
							Message: fmt.Sprintf("a symbolic link exists where you’re trying to create a subdirectory [path: %s]", subTreePath),
							Path:    subTreePath,
							Name:    part,
							Type:    git.EntryModeSymlink,
						}
					} else if entry.IsDir() {
						return nil, models.ErrFilePathInvalid{
							Message: fmt.Sprintf("a directory exists where you’re trying to create a file [path: %s]", subTreePath),
							Path:    subTreePath,
							Name:    part,
							Type:    git.EntryModeTree,
						}
					} else if fileOptions.fromTreePath != fileOptions.treePath || file.Operation == "create" {
						// The entry shouldn't exist if we are creating new file or moving to a new path
						return nil, models.ErrRepoFileAlreadyExists{
							Path: fileOptions.treePath,
						}
					}

				}
			}
		}
	}

	for _, file := range opts.Files {
		switch file.Operation {
		case "create", "update":
			fileOptions := filesOptions[file.TreePath]

			// Get the two paths (might be the same if not moving) from the index if they exist
			filesInIndex, err := t.LsFiles(file.TreePath, file.FromTreePath)
			if err != nil {
				return nil, fmt.Errorf("UpdateRepoFile: %w", err)
			}
			// If is a new file (not updating) then the given path shouldn't exist
			if file.Operation == "create" {
				for _, indexFile := range filesInIndex {
					if indexFile == file.TreePath {
						return nil, models.ErrRepoFileAlreadyExists{
							Path: file.TreePath,
						}
					}
				}
			}

			// Remove the old path from the tree
			if fileOptions.fromTreePath != fileOptions.treePath && len(filesInIndex) > 0 {
				for _, indexFile := range filesInIndex {
					if indexFile == fileOptions.fromTreePath {
						if err := t.RemoveFilesFromIndex(file.FromTreePath); err != nil {
							return nil, err
						}
					}
				}
			}

			content := file.Content
			if fileOptions.bom {
				content = string(charset.UTF8BOM) + content
			}
			if fileOptions.encoding != "UTF-8" {
				charsetEncoding, _ := stdcharset.Lookup(fileOptions.encoding)
				if charsetEncoding != nil {
					result, _, err := transform.String(charsetEncoding.NewEncoder(), content)
					if err != nil {
						// Look if we can't encode back in to the original we should just stick with utf-8
						log.Error("Error re-encoding %s (%s) as %s - will stay as UTF-8: %v", file.TreePath, file.FromTreePath, fileOptions.encoding, err)
						result = content
					}
					content = result
				} else {
					log.Error("Unknown encoding: %s", fileOptions.encoding)
				}
			}
			// Reset the opts.Content to our adjusted content to ensure that LFS gets the correct content
			file.Content = content
			var lfsMetaObject *git_model.LFSMetaObject

			if setting.LFS.StartServer && hasOldBranch {
				// Check there is no way this can return multiple infos
				filename2attribute2info, err := t.gitRepo.CheckAttribute(git.CheckAttributeOpts{
					Attributes: []string{"filter"},
					Filenames:  []string{fileOptions.treePath},
					CachedOnly: true,
				})
				if err != nil {
					return nil, err
				}

				if filename2attribute2info[fileOptions.treePath] != nil && filename2attribute2info[fileOptions.treePath]["filter"] == "lfs" {
					// OK so we are supposed to LFS this data!
					pointer, err := lfs.GeneratePointer(strings.NewReader(file.Content))
					if err != nil {
						return nil, err
					}
					lfsMetaObject = &git_model.LFSMetaObject{Pointer: pointer, RepositoryID: repo.ID}
					content = pointer.StringContent()
				}
			}

			// Add the object to the database
			objectHash, err := t.HashObject(strings.NewReader(content))
			if err != nil {
				return nil, err
			}

			// Add the object to the index
			if fileOptions.executable {
				if err := t.AddObjectToIndex("100755", objectHash, fileOptions.treePath); err != nil {
					return nil, err
				}
			} else {
				if err := t.AddObjectToIndex("100644", objectHash, fileOptions.treePath); err != nil {
					return nil, err
				}
			}

			if lfsMetaObject != nil {
				// We have an LFS object - create it
				lfsMetaObject, err = git_model.NewLFSMetaObject(ctx, lfsMetaObject)
				if err != nil {
					return nil, err
				}
				contentStore := lfs.NewContentStore()
				exist, err := contentStore.Exists(lfsMetaObject.Pointer)
				if err != nil {
					return nil, err
				}
				if !exist {
					if err := contentStore.Put(lfsMetaObject.Pointer, strings.NewReader(file.Content)); err != nil {
						if _, err2 := git_model.RemoveLFSMetaObjectByOid(ctx, repo.ID, lfsMetaObject.Oid); err2 != nil {
							return nil, fmt.Errorf("Error whilst removing failed inserted LFS object %s: %v (Prev Error: %w)", lfsMetaObject.Oid, err2, err)
						}
						return nil, err
					}
				}
			}
		case "delete":
			// Remove the file from the index
			if err := t.RemoveFilesFromIndex(file.TreePath); err != nil {
				return nil, err
			}
		}
	}

	// Now write the tree
	treeHash, err := t.WriteTree()
	if err != nil {
		return nil, err
	}

	// Now commit the tree
	var commitHash string
	if opts.Dates != nil {
		commitHash, err = t.CommitTreeWithDate(opts.LastCommitID, author, committer, treeHash, message, opts.Signoff, opts.Dates.Author, opts.Dates.Committer)
	} else {
		commitHash, err = t.CommitTree(opts.LastCommitID, author, committer, treeHash, message, opts.Signoff)
	}
	if err != nil {
		return nil, err
	}

	// Then push this tree to NewBranch
	if err := t.Push(doer, commitHash, opts.NewBranch); err != nil {
		log.Error("%T %v", err, err)
		return nil, err
	}

	commit, err := t.GetCommit(commitHash)
	if err != nil {
		return nil, err
	}

	files := []*structs.FileResponse{}

	for _, file := range opts.Files {
		fileResponse, err := GetFileResponseFromCommit(ctx, repo, commit, opts.NewBranch, filesOptions[file.TreePath].treePath)
		if err != nil {
			return nil, err
		}
		files = append(files, fileResponse)
	}

	if repo.IsEmpty {
		_ = repo_model.UpdateRepositoryCols(ctx, &repo_model.Repository{ID: repo.ID, IsEmpty: false}, "is_empty")
	}

	return files, nil
}

// VerifyBranchProtection verify the branch protection for modifying the given treePath on the given branch
func VerifyBranchProtection(ctx context.Context, repo *repo_model.Repository, doer *user_model.User, branchName string, treePaths []string) error {
	protectedBranch, err := git_model.GetFirstMatchProtectedBranchRule(ctx, repo.ID, branchName)
	if err != nil {
		return err
	}
	if protectedBranch != nil {
		protectedBranch.Repo = repo
		for _, treePath := range treePaths {
			isUnprotectedFile := false
			glob := protectedBranch.GetUnprotectedFilePatterns()
			if len(glob) != 0 {
				isUnprotectedFile = protectedBranch.IsUnprotectedFile(glob, treePath)
			}
			if !protectedBranch.CanUserPush(ctx, doer) && !isUnprotectedFile {
				return models.ErrUserCannotCommit{
					UserName: doer.LowerName,
				}
			}
			patterns := protectedBranch.GetProtectedFilePatterns()
			for _, pat := range patterns {
				if pat.Match(strings.ToLower(treePath)) {
					return models.ErrFilePathProtected{
						Path: treePath,
					}
				}
			}

		}
		if protectedBranch.RequireSignedCommits {
			_, _, _, err := asymkey_service.SignCRUDAction(ctx, repo.RepoPath(), doer, repo.RepoPath(), branchName)
			if err != nil {
				if !asymkey_service.IsErrWontSign(err) {
					return err
				}
				return models.ErrUserCannotCommit{
					UserName: doer.LowerName,
				}
			}
		}
	}
	return nil
}
