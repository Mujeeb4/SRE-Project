// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package builds

import (
	"net/http"

	bots_model "code.gitea.io/gitea/models/bots"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/bots"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/convert"
	"code.gitea.io/gitea/modules/util"
)

const (
	tplListBuilds base.TplName = "repo/builds/list"
	tplViewBuild  base.TplName = "repo/builds/view"
)

// MustEnableBuilds check if builds are enabled in settings
func MustEnableBuilds(ctx *context.Context) {
	if unit.TypeBuilds.UnitGlobalDisabled() {
		ctx.NotFound("EnableTypeBuilds", nil)
		return
	}

	if ctx.Repo.Repository != nil {
		if !ctx.Repo.CanRead(unit.TypeBuilds) {
			ctx.NotFound("MustEnableBuilds", nil)
			return
		}
	}
}

func List(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.builds")
	ctx.Data["PageIsBuildList"] = true

	defaultBranch, err := ctx.Repo.GitRepo.GetDefaultBranch()
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}
	commit, err := ctx.Repo.GitRepo.GetBranchCommit(defaultBranch)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	workflows, err := bots.ListWorkflows(commit)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["workflows"] = workflows

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	opts := bots_model.FindRunOptions{
		ListOptions: db.ListOptions{
			Page:     page,
			PageSize: convert.ToCorrectPageSize(ctx.FormInt("limit")),
		},
		RepoID: ctx.Repo.Repository.ID,
	}
	if ctx.FormString("state") == "closed" {
		opts.IsClosed = util.OptionalBoolTrue
	} else {
		opts.IsClosed = util.OptionalBoolFalse
	}
	runs, total, err := bots_model.FindRuns(ctx, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	for _, run := range runs {
		run.Repo = ctx.Repo.Repository
	}

	if err := runs.LoadTriggerUser(); err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Data["Builds"] = runs

	pager := context.NewPagination(int(total), opts.PageSize, opts.Page, 5)
	pager.SetDefaultParams(ctx)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplListBuilds)
}
