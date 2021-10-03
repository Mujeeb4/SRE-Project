// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repo

import (
	"net/http"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/convert"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/attachment"
	issue_service "code.gitea.io/gitea/services/issue"
)

/**
 * NOTE about permissions:
 * - repo access is already checked via middleware on the /repos/{owner}/{name} group
 * - issue/pull *read* access is checked on the ../issues group middleware
 *   ("read" access allows posting issues, so posting attachments is fine too!)
 * - setting.Attachment.Enabled is checked on ../assets group middleware
 * All that is left to be checked is
 * - canUserWriteIssueAttachment()
 * - attachmentBelongsToIssue()
 */

// GetIssueAttachment gets a single attachment of the issue
func GetIssueAttachment(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/issues/assets/{attachment_id} issue issueGetIssueAttachment
	// ---
	// summary: Get a issue attachment
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
	// - name: attachment_id
	//   in: path
	//   description: id of the attachment to get
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Attachment"

	attach := getIssueAttachmentSafeRead(ctx)
	if attach == nil {
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAttachment(attach))
}

// ListIssueAttachments lists all attachments of the issue
func ListIssueAttachments(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/issues/{index}/assets issue issueListIssueAttachments
	// ---
	// summary: List issue's attachments
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
	// - name: index
	//   in: path
	//   description: id of the issue
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/AttachmentList"

	issue, err := models.GetIssueByIndex(ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetIssueByIndex", err)
		return
	}
	if err := issue.LoadAttributes(); err != nil {
		ctx.Error(http.StatusInternalServerError, "LoadAttributes", err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAPIIssue(issue).Attachments)
}

// CreateIssueAttachment creates an attachment and saves the given file
func CreateIssueAttachment(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/issues/{index}/assets issue issueCreateIssueAttachment
	// ---
	// summary: Create a issue attachment
	// produces:
	// - application/json
	// consumes:
	// - multipart/form-data
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
	// - name: index
	//   in: path
	//   description: index of the issue
	//   type: integer
	//   format: int64
	//   required: true
	// - name: name
	//   in: query
	//   description: name of the attachment
	//   type: string
	//   required: false
	// - name: attachment
	//   in: formData
	//   description: attachment to upload
	//   type: file
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/Attachment"
	//   "400":
	//     "$ref": "#/responses/error"

	// Check if issue exists and load issue
	issue, err := models.GetIssueByIndex(ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		ctx.NotFoundOrServerError("GetIssueByIndex", models.IsErrIssueNotExist, err)
		return
	}

	if !canUserWriteIssueAttachment(ctx, issue) {
		return
	}

	// Get uploaded file from request
	file, header, err := ctx.Req.FormFile("attachment")
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetFile", err)
		return
	}
	defer file.Close()

	var filename = header.Filename
	if query := ctx.FormString("name"); query != "" {
		filename = query
	}

	attach, err := attachment.UploadAttachment(file, setting.Attachment.AllowedTypes, &models.Attachment{
		Name:       filename,
		UploaderID: ctx.User.ID,
		RepoID:     ctx.Repo.Repository.ID,
		IssueID:    issue.ID,
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "UploadAttachment", err)
		return
	}

	if err != nil {
		ctx.Error(http.StatusInternalServerError, "NewAttachment", err)
		return
	}
	issue.Attachments = append(issue.Attachments, attach)

	if err := issue_service.ChangeContent(issue, ctx.User, issue.Content); err != nil {
		ctx.ServerError("ChangeContent", err)
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToAttachment(attach))
}

// EditIssueAttachment updates the given attachment
func EditIssueAttachment(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/issues/assets/{attachment_id} issue issueEditIssueAttachment
	// ---
	// summary: Edit a issue attachment
	// produces:
	// - application/json
	// consumes:
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
	// - name: attachment_id
	//   in: path
	//   description: id of the attachment to edit
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditAttachmentOptions"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Attachment"

	// get attachment and check permissions
	attach := getIssueAttachmentSafeWrite(ctx)
	if attach == nil {
		return
	}
	// do changes to attachment. only meaningful change is name.
	form := web.GetForm(ctx).(*api.EditAttachmentOptions)
	if form.Name != "" {
		attach.Name = form.Name
	}
	if err := models.UpdateAttachment(attach); err != nil {
		ctx.Error(http.StatusInternalServerError, "UpdateAttachment", attach)
	}
	ctx.JSON(http.StatusCreated, convert.ToAttachment(attach))
}

// DeleteIssueAttachment delete a given attachment
func DeleteIssueAttachment(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/issues/assets/{attachment_id} issue issueDeleteIssueAttachment
	// ---
	// summary: Delete a issue attachment
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
	// - name: attachment_id
	//   in: path
	//   description: id of the attachment to delete
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	attach := getIssueAttachmentSafeWrite(ctx)
	if attach == nil {
		return
	}
	if err := models.DeleteAttachment(attach, true); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteAttachment", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func getIssueAttachmentSafeWrite(ctx *context.APIContext) *models.Attachment {
	attach := getIssueAttachmentSafeRead(ctx)
	if attach == nil {
		return nil
	}
	issue, err := models.GetIssueByID(attach.IssueID)
	if err != nil {
		ctx.NotFoundOrServerError("GetIssueByID", models.IsErrIssueNotExist, err)
		return nil
	}
	if !canUserWriteIssueAttachment(ctx, issue) {
		return nil
	}
	return attach
}

func getIssueAttachmentSafeRead(ctx *context.APIContext) *models.Attachment {
	attachID := ctx.ParamsInt64(":asset")
	attach, err := models.GetAttachmentByID(attachID)
	if err != nil {
		ctx.NotFoundOrServerError("GetAttachmentByID", models.IsErrAttachmentNotExist, err)
		return nil
	}
	if !attachmentBelongsToRepoOrIssue(ctx, attach, nil, nil) {
		return nil
	}
	return attach
}

func canUserWriteIssueAttachment(ctx *context.APIContext, i *models.Issue) (success bool) {
	canEditIssue := ctx.User.ID == i.PosterID && !ctx.IsUserRepoAdmin() && !ctx.IsUserSiteAdmin()
	if !canEditIssue {
		ctx.Error(http.StatusForbidden, "IssueEditPerm", "user should have a permission to editIssue")
		return
	}

	return true
}

func attachmentBelongsToRepoOrIssue(ctx *context.APIContext, a *models.Attachment, issue *models.Issue, comment *models.Comment) (success bool) {
	if a.RepoID != ctx.Repo.Repository.ID {
		log.Debug("Requested attachment[%d] does not belong to repo[%-v].", a.ID, ctx.Repo.Repository)
		ctx.NotFound()
		return
	}
	if a.IssueID == 0 {
		// catch people trying to get release assets ;)
		log.Debug("Requested attachment[%d] is not in an issue.", a.ID)
		ctx.NotFound()
		return
	} else if issue != nil && a.IssueID != issue.ID {
		log.Debug("Requested attachment[%d] does not belong to issue[%d, #%d].", a.ID, issue.ID, issue.Index)
		ctx.NotFound()
		return
	}
	if comment != nil && a.CommentID != comment.ID {
		log.Debug("Requested attachment[%d] does not belong to comment[%d].", a.ID, comment.ID)
		ctx.NotFound()
		return
	}
	return true
}
