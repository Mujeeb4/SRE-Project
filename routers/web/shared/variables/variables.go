// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package varibales

import (
	"net/http"
	"strings"

	"code.gitea.io/gitea/models/db"
	variable_model "code.gitea.io/gitea/models/variable"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
)

func SetVariablesContext(ctx *context.Context, ownerID, repoID int64) {
	variables, err := variable_model.FindVariables(ctx, variable_model.FindVariablesOpts{
		OwnerID: ownerID,
		RepoID:  repoID,
	})
	if err != nil {
		ctx.ServerError("FindVariables", err)
		return
	}
	ctx.Data["Variables"] = variables
}

func DeleteVariable(ctx *context.Context, ownerID, repoID int64, redirectURL string) {
	id := ctx.ParamsInt64(":variableID")

	if _, err := db.DeleteByBean(ctx, &variable_model.Variable{
		ID:      id,
		OwnerID: ownerID,
		RepoID:  repoID,
	}); err != nil {
		log.Error("Delete variable [%d] failed: %v", id, err)
		ctx.Flash.Error(ctx.Tr("actions.variables.deletion.failed"))
	} else {
		ctx.Flash.Success(ctx.Tr("actions.variables.deletion.success"))
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": redirectURL,
	})
}

func CreateVariable(ctx *context.Context, ownerID, repoID int64, redirectURL string) {
	form := web.GetForm(ctx).(*forms.EditVariableForm)

	v, err := variable_model.InsertVariable(ctx, ownerID, repoID, form.Name, form.Data)
	if err != nil {
		log.Error("InsertVariable error: %v", err)
		ctx.Flash.Error(ctx.Tr("actions.variables.creation.failed"))
	} else {
		ctx.Flash.Success(ctx.Tr("actions.variables.creation.success", v.Name))
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": redirectURL,
	})
}

func GetVariable(ctx *context.Context) {
	id := ctx.ParamsInt64(":variableID")

	v, err := variable_model.GetVariableByID(ctx, id)
	if err != nil {
		log.Error("GetVariableByID error: %v", err)
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err:     err.Error(),
			UserMsg: ctx.Tr("actions.variables.id_not_exist", id),
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"id":   v.ID,
		"name": v.Name,
		"data": v.Data,
	})
}

func UpdateVariable(ctx *context.Context, ownerID, repoID int64, redirectURL string) {
	id := ctx.ParamsInt64(":variableID")
	form := web.GetForm(ctx).(*forms.EditVariableForm)
	ok, err := variable_model.UpdateVariable(ctx, &variable_model.Variable{
		ID:      id,
		OwnerID: ownerID,
		RepoID:  repoID,
		Name:    strings.ToUpper(form.Name),
		Data:    form.Data,
	})
	if err != nil || !ok {
		log.Error("UpdateVariable error: %v", err)
		ctx.Flash.Error(ctx.Tr("actions.variables.update.failed"))
	} else {
		ctx.Flash.Success(ctx.Tr("actions.variables.update.success"))
	}
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": redirectURL,
	})
}
