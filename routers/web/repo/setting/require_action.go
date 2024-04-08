// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// WIP RequireAction

package setting

import (
    "errors"
    "net/http"

    "code.gitea.io/gitea/models/db"
    "code.gitea.io/gitea/modules/base"
    // "code.gitea.io/gitea/modules/log"

    "code.gitea.io/gitea/services/context"

    //"code.gitea.io/gitea/modules/setting"
    shared "code.gitea.io/gitea/routers/web/shared/actions"
    actions_model "code.gitea.io/gitea/models/actions"
)

const (
        // let start with org WIP
        tplOrgRequireAction base.TplName = "org/settings/actions"
)

type requireActionsCtx struct {
    OrgID                  int64
    IsOrg                  bool
    RequireActionTemplate  base.TplName
    RedirectLink           string
}

func getRequireActionCtx(ctx *context.Context) (*requireActionsCtx, error) {
    if ctx.Data["PageIsOrgSettings"] == true {
        return &requireActionsCtx{
            OrgID:                  ctx.Org.Organization.ID,
            IsOrg:                  true,
            RequireActionTemplate:  tplOrgRequireAction,
            RedirectLink:           ctx.Org.OrgLink + "/settings/actions/require_action",
        }, nil
    }
    return nil, errors.New("unable to set Require Actions context")
}

// Listing all RequireAction
func RequireAction(ctx *context.Context) {
    ctx.Data["ActionsTitle"] = ctx.Tr("actions.requires")
    ctx.Data["PageType"] = "require_action"
    ctx.Data["PageIsSharedSettingsRequireAction"] = true

    vCtx, err := getRequireActionCtx(ctx)
    if err != nil {
        ctx.ServerError("getRequireActionCtx", err)
        return
    }

    page := ctx.FormInt("page")
    if page <= 1 { page = 1 }
    opts := actions_model.FindRequireActionOptions{
        ListOptions: db.ListOptions{
            Page:     page,
            PageSize: 10,
        },
    }
    shared.SetRequireActionContext(ctx, opts)
    ctx.Data["Link"] = vCtx.RedirectLink
    shared.GlobalEnableWorkflow(ctx, ctx.Org.Organization.ID)
    ctx.HTML(http.StatusOK, vCtx.RequireActionTemplate)
}

func RequireActionCreate(ctx *context.Context) {
    vCtx, err := getRequireActionCtx(ctx)
    if err != nil {
        ctx.ServerError("getRequireActionCtx", err)
        return
    }
    shared.CreateRequireAction(ctx, vCtx.OrgID, vCtx.RedirectLink)
}
