// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	auth_model "code.gitea.io/gitea/internal/models/auth"
	"code.gitea.io/gitea/internal/models/db"
	issues_model "code.gitea.io/gitea/internal/models/issues"
	"code.gitea.io/gitea/internal/models/unittest"
	user_model "code.gitea.io/gitea/internal/models/user"
	"code.gitea.io/gitea/internal/modules/git"
	pull_service "code.gitea.io/gitea/internal/services/pull"
	repo_service "code.gitea.io/gitea/internal/services/repository"
	files_service "code.gitea.io/gitea/internal/services/repository/files"

	"github.com/stretchr/testify/assert"
)

func TestAPIPullUpdate(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		// Create PR to test
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		org26 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 26})
		pr := createOutdatedPR(t, user, org26)

		// Test GetDiverging
		diffCount, err := pull_service.GetDiverging(git.DefaultContext, pr)
		assert.NoError(t, err)
		assert.EqualValues(t, 1, diffCount.Behind)
		assert.EqualValues(t, 1, diffCount.Ahead)
		assert.NoError(t, pr.LoadBaseRepo(db.DefaultContext))
		assert.NoError(t, pr.LoadIssue(db.DefaultContext))

		session := loginUser(t, "user2")
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
		req := NewRequestf(t, "POST", "/api/v1/repos/%s/%s/pulls/%d/update?token="+token, pr.BaseRepo.OwnerName, pr.BaseRepo.Name, pr.Issue.Index)
		session.MakeRequest(t, req, http.StatusOK)

		// Test GetDiverging after update
		diffCount, err = pull_service.GetDiverging(git.DefaultContext, pr)
		assert.NoError(t, err)
		assert.EqualValues(t, 0, diffCount.Behind)
		assert.EqualValues(t, 2, diffCount.Ahead)
	})
}

func TestAPIPullUpdateByRebase(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		// Create PR to test
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		org26 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 26})
		pr := createOutdatedPR(t, user, org26)

		// Test GetDiverging
		diffCount, err := pull_service.GetDiverging(git.DefaultContext, pr)
		assert.NoError(t, err)
		assert.EqualValues(t, 1, diffCount.Behind)
		assert.EqualValues(t, 1, diffCount.Ahead)
		assert.NoError(t, pr.LoadBaseRepo(db.DefaultContext))
		assert.NoError(t, pr.LoadIssue(db.DefaultContext))

		session := loginUser(t, "user2")
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
		req := NewRequestf(t, "POST", "/api/v1/repos/%s/%s/pulls/%d/update?style=rebase&token="+token, pr.BaseRepo.OwnerName, pr.BaseRepo.Name, pr.Issue.Index)
		session.MakeRequest(t, req, http.StatusOK)

		// Test GetDiverging after update
		diffCount, err = pull_service.GetDiverging(git.DefaultContext, pr)
		assert.NoError(t, err)
		assert.EqualValues(t, 0, diffCount.Behind)
		assert.EqualValues(t, 1, diffCount.Ahead)
	})
}

func createOutdatedPR(t *testing.T, actor, forkOrg *user_model.User) *issues_model.PullRequest {
	baseRepo, err := repo_service.CreateRepository(db.DefaultContext, actor, actor, repo_service.CreateRepoOptions{
		Name:        "repo-pr-update",
		Description: "repo-tmp-pr-update description",
		AutoInit:    true,
		Gitignores:  "C,C++",
		License:     "MIT",
		Readme:      "Default",
		IsPrivate:   false,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, baseRepo)

	headRepo, err := repo_service.ForkRepository(git.DefaultContext, actor, forkOrg, repo_service.ForkRepoOptions{
		BaseRepo:    baseRepo,
		Name:        "repo-pr-update",
		Description: "desc",
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, headRepo)

	// create a commit on base Repo
	_, err = files_service.ChangeRepoFiles(git.DefaultContext, baseRepo, actor, &files_service.ChangeRepoFilesOptions{
		Files: []*files_service.ChangeRepoFile{
			{
				Operation:     "create",
				TreePath:      "File_A",
				ContentReader: strings.NewReader("File A"),
			},
		},
		Message:   "Add File A",
		OldBranch: "master",
		NewBranch: "master",
		Author: &files_service.IdentityOptions{
			Name:  actor.Name,
			Email: actor.Email,
		},
		Committer: &files_service.IdentityOptions{
			Name:  actor.Name,
			Email: actor.Email,
		},
		Dates: &files_service.CommitDateOptions{
			Author:    time.Now(),
			Committer: time.Now(),
		},
	})
	assert.NoError(t, err)

	// create a commit on head Repo
	_, err = files_service.ChangeRepoFiles(git.DefaultContext, headRepo, actor, &files_service.ChangeRepoFilesOptions{
		Files: []*files_service.ChangeRepoFile{
			{
				Operation:     "create",
				TreePath:      "File_B",
				ContentReader: strings.NewReader("File B"),
			},
		},
		Message:   "Add File on PR branch",
		OldBranch: "master",
		NewBranch: "newBranch",
		Author: &files_service.IdentityOptions{
			Name:  actor.Name,
			Email: actor.Email,
		},
		Committer: &files_service.IdentityOptions{
			Name:  actor.Name,
			Email: actor.Email,
		},
		Dates: &files_service.CommitDateOptions{
			Author:    time.Now(),
			Committer: time.Now(),
		},
	})
	assert.NoError(t, err)

	// create Pull
	pullIssue := &issues_model.Issue{
		RepoID:   baseRepo.ID,
		Title:    "Test Pull -to-update-",
		PosterID: actor.ID,
		Poster:   actor,
		IsPull:   true,
	}
	pullRequest := &issues_model.PullRequest{
		HeadRepoID: headRepo.ID,
		BaseRepoID: baseRepo.ID,
		HeadBranch: "newBranch",
		BaseBranch: "master",
		HeadRepo:   headRepo,
		BaseRepo:   baseRepo,
		Type:       issues_model.PullRequestGitea,
	}
	err = pull_service.NewPullRequest(git.DefaultContext, baseRepo, pullIssue, nil, nil, pullRequest, nil)
	assert.NoError(t, err)

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{Title: "Test Pull -to-update-"})
	pr, err := issues_model.GetPullRequestByIssueID(db.DefaultContext, issue.ID)
	assert.NoError(t, err)

	return pr
}
