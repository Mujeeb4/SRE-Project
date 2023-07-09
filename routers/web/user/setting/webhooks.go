// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net/http"

	"code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/audit"
)

const (
	tplSettingsHooks base.TplName = "user/settings/hooks"
)

// Webhooks render webhook list page
func Webhooks(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsHooks"] = true
	ctx.Data["BaseLink"] = setting.AppSubURL + "/user/settings/hooks"
	ctx.Data["BaseLinkNew"] = setting.AppSubURL + "/user/settings/hooks"
	ctx.Data["Description"] = ctx.Tr("settings.hooks.desc")

	ws, err := webhook.ListWebhooksByOpts(ctx, &webhook.ListWebhookOptions{OwnerID: ctx.Doer.ID})
	if err != nil {
		ctx.ServerError("ListWebhooksByOpts", err)
		return
	}

	ctx.Data["Webhooks"] = ws
	ctx.HTML(http.StatusOK, tplSettingsHooks)
}

// DeleteWebhook response for delete webhook
func DeleteWebhook(ctx *context.Context) {
	defer ctx.JSON(http.StatusOK, map[string]any{
		"redirect": setting.AppSubURL + "/user/settings/hooks",
	})

	hook, err := webhook.GetWebhookByOwnerID(ctx.Doer.ID, ctx.FormInt64("id"))
	if err != nil {
		ctx.Flash.Error("GetWebhookByOwnerID: " + err.Error())
		return
	}

	if err := webhook.DeleteWebhookByOwnerID(ctx.Doer.ID, ctx.FormInt64("id")); err != nil {
		ctx.Flash.Error("DeleteWebhookByOwnerID: " + err.Error())
	} else {
		audit.Record(audit.UserWebhookRemove, ctx.Doer, ctx.Doer, hook, "Removed webhook %s.", hook.URL)

		ctx.Flash.Success(ctx.Tr("repo.settings.webhook_deletion_success"))
	}
}
