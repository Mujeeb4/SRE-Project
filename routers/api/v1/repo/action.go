// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"net/http"

	actions_model "code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	secret_model "code.gitea.io/gitea/models/secret"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/modules/actions"
	"code.gitea.io/gitea/modules/git"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v1/shared"
	"code.gitea.io/gitea/routers/api/v1/utils"
	actions_service "code.gitea.io/gitea/services/actions"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
	secret_service "code.gitea.io/gitea/services/secrets"

	"github.com/nektos/act/pkg/jobparser"
	"github.com/nektos/act/pkg/model"
)

// ListActionsSecrets list an repo's actions secrets
func (Action) ListActionsSecrets(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/secrets repository repoListActionsSecrets
	// ---
	// summary: List an repo's actions secrets
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repository
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/SecretList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	repo := ctx.Repo.Repository

	opts := &secret_model.FindSecretsOptions{
		RepoID:      repo.ID,
		ListOptions: utils.GetListOptions(ctx),
	}

	secrets, count, err := db.FindAndCount[secret_model.Secret](ctx, opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	apiSecrets := make([]*api.Secret, len(secrets))
	for k, v := range secrets {
		apiSecrets[k] = &api.Secret{
			Name:    v.Name,
			Created: v.CreatedUnix.AsTime(),
		}
	}

	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, apiSecrets)
}

// create or update one secret of the repository
func (Action) CreateOrUpdateSecret(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/actions/secrets/{secretname} repository updateRepoSecret
	// ---
	// summary: Create or Update a secret value in a repository
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repository
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: secretname
	//   in: path
	//   description: name of the secret
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateOrUpdateSecretOption"
	// responses:
	//   "201":
	//     description: response when creating a secret
	//   "204":
	//     description: response when updating a secret
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	repo := ctx.Repo.Repository

	opt := web.GetForm(ctx).(*api.CreateOrUpdateSecretOption)

	_, created, err := secret_service.CreateOrUpdateSecret(ctx, 0, repo.ID, ctx.PathParam("secretname"), opt.Data)
	if err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			ctx.Error(http.StatusBadRequest, "CreateOrUpdateSecret", err)
		} else if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "CreateOrUpdateSecret", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "CreateOrUpdateSecret", err)
		}
		return
	}

	if created {
		ctx.Status(http.StatusCreated)
	} else {
		ctx.Status(http.StatusNoContent)
	}
}

// DeleteSecret delete one secret of the repository
func (Action) DeleteSecret(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/actions/secrets/{secretname} repository deleteRepoSecret
	// ---
	// summary: Delete a secret in a repository
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repository
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: secretname
	//   in: path
	//   description: name of the secret
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     description: delete one secret of the organization
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	repo := ctx.Repo.Repository

	err := secret_service.DeleteSecretByName(ctx, 0, repo.ID, ctx.PathParam("secretname"))
	if err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			ctx.Error(http.StatusBadRequest, "DeleteSecret", err)
		} else if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "DeleteSecret", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "DeleteSecret", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetVariable get a repo-level variable
func (Action) GetVariable(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/variables/{variablename} repository getRepoVariable
	// ---
	// summary: Get a repo-level variable
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: name of the owner
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: variablename
	//   in: path
	//   description: name of the variable
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//			"$ref": "#/responses/ActionVariable"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"
	v, err := actions_service.GetVariable(ctx, actions_model.FindVariablesOpts{
		RepoID: ctx.Repo.Repository.ID,
		Name:   ctx.PathParam("variablename"),
	})
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "GetVariable", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetVariable", err)
		}
		return
	}

	variable := &api.ActionVariable{
		OwnerID: v.OwnerID,
		RepoID:  v.RepoID,
		Name:    v.Name,
		Data:    v.Data,
	}

	ctx.JSON(http.StatusOK, variable)
}

// DeleteVariable delete a repo-level variable
func (Action) DeleteVariable(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/actions/variables/{variablename} repository deleteRepoVariable
	// ---
	// summary: Delete a repo-level variable
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: name of the owner
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: variablename
	//   in: path
	//   description: name of the variable
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//			"$ref": "#/responses/ActionVariable"
	//   "201":
	//     description: response when deleting a variable
	//   "204":
	//     description: response when deleting a variable
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	if err := actions_service.DeleteVariableByName(ctx, 0, ctx.Repo.Repository.ID, ctx.PathParam("variablename")); err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			ctx.Error(http.StatusBadRequest, "DeleteVariableByName", err)
		} else if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "DeleteVariableByName", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "DeleteVariableByName", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// CreateVariable create a repo-level variable
func (Action) CreateVariable(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/actions/variables/{variablename} repository createRepoVariable
	// ---
	// summary: Create a repo-level variable
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: name of the owner
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: variablename
	//   in: path
	//   description: name of the variable
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateVariableOption"
	// responses:
	//   "201":
	//     description: response when creating a repo-level variable
	//   "204":
	//     description: response when creating a repo-level variable
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	opt := web.GetForm(ctx).(*api.CreateVariableOption)

	repoID := ctx.Repo.Repository.ID
	variableName := ctx.PathParam("variablename")

	v, err := actions_service.GetVariable(ctx, actions_model.FindVariablesOpts{
		RepoID: repoID,
		Name:   variableName,
	})
	if err != nil && !errors.Is(err, util.ErrNotExist) {
		ctx.Error(http.StatusInternalServerError, "GetVariable", err)
		return
	}
	if v != nil && v.ID > 0 {
		ctx.Error(http.StatusConflict, "VariableNameAlreadyExists", util.NewAlreadyExistErrorf("variable name %s already exists", variableName))
		return
	}

	if _, err := actions_service.CreateVariable(ctx, 0, repoID, variableName, opt.Value); err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			ctx.Error(http.StatusBadRequest, "CreateVariable", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "CreateVariable", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// UpdateVariable update a repo-level variable
func (Action) UpdateVariable(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/actions/variables/{variablename} repository updateRepoVariable
	// ---
	// summary: Update a repo-level variable
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: name of the owner
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: variablename
	//   in: path
	//   description: name of the variable
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/UpdateVariableOption"
	// responses:
	//   "201":
	//     description: response when updating a repo-level variable
	//   "204":
	//     description: response when updating a repo-level variable
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	opt := web.GetForm(ctx).(*api.UpdateVariableOption)

	v, err := actions_service.GetVariable(ctx, actions_model.FindVariablesOpts{
		RepoID: ctx.Repo.Repository.ID,
		Name:   ctx.PathParam("variablename"),
	})
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "GetVariable", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetVariable", err)
		}
		return
	}

	if opt.Name == "" {
		opt.Name = ctx.PathParam("variablename")
	}
	if _, err := actions_service.UpdateVariable(ctx, v.ID, opt.Name, opt.Value); err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			ctx.Error(http.StatusBadRequest, "UpdateVariable", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "UpdateVariable", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ListVariables list repo-level variables
func (Action) ListVariables(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/variables repository getRepoVariablesList
	// ---
	// summary: Get repo-level variables list
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: name of the owner
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//		 "$ref": "#/responses/VariableList"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	vars, count, err := db.FindAndCount[actions_model.ActionVariable](ctx, &actions_model.FindVariablesOpts{
		RepoID:      ctx.Repo.Repository.ID,
		ListOptions: utils.GetListOptions(ctx),
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FindVariables", err)
		return
	}

	variables := make([]*api.ActionVariable, len(vars))
	for i, v := range vars {
		variables[i] = &api.ActionVariable{
			OwnerID: v.OwnerID,
			RepoID:  v.RepoID,
			Name:    v.Name,
		}
	}

	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, variables)
}

// GetRegistrationToken returns the token to register repo runners
func (Action) GetRegistrationToken(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/runners/registration-token repository repoGetRunnerRegistrationToken
	// ---
	// summary: Get a repository's actions runner registration token
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/RegistrationToken"

	shared.GetRegistrationToken(ctx, 0, ctx.Repo.Repository.ID)
}

var _ actions_service.API = new(Action)

// Action implements actions_service.API
type Action struct{}

// NewAction creates a new Action service
func NewAction() actions_service.API {
	return Action{}
}

// ListActionTasks list all the actions of a repository
func ListActionTasks(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/tasks repository ListActionTasks
	// ---
	// summary: List a repository's action tasks
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results, default maximum page size is 50
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/TasksList"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/conflict"
	//   "422":
	//     "$ref": "#/responses/validationError"

	tasks, total, err := db.FindAndCount[actions_model.ActionTask](ctx, &actions_model.FindTaskOptions{
		ListOptions: utils.GetListOptions(ctx),
		RepoID:      ctx.Repo.Repository.ID,
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListActionTasks", err)
		return
	}

	res := new(api.ActionTaskResponse)
	res.TotalCount = total

	res.Entries = make([]*api.ActionTask, len(tasks))
	for i := range tasks {
		convertedTask, err := convert.ToActionTask(ctx, tasks[i])
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "ToActionTask", err)
			return
		}
		res.Entries[i] = convertedTask
	}

	ctx.JSON(http.StatusOK, &res)
}

// ActionWorkflow implements actions_service.WorkflowAPI
type ActionWorkflow struct{}

// NewActionWorkflow creates a new ActionWorkflow service
func NewActionWorkflow() actions_service.WorkflowAPI {
	return ActionWorkflow{}
}

func (a ActionWorkflow) ListRepositoryWorkflows(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/workflows repository ListRepositoryWorkflows
	// ---
	// summary: List repository workflows
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ActionWorkflowList"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/conflict"
	//   "422":
	//     "$ref": "#/responses/validationError"
	//   "500":
	//     "$ref": "#/responses/error"

	workflows, err := actions_service.ListActionWorkflows(ctx)
	if err != nil {
		return
	}

	if len(workflows) == 0 {
		ctx.JSON(http.StatusNotFound, nil)
	}

	ctx.SetTotalCountHeader(int64(len(workflows)))
	ctx.JSON(http.StatusOK, workflows)
}

func (a ActionWorkflow) GetWorkflow(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/workflows/{workflow_id} repository GetWorkflow
	// ---
	// summary: Get a workflow
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: workflow_id
	//   in: path
	//   description: id of the workflow
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ActionWorkflow"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/conflict"
	//   "422":
	//     "$ref": "#/responses/validationError"
	//   "500":
	//     "$ref": "#/responses/error"

	workflowID := ctx.PathParam("workflow_id")
	if len(workflowID) == 0 {
		ctx.Error(http.StatusUnprocessableEntity, "MissingWorkflowParameter", util.NewInvalidArgumentErrorf("workflow_id is required parameter"))
		return
	}

	workflow, err := actions_service.GetActionWorkflow(ctx, workflowID)
	if err != nil {
		return
	}

	if workflow == nil {
		ctx.JSON(http.StatusNotFound, nil)
	}

	ctx.JSON(http.StatusOK, workflow)
}

func (a ActionWorkflow) DisableWorkflow(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/actions/workflows/{workflow_id}/disable repository DisableWorkflow
	// ---
	// summary: Disable a workflow
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: workflow_id
	//   in: path
	//   description: id of the workflow
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     description: No Content
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/conflict"
	//   "422":
	//     "$ref": "#/responses/validationError"

	workflowID := ctx.PathParam("workflow_id")
	if len(workflowID) == 0 {
		ctx.Error(http.StatusUnprocessableEntity, "MissingWorkflowParameter", util.NewInvalidArgumentErrorf("workflow_id is required parameter"))
		return
	}

	err := actions_service.DisableActionWorkflow(ctx, workflowID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "DisableActionWorkflow", err)
	}

	ctx.Status(http.StatusNoContent)
}

func (a ActionWorkflow) DispatchWorkflow(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/actions/workflows/{workflow_id}/dispatches repository DispatchWorkflow
	// ---
	// summary: Create a workflow dispatch event
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: workflow_id
	//   in: path
	//   description: id of the workflow
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateActionWorkflowDispatch"
	// responses:
	//   "204":
	//     description: No Content
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/conflict"
	//   "422":
	//     "$ref": "#/responses/validationError"

	opt := web.GetForm(ctx).(*api.CreateActionWorkflowDispatch)

	workflowID := ctx.PathParam("workflow_id")
	if len(workflowID) == 0 {
		ctx.Error(http.StatusUnprocessableEntity, "MissingWorkflowParameter", util.NewInvalidArgumentErrorf("workflow_id is required parameter"))
		return
	}

	ref := opt.Ref
	if len(ref) == 0 {
		ctx.Error(http.StatusUnprocessableEntity, "MissingWorkflowParameter", util.NewInvalidArgumentErrorf("ref is required parameter"))
		return
	}

	actions_service.DispatchActionWorkflow(ctx, workflowID, opt)

	ctx.Status(http.StatusNoContent)
}

func (a ActionWorkflow) EnableWorkflow(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/actions/workflows/{workflow_id}/enable repository EnableWorkflow
	// ---
	// summary: Enable a workflow
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: workflow_id
	//   in: path
	//   description: id of the workflow
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     description: No Content
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/conflict"
	//   "422":
	//     "$ref": "#/responses/validationError"

	workflowID := ctx.PathParam("workflow_id")
	if len(workflowID) == 0 {
		ctx.Error(http.StatusUnprocessableEntity, "MissingWorkflowParameter", util.NewInvalidArgumentErrorf("workflow_id is required parameter"))
		return
	}

	err := actions_service.EnableActionWorkflow(ctx, workflowID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "EnableActionWorkflow", err)
	}

	ctx.Status(http.StatusNoContent)
}
