// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/auth"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	user_setting "code.gitea.io/gitea/routers/web/user/setting"
)

var (
	tplSettingsApplications          base.TplName = "admin/applications/list"
	tplSettingsOauth2ApplicationEdit base.TplName = "admin/applications/oauth2_edit"
)

func newOAuth2CommonHandlers(doer *user_model.User) *user_setting.OAuth2CommonHandlers {
	return &user_setting.OAuth2CommonHandlers{
		Doer:               doer,
		Owner:              nil,
		BasePathList:       fmt.Sprintf("%s/admin/applications", setting.AppSubURL),
		BasePathEditPrefix: fmt.Sprintf("%s/admin/applications/oauth2", setting.AppSubURL),
		TplAppEdit:         tplSettingsOauth2ApplicationEdit,
	}
}

// Applications render org applications page (for org, at the moment, there are only OAuth2 applications)
func Applications(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.applications")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminApplications"] = true

	apps, err := auth.GetOAuth2ApplicationsByUserID(ctx, 0)
	if err != nil {
		ctx.ServerError("GetOAuth2ApplicationsByUserID", err)
		return
	}
	ctx.Data["Applications"] = apps

	ctx.HTML(http.StatusOK, tplSettingsApplications)
}

// ApplicationsPost response for adding an oauth2 application
func ApplicationsPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.applications")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminApplications"] = true

	oa := newOAuth2CommonHandlers(ctx.Doer)
	oa.AddApp(ctx)
}

// EditApplication displays the given application
func EditApplication(ctx *context.Context) {
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminApplications"] = true

	oa := newOAuth2CommonHandlers(ctx.Doer)
	oa.EditShow(ctx)
}

// EditApplicationPost response for editing oauth2 application
func EditApplicationPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.applications")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminApplications"] = true

	oa := newOAuth2CommonHandlers(ctx.Doer)
	oa.EditSave(ctx)
}

// ApplicationsRegenerateSecret handles the post request for regenerating the secret
func ApplicationsRegenerateSecret(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminApplications"] = true

	oa := newOAuth2CommonHandlers(ctx.Doer)
	oa.RegenerateSecret(ctx)
}

// DeleteApplication deletes the given oauth2 application
func DeleteApplication(ctx *context.Context) {
	oa := newOAuth2CommonHandlers(ctx.Doer)
	oa.DeleteApp(ctx)
}

// TODO: revokes the grant with the given id
