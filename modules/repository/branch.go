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
)

// SyncBranches synchronizes branch table with repository branches
func SyncBranches(ctx context.Context, repo *repo_model.Repository, gitRepo *git.Repository) error {
	log.Debug("SyncBranches: in Repo[%d:%s/%s]", repo.ID, repo.OwnerName, repo.Name)

	const limit = 100
	var allBranches []string
	for page := 1; ; page++ {
		branches, _, err := gitRepo.GetBranchNames(page*limit, limit)
		if err != nil {
			return err
		}
		if len(branches) == 0 {
			break
		}
		allBranches = append(allBranches, branches...)
	}

	dbBranches, err := git_model.LoadAllBranches(ctx, repo.ID)
	if err != nil {
		return err
	}

	var toAdd []string
	var toRemove []int64
	for _, branch := range allBranches {
		var found bool
		for _, dbBranch := range dbBranches {
			if branch == dbBranch.Name {
				found = true
				break
			}
		}
		if !found {
			toAdd = append(toAdd, branch)
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
		if !found {
			toRemove = append(toRemove, dbBranch.ID)
		}
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		if len(toAdd) > 0 {
			err = git_model.AddBranches(ctx, repo.ID, toAdd)
			if err != nil {
				return err
			}
		}

		if len(toRemove) > 0 {
			err = git_model.DeleteBranches(ctx, repo.ID, toRemove)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
