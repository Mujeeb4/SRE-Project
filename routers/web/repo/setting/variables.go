// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"errors"
	"net/http"

	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	shared "code.gitea.io/gitea/routers/web/shared/variables"
)

const (
	// TODO: Separate from runners when layout is ready
	tplRepoVariables base.TplName = "repo/settings/actions"
	tplOrgVariables  base.TplName = "org/settings/actions"
)

type variablesCtx struct {
	OwnerID           int64
	RepoID            int64
	IsRepo            bool
	IsOrg             bool
	VariablesTemplate base.TplName
	RedirectLink      string
}

func getVariablesCtx(ctx *context.Context) (*variablesCtx, error) {
	if ctx.Data["PageIsRepoSettings"] == true {
		return &variablesCtx{
			RepoID:            ctx.Repo.Repository.ID,
			IsRepo:            true,
			VariablesTemplate: tplRepoVariables,
			RedirectLink:      ctx.Repo.RepoLink + "/settings/actions/variables",
		}, nil
	}

	if ctx.Data["PageIsOrgSettings"] == true {
		return &variablesCtx{
			OwnerID:           ctx.ContextUser.ID,
			IsOrg:             true,
			VariablesTemplate: tplOrgVariables,
			RedirectLink:      ctx.Org.OrgLink + "/settings/actions/variables",
		}, nil
	}

	return nil, errors.New("unable to set Variables context")
}

func Variables(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("actions.variables")
	ctx.Data["PageType"] = "variables"
	ctx.Data["PageIsSharedSettingsVariables"] = true

	vCtx, err := getVariablesCtx(ctx)
	if err != nil {
		ctx.ServerError("getVariablesCtx", err)
		return
	}

	shared.SetVariablesContext(ctx, vCtx.OwnerID, vCtx.RepoID)
	if ctx.Written() {
		return
	}

	ctx.HTML(http.StatusOK, vCtx.VariablesTemplate)
}

func VariableDelete(ctx *context.Context) {
	vCtx, err := getVariablesCtx(ctx)
	if err != nil {
		ctx.ServerError("getVariablesCtx", err)
		return
	}
	shared.DeleteVariable(ctx, vCtx.OwnerID, vCtx.RepoID, vCtx.RedirectLink)
}

func VariableCreate(ctx *context.Context) {
	vCtx, err := getVariablesCtx(ctx)
	if err != nil {
		ctx.ServerError("getVariablesCtx", err)
		return
	}
	shared.CreateVariable(ctx, vCtx.OwnerID, vCtx.RepoID, vCtx.RedirectLink)
}

func VariableUpdate(ctx *context.Context) {
	vCtx, err := getVariablesCtx(ctx)
	if err != nil {
		ctx.ServerError("getVariablesCtx", err)
		return
	}
	shared.UpdateVariable(ctx, vCtx.OwnerID, vCtx.RepoID, vCtx.RedirectLink)
}

func VariableByID(ctx *context.Context) {
	shared.GetVariable(ctx)
}
