// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"

	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"

	"xorm.io/builder"
)

func SyncAllBranches(ctx context.Context, doerID int64) error {
	log.Trace("Synchronizing repository branches (this may take a while)")
	return db.Iterate(ctx, builder.Eq{"is_empty": false}, func(ctx context.Context, repo *repo_model.Repository) error {
		return SyncRepoBranches(ctx, repo.ID, doerID)
	})
}

// SyncRepoBranches synchronizes branch table with repository branches
func SyncRepoBranches(ctx context.Context, repoID, doerID int64) error {
	repo, err := repo_model.GetRepositoryByID(ctx, repoID)
	if err != nil {
		return err
	}

	log.Debug("SyncRepoBranches: in Repo[%d:%s]", repo.ID, repo.FullName())

	gitRepo, err := git.OpenRepository(ctx, repo.RepoPath())
	if err != nil {
		log.Error("OpenRepository[%s]: %w", repo.RepoPath(), err)
		return nil
	}
	defer gitRepo.Close()

	return SyncRepoBranchesWithRepo(ctx, repo, gitRepo, doerID)
}

func SyncRepoBranchesWithRepo(ctx context.Context, repo *repo_model.Repository, gitRepo *git.Repository, doerID int64) error {
	var allBranches []string
	for page := 0; ; page++ {
		branches, _, err := gitRepo.GetBranchNames(page*100, 100)
		if err != nil {
			return err
		}
		allBranches = append(allBranches, branches...)
		if len(branches) < 100 {
			break
		}
	}
	log.Trace("SyncRepoBranches[%s]: branches[%d]: %v", repo.FullName(), len(allBranches), allBranches)

	dbBranches, err := git_model.LoadAllBranches(ctx, repo.ID)
	if err != nil {
		return err
	}

	var toAdd []*git_model.Branch
	var toUpdate []*git_model.Branch
	var toRemove []int64
	for _, branch := range allBranches {
		var dbb *git_model.Branch
		for _, dbBranch := range dbBranches {
			if branch == dbBranch.Name {
				dbb = dbBranch
				break
			}
		}
		commit, err := gitRepo.GetBranchCommit(branch)
		if err != nil {
			return err
		}
		if dbb == nil {
			toAdd = append(toAdd, &git_model.Branch{
				RepoID:        repo.ID,
				Name:          branch,
				CommitSHA:     commit.ID.String(),
				CommitMessage: commit.CommitMessage,
				PusherID:      doerID,
				CommitTime:    timeutil.TimeStamp(commit.Author.When.Unix()),
			})
		} else if commit.ID.String() != dbb.CommitSHA {
			toUpdate = append(toUpdate, &git_model.Branch{
				ID:            dbb.ID,
				RepoID:        repo.ID,
				Name:          branch,
				CommitSHA:     commit.ID.String(),
				CommitMessage: commit.CommitMessage,
				PusherID:      doerID,
				CommitTime:    timeutil.TimeStamp(commit.Author.When.Unix()),
			})
		}
	}

	for _, dbBranch := range dbBranches {
		var found bool
		for _, branch := range allBranches {
			if branch == dbBranch.Name {
				found = true
				break
			}
		}
		if !found && !dbBranch.IsDeleted {
			toRemove = append(toRemove, dbBranch.ID)
		}
	}

	log.Trace("SyncRepoBranches[%s]: toAdd: %v, toUpdate: %v, toRemove: %v", repo.FullName(), toAdd, toUpdate, toRemove)

	if len(toAdd) == 0 && len(toRemove) == 0 && len(toUpdate) == 0 {
		return nil
	}

	return db.WithTx(ctx, func(subCtx context.Context) error {
		if len(toAdd) > 0 {
			if err := git_model.AddBranches(subCtx, toAdd); err != nil {
				return err
			}
		}

		if len(toUpdate) > 0 {
			for _, b := range toUpdate {
				if _, err := db.GetEngine(subCtx).ID(b.ID).
					Cols("commit_sha, commit_message, pusher_id, commit_time, is_deleted").
					Update(b); err != nil {
					return err
				}
			}
		}

		if len(toRemove) > 0 {
			err = git_model.DeleteBranches(subCtx, repo.ID, doerID, toRemove)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
