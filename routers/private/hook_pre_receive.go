// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package private

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/models"
	asymkey_model "code.gitea.io/gitea/models/asymkey"
	git_model "code.gitea.io/gitea/models/git"
	issues_model "code.gitea.io/gitea/models/issues"
	perm_model "code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
	"code.gitea.io/gitea/modules/web"
	pull_service "code.gitea.io/gitea/services/pull"
)

type preReceiveContext struct {
	*gitea_context.PrivateContext

	// loadedPusher indicates that where the following information are loaded
	loadedPusher        bool
	user                *user_model.User // it's the org user if a DeployKey is used
	userPerm            access_model.Permission
	deployKeyAccessMode perm_model.AccessMode

	canCreatePullRequest        bool
	checkedCanCreatePullRequest bool

	canWriteCode        bool
	checkedCanWriteCode bool

	protectedTags    []*git_model.ProtectedTag
	gotProtectedTags bool

	env []string

	opts *private.HookOptions

	branchName string
}

// CanWriteCode returns true if pusher can write code
func (ctx *preReceiveContext) CanWriteCode() bool {
	if !ctx.checkedCanWriteCode {
		if !ctx.loadPusherAndPermission() {
			return false
		}
		ctx.canWriteCode = issues_model.CanMaintainerWriteToBranch(ctx.userPerm, ctx.branchName, ctx.user) || ctx.deployKeyAccessMode >= perm_model.AccessModeWrite
		ctx.checkedCanWriteCode = true
	}
	return ctx.canWriteCode
}

// AssertCanWriteCode returns true if pusher can write code
func (ctx *preReceiveContext) AssertCanWriteCode() bool {
	if !ctx.CanWriteCode() {
		if ctx.Written() {
			return false
		}
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: "User permission denied for writing.",
		})
		return false
	}
	return true
}

// CanCreatePullRequest returns true if pusher can create pull requests
func (ctx *preReceiveContext) CanCreatePullRequest() bool {
	if !ctx.checkedCanCreatePullRequest {
		if !ctx.loadPusherAndPermission() {
			return false
		}
		ctx.canCreatePullRequest = ctx.userPerm.CanRead(unit.TypePullRequests)
		ctx.checkedCanCreatePullRequest = true
	}
	return ctx.canCreatePullRequest
}

// AssertCreatePullRequest returns true if can create pull requests
func (ctx *preReceiveContext) AssertCreatePullRequest() bool {
	if !ctx.CanCreatePullRequest() {
		if ctx.Written() {
			return false
		}
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: "User permission denied for creating pull-request.",
		})
		return false
	}
	return true
}

// CalculateSizeOfObject calculates the size of one git object via git cat-file -s command
func CalculateSizeOfObject(ctx *gitea_context.PrivateContext, opts *git.RunOpts, objectID string) (objectSize int64) {
	objectSizeStr, _, err := git.NewCommand(ctx, "cat-file", "-s").AddDynamicArguments(objectID).RunStdString(opts)
	if err != nil {
		log.Trace("CalculateSizeOfRemovedObjects: Error during git cat-file -s on object: %s", objectID)
		return
	}

	objectSize, _ = strconv.ParseInt(strings.TrimSpace(objectSizeStr), 10, 64)
	if err != nil {
		log.Trace("CalculateSizeOfRemovedObjects: Error during ParseInt on string '%s'", objectID)
		return
	}
	return
}

// CalculateSizeOfObjects calculates the size of objects added and removed from the repository by new commit
func CalculateSizeOfObjects(ctx *gitea_context.PrivateContext, opts *git.RunOpts, newCommitObjects map[string]bool, oldCommitObjects map[string]bool, otherCommitObjects map[string]bool) (addedSize int64, removedSize int64) {

	// Calculate size of objects that were added
	for objectID := range newCommitObjects {
		if _, exists := oldCommitObjects[objectID]; !exists {
			// objectID is not referenced in the list of objects of old commit so it is a new object
			// Calculate its size and add it to the addedSize
			addedSize += CalculateSizeOfObject(ctx, opts, objectID)
		}
		// We might check here if new object is not already in the rest of repo to be precise
		// However our goal is to prevent growth of repository so on determination of addedSize
		// We can skip this preciseness, addedSize will be more then real addedSize
		// TODO - do not count size of object that is referenced in other part of repo but not referenced neither in old nor new commit
		//        git will not add the object twice
	}

	// Calculate size of objects that were removed
	for objectID := range oldCommitObjects {
		if _, exists := newCommitObjects[objectID]; !exists {
			// objectID is not referenced in the list of new commit objects so it was possibly removed
			if _, exists := otherCommitObjects[objectID]; !exists {
				// objectID is not referenced in rest of the objects of the repository so it was removed
				// Calculate its size and add it to the addedSize
				removedSize += CalculateSizeOfObject(ctx, opts, objectID)
			}
		}

	}
	return
}

// ConvertObjectsToMap takes a newline-separated string of git objects and
// converts it into a map for efficient lookup.
func ConvertObjectsToMap(objects string) map[string]bool {
	objectsMap := make(map[string]bool)
	for _, object := range strings.Split(objects, "\n") {
		if len(object) == 0 {
			continue
		}
		objectID := strings.Split(object, " ")[0]
		objectsMap[objectID] = true
	}
	return objectsMap
}

// HookPreReceive checks whether a individual commit is acceptable
func HookPreReceive(ctx *gitea_context.PrivateContext) {
	startTime := time.Now()

	opts := web.GetForm(ctx).(*private.HookOptions)

	ourCtx := &preReceiveContext{
		PrivateContext: ctx,
		env:            generateGitEnv(opts), // Generate git environment for checking commits
		opts:           opts,
	}

	repo := ourCtx.Repo.Repository

	var addedSize int64
	var removedSize int64

	// Calculating total size of the push using git count-objects
	pushSize, err := git.CountObjectsWithEnv(ctx, repo.RepoPath(), ourCtx.env)
	if err != nil {
		log.Error("Unable to get repository size with env %v: %s Error: %v", repo.RepoPath(), ourCtx.env, err)
		ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"err": err.Error(),
		})
		return
	}

	// Cache whether the repository would breach the size limit after the operation
	isRepoOversized := repo.RepoSizeIsOversized(pushSize.Size)
	log.Trace("Push size %d", pushSize.Size)

	// Iterate across the provided old commit IDs
	for i := range opts.OldCommitIDs {
		oldCommitID := opts.OldCommitIDs[i]
		newCommitID := opts.NewCommitIDs[i]
		refFullName := opts.RefFullNames[i]

		// If operation is in potential breach of size limit prepare data for analysis
		if isRepoOversized {
			// Create cache of objects in old commit
			gitObjects, _, err := git.NewCommand(ctx, "rev-list", "--objects").AddDynamicArguments(oldCommitID).RunStdString(&git.RunOpts{Dir: repo.RepoPath(), Env: ourCtx.env})
			if err != nil {
				log.Error("Unable to list objects in old commit: %s in %-v Error: %v", oldCommitID, repo, err)
				ctx.JSON(http.StatusInternalServerError, private.Response{
					Err: fmt.Sprintf("Fail to list objects in old commit: %v", err),
				})
				return
			}
			oldCommitObjects := ConvertObjectsToMap(gitObjects)

			// Create cache of objects in new commit
			gitObjects, _, err = git.NewCommand(ctx, "rev-list", "--objects").AddDynamicArguments(newCommitID).RunStdString(&git.RunOpts{Dir: repo.RepoPath(), Env: ourCtx.env})
			if err != nil {
				log.Error("Unable to list objects in new commit %s in %-v Error: %v", newCommitID, repo, err)
				ctx.JSON(http.StatusInternalServerError, private.Response{
					Err: fmt.Sprintf("Fail to list objects in new commit: %v", err),
				})
				return
			}
			newCommitObjects := ConvertObjectsToMap(gitObjects)

			// Create cache of objects that are in the repository but not part of old or new commit
			gitObjects, _, err = git.NewCommand(ctx, "rev-list", "--objects", "--all").AddDynamicArguments("^"+oldCommitID, "^"+newCommitID).RunStdString(&git.RunOpts{Dir: repo.RepoPath(), Env: ourCtx.env})
			if err != nil {
				log.Error("Unable to list objects in the repo that are missing from both old %s and new %s commits in %-v Error: %v", oldCommitID, newCommitID, repo, err)
				ctx.JSON(http.StatusInternalServerError, private.Response{
					Err: fmt.Sprintf("Fail to list objects missing from both old and new commits: %v", err),
				})
				return
			}
			otherCommitObjects := ConvertObjectsToMap(gitObjects)

			// Calculate size that was added and removed by the new commit
			addedSize, removedSize = CalculateSizeOfObjects(ctx, &git.RunOpts{Dir: repo.RepoPath(), Env: ourCtx.env}, newCommitObjects, oldCommitObjects, otherCommitObjects)
		}

		switch {
		case strings.HasPrefix(refFullName, git.BranchPrefix):
			preReceiveBranch(ourCtx, oldCommitID, newCommitID, refFullName)
		case strings.HasPrefix(refFullName, git.TagPrefix):
			preReceiveTag(ourCtx, oldCommitID, newCommitID, refFullName)
		case git.SupportProcReceive && strings.HasPrefix(refFullName, git.PullRequestPrefix):
			preReceivePullRequest(ourCtx, oldCommitID, newCommitID, refFullName)
		default:
			ourCtx.AssertCanWriteCode()
		}
		if ctx.Written() {
			return
		}
	}

	duration := time.Since(startTime)
	log.Warn("Addition in size is: %d, removal in size is: %d, limit size: %s, push size: %d. Took %s seconds.", addedSize, removedSize, base.FileSize(repo.GetActualSizeLimit()), pushSize.Size, duration)

	// If total of commits add more size then they remove and we are in a potential breach of size limit -- abort
	if (addedSize > removedSize) && isRepoOversized {
		log.Warn("Forbidden: new repo size %s is over limitation of %s. Push size: %s. Took %s seconds.", base.FileSize(addedSize-removedSize), base.FileSize(repo.GetActualSizeLimit()), base.FileSize(pushSize.Size), duration)
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: fmt.Sprintf("Repository size is over limitation of %s", base.FileSize(repo.GetActualSizeLimit())),
		})
		return
	}

	ctx.PlainText(http.StatusOK, "ok")
}

func preReceiveBranch(ctx *preReceiveContext, oldCommitID, newCommitID, refFullName string) {
	branchName := strings.TrimPrefix(refFullName, git.BranchPrefix)
	ctx.branchName = branchName

	if !ctx.AssertCanWriteCode() {
		return
	}

	repo := ctx.Repo.Repository
	gitRepo := ctx.Repo.GitRepo

	if branchName == repo.DefaultBranch && newCommitID == git.EmptySHA {
		log.Warn("Forbidden: Branch: %s is the default branch in %-v and cannot be deleted", branchName, repo)
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: fmt.Sprintf("branch %s is the default branch and cannot be deleted", branchName),
		})
		return
	}

	protectBranch, err := git_model.GetFirstMatchProtectedBranchRule(ctx, repo.ID, branchName)
	if err != nil {
		log.Error("Unable to get protected branch: %s in %-v Error: %v", branchName, repo, err)
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: err.Error(),
		})
		return
	}

	// Allow pushes to non-protected branches
	if protectBranch == nil {
		return
	}
	protectBranch.Repo = repo

	// This ref is a protected branch.
	//
	// First of all we need to enforce absolutely:
	//
	// 1. Detect and prevent deletion of the branch
	if newCommitID == git.EmptySHA {
		log.Warn("Forbidden: Branch: %s in %-v is protected from deletion", branchName, repo)
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: fmt.Sprintf("branch %s is protected from deletion", branchName),
		})
		return
	}

	// 2. Disallow force pushes to protected branches
	if git.EmptySHA != oldCommitID {
		output, _, err := git.NewCommand(ctx, "rev-list", "--max-count=1").AddDynamicArguments(oldCommitID, "^"+newCommitID).RunStdString(&git.RunOpts{Dir: repo.RepoPath(), Env: ctx.env})
		if err != nil {
			log.Error("Unable to detect force push between: %s and %s in %-v Error: %v", oldCommitID, newCommitID, repo, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Fail to detect force push: %v", err),
			})
			return
		} else if len(output) > 0 {
			log.Warn("Forbidden: Branch: %s in %-v is protected from force push", branchName, repo)
			ctx.JSON(http.StatusForbidden, private.Response{
				UserMsg: fmt.Sprintf("branch %s is protected from force push", branchName),
			})
			return

		}
	}

	// 3. Enforce require signed commits
	if protectBranch.RequireSignedCommits {
		err := verifyCommits(oldCommitID, newCommitID, gitRepo, ctx.env)
		if err != nil {
			if !isErrUnverifiedCommit(err) {
				log.Error("Unable to check commits from %s to %s in %-v: %v", oldCommitID, newCommitID, repo, err)
				ctx.JSON(http.StatusInternalServerError, private.Response{
					Err: fmt.Sprintf("Unable to check commits from %s to %s: %v", oldCommitID, newCommitID, err),
				})
				return
			}
			unverifiedCommit := err.(*errUnverifiedCommit).sha
			log.Warn("Forbidden: Branch: %s in %-v is protected from unverified commit %s", branchName, repo, unverifiedCommit)
			ctx.JSON(http.StatusForbidden, private.Response{
				UserMsg: fmt.Sprintf("branch %s is protected from unverified commit %s", branchName, unverifiedCommit),
			})
			return
		}
	}

	// Now there are several tests which can be overridden:
	//
	// 4. Check protected file patterns - this is overridable from the UI
	changedProtectedfiles := false
	protectedFilePath := ""

	globs := protectBranch.GetProtectedFilePatterns()
	if len(globs) > 0 {
		_, err := pull_service.CheckFileProtection(gitRepo, oldCommitID, newCommitID, globs, 1, ctx.env)
		if err != nil {
			if !models.IsErrFilePathProtected(err) {
				log.Error("Unable to check file protection for commits from %s to %s in %-v: %v", oldCommitID, newCommitID, repo, err)
				ctx.JSON(http.StatusInternalServerError, private.Response{
					Err: fmt.Sprintf("Unable to check file protection for commits from %s to %s: %v", oldCommitID, newCommitID, err),
				})
				return
			}

			changedProtectedfiles = true
			protectedFilePath = err.(models.ErrFilePathProtected).Path
		}
	}

	// 5. Check if the doer is allowed to push
	var canPush bool
	if ctx.opts.DeployKeyID != 0 {
		canPush = !changedProtectedfiles && protectBranch.CanPush && (!protectBranch.EnableWhitelist || protectBranch.WhitelistDeployKeys)
	} else {
		user, err := user_model.GetUserByID(ctx, ctx.opts.UserID)
		if err != nil {
			log.Error("Unable to GetUserByID for commits from %s to %s in %-v: %v", oldCommitID, newCommitID, repo, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to GetUserByID for commits from %s to %s: %v", oldCommitID, newCommitID, err),
			})
			return
		}
		canPush = !changedProtectedfiles && protectBranch.CanUserPush(ctx, user)
	}

	// 6. If we're not allowed to push directly
	if !canPush {
		// Is this is a merge from the UI/API?
		if ctx.opts.PullRequestID == 0 {
			// 6a. If we're not merging from the UI/API then there are two ways we got here:
			//
			// We are changing a protected file and we're not allowed to do that
			if changedProtectedfiles {
				log.Warn("Forbidden: Branch: %s in %-v is protected from changing file %s", branchName, repo, protectedFilePath)
				ctx.JSON(http.StatusForbidden, private.Response{
					UserMsg: fmt.Sprintf("branch %s is protected from changing file %s", branchName, protectedFilePath),
				})
				return
			}

			// Allow commits that only touch unprotected files
			globs := protectBranch.GetUnprotectedFilePatterns()
			if len(globs) > 0 {
				unprotectedFilesOnly, err := pull_service.CheckUnprotectedFiles(gitRepo, oldCommitID, newCommitID, globs, ctx.env)
				if err != nil {
					log.Error("Unable to check file protection for commits from %s to %s in %-v: %v", oldCommitID, newCommitID, repo, err)
					ctx.JSON(http.StatusInternalServerError, private.Response{
						Err: fmt.Sprintf("Unable to check file protection for commits from %s to %s: %v", oldCommitID, newCommitID, err),
					})
					return
				}
				if unprotectedFilesOnly {
					// Commit only touches unprotected files, this is allowed
					return
				}
			}

			// Or we're simply not able to push to this protected branch
			log.Warn("Forbidden: User %d is not allowed to push to protected branch: %s in %-v", ctx.opts.UserID, branchName, repo)
			ctx.JSON(http.StatusForbidden, private.Response{
				UserMsg: fmt.Sprintf("Not allowed to push to protected branch %s", branchName),
			})
			return
		}
		// 6b. Merge (from UI or API)

		// Get the PR, user and permissions for the user in the repository
		pr, err := issues_model.GetPullRequestByID(ctx, ctx.opts.PullRequestID)
		if err != nil {
			log.Error("Unable to get PullRequest %d Error: %v", ctx.opts.PullRequestID, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to get PullRequest %d Error: %v", ctx.opts.PullRequestID, err),
			})
			return
		}

		// although we should have called `loadPusherAndPermission` before, here we call it explicitly again because we need to access ctx.user below
		if !ctx.loadPusherAndPermission() {
			// if error occurs, loadPusherAndPermission had written the error response
			return
		}

		// Now check if the user is allowed to merge PRs for this repository
		// Note: we can use ctx.perm and ctx.user directly as they will have been loaded above
		allowedMerge, err := pull_service.IsUserAllowedToMerge(ctx, pr, ctx.userPerm, ctx.user)
		if err != nil {
			log.Error("Error calculating if allowed to merge: %v", err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Error calculating if allowed to merge: %v", err),
			})
			return
		}

		if !allowedMerge {
			log.Warn("Forbidden: User %d is not allowed to push to protected branch: %s in %-v and is not allowed to merge pr #%d", ctx.opts.UserID, branchName, repo, pr.Index)
			ctx.JSON(http.StatusForbidden, private.Response{
				UserMsg: fmt.Sprintf("Not allowed to push to protected branch %s", branchName),
			})
			return
		}

		// If we're an admin for the repository we can ignore status checks, reviews and override protected files
		if ctx.userPerm.IsAdmin() {
			return
		}

		// Now if we're not an admin - we can't overwrite protected files so fail now
		if changedProtectedfiles {
			log.Warn("Forbidden: Branch: %s in %-v is protected from changing file %s", branchName, repo, protectedFilePath)
			ctx.JSON(http.StatusForbidden, private.Response{
				UserMsg: fmt.Sprintf("branch %s is protected from changing file %s", branchName, protectedFilePath),
			})
			return
		}

		// Check all status checks and reviews are ok
		if err := pull_service.CheckPullBranchProtections(ctx, pr, true); err != nil {
			if models.IsErrDisallowedToMerge(err) {
				log.Warn("Forbidden: User %d is not allowed push to protected branch %s in %-v and pr #%d is not ready to be merged: %s", ctx.opts.UserID, branchName, repo, pr.Index, err.Error())
				ctx.JSON(http.StatusForbidden, private.Response{
					UserMsg: fmt.Sprintf("Not allowed to push to protected branch %s and pr #%d is not ready to be merged: %s", branchName, ctx.opts.PullRequestID, err.Error()),
				})
				return
			}
			log.Error("Unable to check if mergable: protected branch %s in %-v and pr #%d. Error: %v", ctx.opts.UserID, branchName, repo, pr.Index, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to get status of pull request %d. Error: %v", ctx.opts.PullRequestID, err),
			})
			return
		}
	}
}

func preReceiveTag(ctx *preReceiveContext, oldCommitID, newCommitID, refFullName string) {
	if !ctx.AssertCanWriteCode() {
		return
	}

	tagName := strings.TrimPrefix(refFullName, git.TagPrefix)

	if !ctx.gotProtectedTags {
		var err error
		ctx.protectedTags, err = git_model.GetProtectedTags(ctx, ctx.Repo.Repository.ID)
		if err != nil {
			log.Error("Unable to get protected tags for %-v Error: %v", ctx.Repo.Repository, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: err.Error(),
			})
			return
		}
		ctx.gotProtectedTags = true
	}

	isAllowed, err := git_model.IsUserAllowedToControlTag(ctx, ctx.protectedTags, tagName, ctx.opts.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: err.Error(),
		})
		return
	}
	if !isAllowed {
		log.Warn("Forbidden: Tag %s in %-v is protected", tagName, ctx.Repo.Repository)
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: fmt.Sprintf("Tag %s is protected", tagName),
		})
		return
	}
}

func preReceivePullRequest(ctx *preReceiveContext, oldCommitID, newCommitID, refFullName string) {
	if !ctx.AssertCreatePullRequest() {
		return
	}

	if ctx.Repo.Repository.IsEmpty {
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: "Can't create pull request for an empty repository.",
		})
		return
	}

	if ctx.opts.IsWiki {
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: "Pull requests are not supported on the wiki.",
		})
		return
	}

	baseBranchName := refFullName[len(git.PullRequestPrefix):]

	baseBranchExist := false
	if ctx.Repo.GitRepo.IsBranchExist(baseBranchName) {
		baseBranchExist = true
	}

	if !baseBranchExist {
		for p, v := range baseBranchName {
			if v == '/' && ctx.Repo.GitRepo.IsBranchExist(baseBranchName[:p]) && p != len(baseBranchName)-1 {
				baseBranchExist = true
				break
			}
		}
	}

	if !baseBranchExist {
		ctx.JSON(http.StatusForbidden, private.Response{
			UserMsg: fmt.Sprintf("Unexpected ref: %s", refFullName),
		})
		return
	}
}

func generateGitEnv(opts *private.HookOptions) (env []string) {
	env = os.Environ()
	if opts.GitAlternativeObjectDirectories != "" {
		env = append(env,
			private.GitAlternativeObjectDirectories+"="+opts.GitAlternativeObjectDirectories)
	}
	if opts.GitObjectDirectory != "" {
		env = append(env,
			private.GitObjectDirectory+"="+opts.GitObjectDirectory)
	}
	if opts.GitQuarantinePath != "" {
		env = append(env,
			private.GitQuarantinePath+"="+opts.GitQuarantinePath)
	}
	return env
}

// loadPusherAndPermission returns false if an error occurs, and it writes the error response
func (ctx *preReceiveContext) loadPusherAndPermission() bool {
	if ctx.loadedPusher {
		return true
	}

	if ctx.opts.UserID == user_model.ActionsUserID {
		ctx.user = user_model.NewActionsUser()
		ctx.userPerm.AccessMode = perm_model.AccessMode(ctx.opts.ActionPerm)
		if err := ctx.Repo.Repository.LoadUnits(ctx); err != nil {
			log.Error("Unable to get User id %d Error: %v", ctx.opts.UserID, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to get User id %d Error: %v", ctx.opts.UserID, err),
			})
			return false
		}
		ctx.userPerm.Units = ctx.Repo.Repository.Units
		ctx.userPerm.UnitsMode = make(map[unit.Type]perm_model.AccessMode)
		for _, u := range ctx.Repo.Repository.Units {
			ctx.userPerm.UnitsMode[u.Type] = ctx.userPerm.AccessMode
		}
	} else {
		user, err := user_model.GetUserByID(ctx, ctx.opts.UserID)
		if err != nil {
			log.Error("Unable to get User id %d Error: %v", ctx.opts.UserID, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to get User id %d Error: %v", ctx.opts.UserID, err),
			})
			return false
		}
		ctx.user = user
		userPerm, err := access_model.GetUserRepoPermission(ctx, ctx.Repo.Repository, user)
		if err != nil {
			log.Error("Unable to get Repo permission of repo %s/%s of User %s: %v", ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name, user.Name, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to get Repo permission of repo %s/%s of User %s: %v", ctx.Repo.Repository.OwnerName, ctx.Repo.Repository.Name, user.Name, err),
			})
			return false
		}
		ctx.userPerm = userPerm
	}

	if ctx.opts.DeployKeyID != 0 {
		deployKey, err := asymkey_model.GetDeployKeyByID(ctx, ctx.opts.DeployKeyID)
		if err != nil {
			log.Error("Unable to get DeployKey id %d Error: %v", ctx.opts.DeployKeyID, err)
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to get DeployKey id %d Error: %v", ctx.opts.DeployKeyID, err),
			})
			return false
		}
		ctx.deployKeyAccessMode = deployKey.Mode
	}

	ctx.loadedPusher = true
	return true
}
