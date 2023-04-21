// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"fmt"
	"net/http"
	"net/url"

	actions_model "code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	actions_shared "code.gitea.io/gitea/routers/web/shared/actions"
)

const (
	// tplSettingsActions template path for render actions settings
	tplSettingsActions base.TplName = "org/settings/actions"
	// tplSettingsRunnersEdit template path for render runners edit settings
	tplSettingsRunnersEdit base.TplName = "org/settings/runners_edit"
)

// Actions render settings/actions page for organization level
func Actions(ctx *context.Context) {
	pageType := ctx.Params(":type")
	if pageType == "runners" {
		ctx.Data["PageIsOrgSettingsRunners"] = true
		ctx.Data["RunnersBaseLink"] = ctx.Link
		page := ctx.FormInt("page")
		if page <= 1 {
			page = 1
		}

		opts := actions_model.FindRunnerOptions{
			ListOptions: db.ListOptions{
				Page:     page,
				PageSize: 100,
			},
			Sort:          ctx.Req.URL.Query().Get("sort"),
			Filter:        ctx.Req.URL.Query().Get("q"),
			OwnerID:       ctx.Org.Organization.ID,
			WithAvailable: true,
		}

		actions_shared.RunnersList(ctx, opts)
	} else if pageType == "secrets" {
		ctx.Data["PageIsOrgSettingsSecrets"] = true
		ctx.Data["SecretsBaseLink"] = ctx.Link
		PrepareSecretsData(ctx)
	} else {
		ctx.ServerError("Unknown Page Type", fmt.Errorf("Unknown Actions Settings Type: %s", pageType))
		return
	}
	ctx.Data["Title"] = ctx.Tr("actions.actions")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageType"] = pageType

	ctx.HTML(http.StatusOK, tplSettingsActions)
}

// ResetRunnerRegistrationToken reset runner registration token
func ResetRunnerRegistrationToken(ctx *context.Context) {
	actions_shared.RunnerResetRegistrationToken(ctx,
		ctx.Org.Organization.ID, 0,
		ctx.Org.OrgLink+"/settings/actions/runners")
}

// RunnersEdit render runner edit page for organization level
func RunnersEdit(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("org.runners.edit")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsOrgSettingsRunners"] = true
	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}

	actions_shared.RunnerDetails(ctx, page,
		ctx.ParamsInt64(":runnerid"), ctx.Org.Organization.ID, 0,
	)

	ctx.HTML(http.StatusOK, tplSettingsRunnersEdit)
}

// RunnersEditPost response for editing runner
func RunnersEditPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("org.runners.edit")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsOrgSettingsRunners"] = true
	actions_shared.RunnerDetailsEditPost(ctx, ctx.ParamsInt64(":runnerid"),
		ctx.Org.Organization.ID, 0,
		ctx.Org.OrgLink+"/settings/actions/runners/"+url.PathEscape(ctx.Params(":runnerid")))
}

// RunnerDeletePost response for deleting runner
func RunnerDeletePost(ctx *context.Context) {
	actions_shared.RunnerDeletePost(ctx,
		ctx.ParamsInt64(":runnerid"),
		ctx.Org.OrgLink+"/settings/actions/runners",
		ctx.Org.OrgLink+"/settings/actions/runners/"+url.PathEscape(ctx.Params(":runnerid")))
}

func RedirectToRunnersSettings(ctx *context.Context) {
	ctx.Redirect(ctx.Org.OrgLink + "/settings/actions/runners")
}
