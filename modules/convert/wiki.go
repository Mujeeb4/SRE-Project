// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package convert

import (
	"time"

	"code.gitea.io/gitea/modules/git"
	api "code.gitea.io/gitea/modules/structs"
)

// ToWikiCommit convert a git commit into a WikiCommit
func ToWikiCommit(commit *git.Commit) *api.WikiCommit {
	return &api.WikiCommit{
		ID: commit.ID.String(),
		Author: &api.CommitUser{
			Identity: api.Identity{
				Name:  commit.Author.Name,
				Email: commit.Author.Email,
			},
			Date: commit.Author.When.Format(time.RFC3339),
		},
		Committer: &api.CommitUser{
			Identity: api.Identity{
				Name:  commit.Committer.Name,
				Email: commit.Committer.Email,
			},
			Date: commit.Committer.When.Format(time.RFC3339),
		},
		Message: commit.CommitMessage,
	}
}

// ToWikiCommitList convert a list of git commits into a WikiCommitList
func ToWikiCommitList(commits []*git.Commit, count int64) *api.WikiCommitList {
	result := make([]*api.WikiCommit, len(commits))
	for i := range commits {
		result[i] = ToWikiCommit(commits[i])
	}
	return &api.WikiCommitList{
		WikiCommits: result,
		Count:       count,
	}
}
