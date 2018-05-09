// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"code.gitea.io/gitea/modules/util"

	"github.com/go-xorm/xorm"
)

func addMultipleAssignees(x *xorm.Engine) error {

	// Redeclare issue struct
	type Issue struct {
		ID          int64  `xorm:"pk autoincr"`
		RepoID      int64  `xorm:"INDEX UNIQUE(repo_index)"`
		Index       int64  `xorm:"UNIQUE(repo_index)"` // Index in one repository.
		PosterID    int64  `xorm:"INDEX"`
		Title       string `xorm:"name"`
		Content     string `xorm:"TEXT"`
		MilestoneID int64  `xorm:"INDEX"`
		Priority    int
		AssigneeID  int64 `xorm:"INDEX"`
		IsClosed    bool  `xorm:"INDEX"`
		IsPull      bool  `xorm:"INDEX"` // Indicates whether is a pull request or not.
		NumComments int
		Ref         string

		DeadlineUnix util.TimeStamp `xorm:"INDEX"`
		CreatedUnix  util.TimeStamp `xorm:"INDEX created"`
		UpdatedUnix  util.TimeStamp `xorm:"INDEX updated"`
		ClosedUnix   util.TimeStamp `xorm:"INDEX"`
	}

	allIssues := []Issue{}
	err := x.Find(&allIssues)
	if err != nil {
		return err
	}

	// Create the table
	type IssueAssignees struct {
		ID         int64 `xorm:"pk autoincr"`
		AssigneeID int64 `xorm:"INDEX"`
		IssueID    int64 `xorm:"INDEX"`
	}
	err = x.Sync2(IssueAssignees{})
	if err != nil {
		return err
	}

	// Range over all issues and insert a new entry for each issue/assignee
	sess := x.NewSession()
	defer sess.Close()

	err = sess.Begin()
	if err != nil {
		return err
	}

	for _, issue := range allIssues {
		if issue.AssigneeID != 0 {
			_, err := sess.Insert(IssueAssignees{IssueID: issue.ID, AssigneeID: issue.AssigneeID})
			if err != nil {
				sess.Rollback()
				return err
			}
		}
	}

	// Updated the comment table
	type Comment struct {
		ID              int64 `xorm:"pk autoincr"`
		Type            int
		PosterID        int64 `xorm:"INDEX"`
		IssueID         int64 `xorm:"INDEX"`
		LabelID         int64
		OldMilestoneID  int64
		MilestoneID     int64
		OldAssigneeID   int64
		AssigneeID      int64
		RemovedAssignee bool
		OldTitle        string
		NewTitle        string

		CommitID        int64
		Line            int64
		Content         string `xorm:"TEXT"`
		RenderedContent string `xorm:"-"`

		CreatedUnix util.TimeStamp `xorm:"INDEX created"`
		UpdatedUnix util.TimeStamp `xorm:"INDEX updated"`

		// Reference issue in commit message
		CommitSHA string `xorm:"VARCHAR(40)"`
	}
	if err := x.Sync2(Comment{}); err != nil {
		return err
	}

	// Migrate comments
	// First update everything to not have nulls in db
	if _, err := sess.Where("type = ?", 9).Cols("removed_assignee").Update(Comment{RemovedAssignee: false}); err != nil {
		return err
	}

	allAssignementComments := []Comment{}
	if err := sess.Where("type = ?", 9).Find(&allAssignementComments); err != nil {
		return err
	}

	for _, comment := range allAssignementComments {
		// Everytime where OldAssigneeID is > 0, the assignement was removed.
		if comment.OldAssigneeID > 0 {
			_, err = sess.ID(comment.ID).Update(Comment{RemovedAssignee: true})
		}
	}

	if err := dropTableColumns(sess, "issue", "assignee_id"); err != nil {
		return err
	}

	if err := dropTableColumns(sess, "issue_user", "is_assigned"); err != nil {
		return err
	}
	return sess.Commit()
}
