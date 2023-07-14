// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package internal

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/util"
)

// IndexerData data stored in the issue indexer
type IndexerData struct {
	ID     int64 `json:"id"`
	RepoID int64 `json:"repo_id"`

	// Fields used for keyword searching
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Comments []string `json:"comments"`

	// Fields used for filtering
	IsPull             bool               `json:"is_pull"`
	IsClosed           bool               `json:"is_closed"`         // So if the status of an issue has changed, we should reindex the issue.
	Labels             []int64            `json:"labels"`            // So if the labels of an issue have changed, we should reindex the issue.
	NoLabels           bool               `json:"no_labels"`         // True if Labels is empty
	MilestoneIDs       []int64            `json:"milestone_ids"`     // So if the milestones of an issue have changed, we should reindex the issue.
	NoMilestone        bool               `json:"no_milestone"`      // True if Milestones is empty
	ProjectIDs         []int64            `json:"project_ids"`       // So if the projects of an issue have changed, we should reindex the issue.
	ProjectBoardIDs    []int64            `json:"project_board_ids"` // So if the projects of an issue have changed, we should reindex the issue.
	NoProject          bool               `json:"no_project"`        // True if ProjectIDs is empty
	PosterID           int64              `json:"poster_id"`
	AssigneeID         int64              `json:"assignee_id"` // So if the assignee of an issue has changed, we should reindex the issue.
	MentionIDs         []int64            `json:"mention_ids"`
	ReviewedIDs        []int64            `json:"reviewed_ids"`         // So if the reviewers of an issue have changed, we should reindex the issue.
	ReviewRequestedIDs []int64            `json:"review_requested_ids"` // So if the requested reviewers of an issue have changed, we should reindex the issue.
	SubscriberIDs      []int64            `json:"subscriber_ids"`       // So if the subscribers of an issue have changed, we should reindex the issue.
	UpdatedUnix        timeutil.TimeStamp `json:"updated_unix"`

	// Fields used for sorting
	CreatedUnix  timeutil.TimeStamp `json:"created_unix"`
	DueUnix      timeutil.TimeStamp `json:"due_unix"`
	CommentCount int64              `json:"comment_count"`
}

// Match represents on search result
type Match struct {
	ID    int64   `json:"id"`
	Score float64 `json:"score"`
}

// SearchResult represents search results
type SearchResult struct {
	Total int64
	Hits  []Match

	// Imprecise indicates that the result is not accurate, and it needs second filtering and sorting by database.
	// It could be removed when all engines support filtering and sorting.
	Imprecise bool
}

// SearchOptions represents search options
// So the search engine should support:
//   - Filter by boolean/int value
//   - Filter by "array contains any of specified elements"
//   - Filter by "array doesn't contain any of specified elements"
type SearchOptions struct {
	Keyword string // keyword to search

	RepoIDs []int64 // repository IDs which the issues belong to

	IsPull   util.OptionalBool // if the issues is a pull request
	IsClosed util.OptionalBool // if the issues is closed

	IncludedLabelIDs []int64 // labels the issues have
	ExcludedLabelIDs []int64 // labels the issues don't have
	NoLabel          bool    // if the issues have no label, if true, IncludedLabelIDs and ExcludedLabelIDs will be ignored

	MilestoneIDs []int64 // milestones the issues have
	NoMilestone  bool    // if the issues have no milestones, if true, MilestoneIDs will be ignored

	ProjectID      *int64 // project the issues belong to
	ProjectBoardID *int64 // project board the issues belong to

	PosterID *int64 // poster of the issues

	AssigneeID *int64 // assignee of the issues, zero means no assignee

	MentionID *int64 // mentioned user of the issues

	ReviewedID *int64 // reviewer of the issues

	ReviewRequestedID *int64 // requested reviewer of the issues

	SubscriberID *int64 // subscriber of the issues

	UpdatedAfterUnix  *int64
	UpdatedBeforeUnix *int64

	db.Paginator

	SortBy SearchOptionsSortBy // sort by field
}

type SearchOptionsSortBy string

const (
	SearchOptionsSortByCreatedDesc  SearchOptionsSortBy = "-created"
	SearchOptionsSortByUpdatedDesc  SearchOptionsSortBy = "-updated"
	SearchOptionsSortByCommentsDesc SearchOptionsSortBy = "-comments"
	SearchOptionsSortByDueDesc      SearchOptionsSortBy = "-due"
	SearchOptionsSortByCreatedAsc   SearchOptionsSortBy = "created"
	SearchOptionsSortByUpdatedAsc   SearchOptionsSortBy = "updated"
	SearchOptionsSortByCommentsAsc  SearchOptionsSortBy = "comments"
	SearchOptionsSortByDueAsc       SearchOptionsSortBy = "due"
)
