// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repo

import (
	"net/http"
	"testing"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/context"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/web"

	"github.com/stretchr/testify/assert"
)

func TestRepoEdit(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := test.MockContext(t, "user2/repo1")
	test.LoadRepo(t, ctx, 1)
	test.LoadUser(t, ctx, 2)
	ctx.Repo.Owner = ctx.Doer
	description := "new description"
	website := "http://wwww.newwebsite.com"
	private := true
	hasIssues := false
	hasWiki := false
	defaultBranch := "master"
	hasPullRequests := true
	ignoreWhitespaceConflicts := true
	allowMerge := false
	allowRebase := false
	allowRebaseMerge := false
	allowSquashMerge := false
	archived := true
	opts := api.EditRepoOption{
		Name:                      &ctx.Repo.Repository.Name,
		Description:               &description,
		Website:                   &website,
		Private:                   &private,
		HasIssues:                 &hasIssues,
		HasWiki:                   &hasWiki,
		DefaultBranch:             &defaultBranch,
		HasPullRequests:           &hasPullRequests,
		IgnoreWhitespaceConflicts: &ignoreWhitespaceConflicts,
		AllowMerge:                &allowMerge,
		AllowRebase:               &allowRebase,
		AllowRebaseMerge:          &allowRebaseMerge,
		AllowSquash:               &allowSquashMerge,
		Archived:                  &archived,
	}

	apiCtx := &context.APIContext{Context: ctx, Org: nil}
	web.SetForm(apiCtx, &opts)
	Edit(apiCtx)

	assert.EqualValues(t, http.StatusOK, ctx.Resp.Status())
	unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
		ID: 1,
	}, unittest.Cond("name = ? AND is_archived = 1", *opts.Name))
}

func TestRepoEditNameChange(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := test.MockContext(t, "user2/repo1")
	test.LoadRepo(t, ctx, 1)
	test.LoadUser(t, ctx, 2)
	ctx.Repo.Owner = ctx.Doer
	name := "newname"
	opts := api.EditRepoOption{
		Name: &name,
	}

	apiCtx := &context.APIContext{Context: ctx, Org: nil}
	web.SetForm(apiCtx, &opts)
	Edit(apiCtx)
	assert.EqualValues(t, http.StatusOK, ctx.Resp.Status())

	unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
		ID: 1,
	}, unittest.Cond("name = ?", opts.Name))
}
