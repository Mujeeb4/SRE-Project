// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	project_model "code.gitea.io/gitea/models/project"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/util"
)

// LoadProject load the project the issue was assigned to
func (issue *Issue) LoadProject(ctx context.Context) (err error) {
	if issue.Project == nil {
		var p project_model.Project
		if _, err = db.GetEngine(ctx).Table("project").
			Join("INNER", "project_issue", "project.id=project_issue.project_id").
			Where("project_issue.issue_id = ?", issue.ID).
			Get(&p); err != nil {
			return err
		}
		issue.Project = &p
	}
	return err
}

// ProjectID return project id if issue was assigned to one
func (issue *Issue) ProjectID() int64 {
	return issue.projectID(db.DefaultContext)
}

func (issue *Issue) projectID(ctx context.Context) int64 {
	var ip project_model.ProjectIssue
	has, err := db.GetEngine(ctx).Where("issue_id=?", issue.ID).Get(&ip)
	if err != nil || !has {
		return 0
	}
	return ip.ProjectID
}

// ProjectBoardID return project board id if issue was assigned to one
func (issue *Issue) ProjectBoardID() int64 {
	return issue.projectBoardID(db.DefaultContext)
}

func (issue *Issue) projectBoardID(ctx context.Context) int64 {
	var ip project_model.ProjectIssue
	has, err := db.GetEngine(ctx).Where("issue_id=?", issue.ID).Get(&ip)
	if err != nil || !has {
		return 0
	}
	return ip.ProjectBoardID
}

// LoadIssuesFromBoard load issues assigned to this board
func LoadIssuesFromBoard(ctx context.Context, b *project_model.Board, doer *user_model.User, isClosed util.OptionalBool) (IssueList, error) {
	issueList := make([]*Issue, 0, 10)

	if err := b.LoadProject(ctx); err != nil {
		return nil, err
	}

	if b.ID != 0 {
		issues, err := Issues(ctx, &IssuesOptions{
			ProjectBoardID: b.ID,
			ProjectID:      b.ProjectID,
			SortType:       "project-column-sorting",
			IsClosed:       isClosed,
		})
		if err != nil {
			return nil, err
		}
		for _, issue := range issues {
			if canRetrievedByDoer, err := issue.CanRetrievedByDoer(ctx, b.Project, doer); err != nil {
				return nil, err
			} else if canRetrievedByDoer {
				issueList = append(issueList, issue)
			}
		}
	}

	if b.Default {
		issues, err := Issues(ctx, &IssuesOptions{
			ProjectBoardID: -1, // Issues without ProjectBoardID
			ProjectID:      b.ProjectID,
			SortType:       "project-column-sorting",
			IsClosed:       isClosed,
		})
		if err != nil {
			return nil, err
		}
		for _, issue := range issues {
			if canRetrievedByDoer, err := issue.CanRetrievedByDoer(ctx, b.Project, doer); err != nil {
				return nil, err
			} else if canRetrievedByDoer {
				issueList = append(issueList, issue)
			}
		}
	}

	if err := IssueList(issueList).LoadComments(ctx); err != nil {
		return nil, err
	}

	return issueList, nil
}

// LoadIssuesFromBoardList load issues assigned to the boards
func LoadIssuesFromBoardList(ctx context.Context, bs project_model.BoardList, doer *user_model.User, isClosed util.OptionalBool) (map[int64]IssueList, error) {
	issuesMap := make(map[int64]IssueList, len(bs))
	for _, b := range bs {
		il, err := LoadIssuesFromBoard(ctx, b, doer, isClosed)
		if err != nil {
			return nil, err
		}
		issuesMap[b.ID] = il
	}
	return issuesMap, nil
}

// NumIssuesInProjects returns counter of all issues assigned to a project list which doer can access
func NumIssuesInProjects(ctx context.Context, pl project_model.List, doer *user_model.User, isClosed util.OptionalBool) (int, error) {
	numIssuesInProjects := int(0)
	for _, p := range pl {
		numIssuesInProject, err := NumIssuesInProject(ctx, p, doer, isClosed)
		if err != nil {
			return 0, err
		}
		numIssuesInProjects += numIssuesInProject
	}

	return numIssuesInProjects, nil
}

// NumIssuesInProject returns counter of all issues assigned to a project which doer can access
func NumIssuesInProject(ctx context.Context, p *project_model.Project, doer *user_model.User, isClosed util.OptionalBool) (int, error) {
	numIssuesInProject := int(0)
	bs, err := project_model.GetBoards(ctx, p.ID)
	if err != nil {
		return 0, err
	}
	im, err := LoadIssuesFromBoardList(ctx, bs, doer, isClosed)
	if err != nil {
		return 0, err
	}
	for _, il := range im {
		numIssuesInProject += len(il)
	}
	return numIssuesInProject, nil
}

// ChangeProjectAssign changes the project associated with an issue
func ChangeProjectAssign(issue *Issue, doer *user_model.User, newProjectID int64) error {
	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		return err
	}
	defer committer.Close()

	if err := addUpdateIssueProject(ctx, issue, doer, newProjectID); err != nil {
		return err
	}

	return committer.Commit()
}

func addUpdateIssueProject(ctx context.Context, issue *Issue, doer *user_model.User, newProjectID int64) error {
	oldProjectID := issue.projectID(ctx)

	if err := issue.LoadRepo(ctx); err != nil {
		return err
	}

	// Only check if we add a new project and not remove it.
	if newProjectID > 0 {
		newProject, err := project_model.GetProjectByID(ctx, newProjectID)
		if err != nil {
			return err
		}

		if canWriteByDoer, err := newProject.CanWriteByDoer(ctx, issue.Repo, doer); err != nil {
			return err
		} else if !canWriteByDoer {
			return fmt.Errorf("issue's repository can't be retrieved by doer")
		}
	}

	if _, err := db.GetEngine(ctx).Where("project_issue.issue_id=?", issue.ID).Delete(&project_model.ProjectIssue{}); err != nil {
		return err
	}

	if oldProjectID > 0 || newProjectID > 0 {
		if _, err := CreateComment(ctx, &CreateCommentOptions{
			Type:         CommentTypeProject,
			Doer:         doer,
			Repo:         issue.Repo,
			Issue:        issue,
			OldProjectID: oldProjectID,
			ProjectID:    newProjectID,
		}); err != nil {
			return err
		}
	}

	return db.Insert(ctx, &project_model.ProjectIssue{
		IssueID:   issue.ID,
		ProjectID: newProjectID,
	})
}

// MoveIssueAcrossProjectBoards move a card from one board to another
func MoveIssueAcrossProjectBoards(issue *Issue, board *project_model.Board) error {
	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		return err
	}
	defer committer.Close()
	sess := db.GetEngine(ctx)

	var pis project_model.ProjectIssue
	has, err := sess.Where("issue_id=?", issue.ID).Get(&pis)
	if err != nil {
		return err
	}

	if !has {
		return fmt.Errorf("issue has to be added to a project first")
	}

	pis.ProjectBoardID = board.ID
	if _, err := sess.ID(pis.ID).Cols("project_board_id").Update(&pis); err != nil {
		return err
	}

	return committer.Commit()
}
