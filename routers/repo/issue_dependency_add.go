// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repo

import (
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
)

// AddDependency adds new dependencies
func AddDependency(c *context.Context) {

	// TODO: should should an issue only have dependencies in it's own repo?

	depID, err := strconv.ParseInt(c.Req.PostForm.Get("newDependency"), 10, 64)
	if err != nil {
		c.Handle(http.StatusBadRequest, "issue ID is not int", err)
		return
	}

	issueIndex := c.ParamsInt64("index")
	issue, err := models.GetIssueByIndex(c.Repo.Repository.ID, issueIndex)
	if err != nil {
		c.Handle(http.StatusInternalServerError, "GetIssueByIndex", err)
		return
	}

	// Check if the Repo is allowed to have dependencies
	if !c.Repo.Repository.UnitEnabled(models.UnitTypeIssueDependencies) {
		c.Handle(404, "MustEnableIssueDependencies", nil)
		return
	}

	// Dependency
	dep, err := models.GetIssueByID(depID)
	if err != nil {
		c.Handle(http.StatusInternalServerError, "GetIssueByID", err)
		return
	}

	// Check if issue and dependency is the same
	if dep.Index == issueIndex {
		c.Flash.Error(c.Tr("issues.dependency.add_error_same_issue"))
	} else {

		exists, depExists, err := models.CreateIssueDependency(c.User, issue, dep)
		if err != nil {
			c.Handle(http.StatusInternalServerError, "CreateOrUpdateIssueDependency", err)
			return
		}

		if !depExists {
			c.Flash.Error(c.Tr("add_error_dep_not_exist"))
		}

		if exists {
			c.Flash.Error(c.Tr("add_error_dep_exists"))
		}
	}

	url := fmt.Sprintf("%s/issues/%d", c.Repo.RepoLink, issueIndex)
	c.Redirect(url, http.StatusSeeOther)
}
