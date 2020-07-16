// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"fmt"
	"sort"
	"time"

	"code.gitea.io/gitea/modules/git"

	"xorm.io/xorm"
)

// ActivityAuthorData represents statistical git commit count data
type ActivityAuthorData struct {
	Name       string `json:"name"`
	Login      string `json:"login"`
	AvatarLink string `json:"avatar_link"`
	HomeLink   string `json:"home_link"`
	Commits    int64  `json:"commits"`
}

// ActivityStats represets issue and pull request information.
type ActivityStats struct {
	OpenedPRs                   PullRequestList
	OpenedPRAuthorCount         int64
	MergedPRs                   PullRequestList
	MergedPRAuthorCount         int64
	OpenedIssues                IssueList
	OpenedIssueAuthorCount      int64
	ClosedIssues                IssueList
	ClosedIssueAuthorCount      int64
	UnresolvedIssues            IssueList
	PublishedReleases           []*Release
	PublishedReleaseAuthorCount int64
	Code                        *git.CodeActivityStats
}

// GetActivityStats return stats for repository at given time range
func GetActivityStats(repo *Repository, timeFrom time.Time, releases, issues, prs, code bool) (*ActivityStats, error) {
	stats := &ActivityStats{Code: &git.CodeActivityStats{}}
	if releases {
		if err := stats.FillReleases(repo.ID, timeFrom); err != nil {
			return nil, fmt.Errorf("FillReleases: %v", err)
		}
	}
	if prs {
		if err := stats.FillPullRequests(repo.ID, timeFrom); err != nil {
			return nil, fmt.Errorf("FillPullRequests: %v", err)
		}
	}
	if issues {
		if err := stats.FillIssues(repo.ID, timeFrom); err != nil {
			return nil, fmt.Errorf("FillIssues: %v", err)
		}
	}
	if err := stats.FillUnresolvedIssues(repo.ID, timeFrom, issues, prs); err != nil {
		return nil, fmt.Errorf("FillUnresolvedIssues: %v", err)
	}
	if code {
		gitRepo, err := git.OpenRepository(repo.RepoPath())
		if err != nil {
			return nil, fmt.Errorf("OpenRepository: %v", err)
		}
		defer gitRepo.Close()

		code, err := gitRepo.GetCodeActivityStats(timeFrom, repo.DefaultBranch)
		if err != nil {
			return nil, fmt.Errorf("FillFromGit: %v", err)
		}
		stats.Code = code
	}
	return stats, nil
}

// GetActivityStatsTopAuthors returns top author stats for git commits for all branches
func GetActivityStatsTopAuthors(repo *Repository, timeFrom time.Time, count int) ([]*ActivityAuthorData, error) {
	gitRepo, err := git.OpenRepository(repo.RepoPath())
	if err != nil {
		return nil, fmt.Errorf("OpenRepository: %v", err)
	}
	defer gitRepo.Close()

	code, err := gitRepo.GetCodeActivityStats(timeFrom, "")
	if err != nil {
		return nil, fmt.Errorf("FillFromGit: %v", err)
	}
	if code.Authors == nil {
		return nil, nil
	}
	users := make(map[int64]*ActivityAuthorData)
	var unknownUserID int64
	unknownUserAvatarLink := NewGhostUser().AvatarLink()
	for _, v := range code.Authors {
		if len(v.Email) == 0 {
			continue
		}
		u, err := GetUserByEmail(v.Email)
		if u == nil || IsErrUserNotExist(err) {
			unknownUserID--
			users[unknownUserID] = &ActivityAuthorData{
				Name:       v.Name,
				AvatarLink: unknownUserAvatarLink,
				Commits:    v.Commits,
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		if user, ok := users[u.ID]; !ok {
			users[u.ID] = &ActivityAuthorData{
				Name:       u.DisplayName(),
				Login:      u.LowerName,
				AvatarLink: u.AvatarLink(),
				HomeLink:   u.HomeLink(),
				Commits:    v.Commits,
			}
		} else {
			user.Commits += v.Commits
		}
	}
	v := make([]*ActivityAuthorData, 0)
	for _, u := range users {
		v = append(v, u)
	}

	sort.Slice(v, func(i, j int) bool {
		return v[i].Commits > v[j].Commits
	})

	cnt := count
	if cnt > len(v) {
		cnt = len(v)
	}

	return v[:cnt], nil
}

// ActivePRCount returns total active pull request count
func (stats *ActivityStats) ActivePRCount() int {
	return stats.OpenedPRCount() + stats.MergedPRCount()
}

// OpenedPRCount returns opened pull request count
func (stats *ActivityStats) OpenedPRCount() int {
	return len(stats.OpenedPRs)
}

// OpenedPRPerc returns opened pull request percents from total active
func (stats *ActivityStats) OpenedPRPerc() int {
	return int(float32(stats.OpenedPRCount()) / float32(stats.ActivePRCount()) * 100.0)
}

// MergedPRCount returns merged pull request count
func (stats *ActivityStats) MergedPRCount() int {
	return len(stats.MergedPRs)
}

// MergedPRPerc returns merged pull request percent from total active
func (stats *ActivityStats) MergedPRPerc() int {
	return int(float32(stats.MergedPRCount()) / float32(stats.ActivePRCount()) * 100.0)
}

// ActiveIssueCount returns total active issue count
func (stats *ActivityStats) ActiveIssueCount() int {
	return stats.OpenedIssueCount() + stats.ClosedIssueCount()
}

// OpenedIssueCount returns open issue count
func (stats *ActivityStats) OpenedIssueCount() int {
	return len(stats.OpenedIssues)
}

// OpenedIssuePerc returns open issue count percent from total active
func (stats *ActivityStats) OpenedIssuePerc() int {
	return int(float32(stats.OpenedIssueCount()) / float32(stats.ActiveIssueCount()) * 100.0)
}

// ClosedIssueCount returns closed issue count
func (stats *ActivityStats) ClosedIssueCount() int {
	return len(stats.ClosedIssues)
}

// ClosedIssuePerc returns closed issue count percent from total active
func (stats *ActivityStats) ClosedIssuePerc() int {
	return int(float32(stats.ClosedIssueCount()) / float32(stats.ActiveIssueCount()) * 100.0)
}

// UnresolvedIssueCount returns unresolved issue and pull request count
func (stats *ActivityStats) UnresolvedIssueCount() int {
	return len(stats.UnresolvedIssues)
}

// PublishedReleaseCount returns published release count
func (stats *ActivityStats) PublishedReleaseCount() int {
	return len(stats.PublishedReleases)
}

// FillPullRequests returns pull request information for activity page
func (stats *ActivityStats) FillPullRequests(repoID int64, fromTime time.Time) error {
	var (
		err          error
		count        int64
		rPullRequest = RealTableName("pull_request")
		rIssue       = RealTableName("issue")
	)

	// Merged pull requests
	sess := pullRequestsForActivityStatement(repoID, fromTime, true)
	sess.OrderBy(rPullRequest + ".merged_unix DESC")
	stats.MergedPRs = make(PullRequestList, 0)
	if err = sess.Find(&stats.MergedPRs); err != nil {
		return err
	}
	if err = stats.MergedPRs.LoadAttributes(); err != nil {
		return err
	}

	// Merged pull request authors
	sess = pullRequestsForActivityStatement(repoID, fromTime, true)
	if _, err = sess.Select("count(distinct " + rIssue + ".poster_id) as `count`").
		Table(rPullRequest).
		Get(&count); err != nil {
		return err
	}
	stats.MergedPRAuthorCount = count

	// Opened pull requests
	sess = pullRequestsForActivityStatement(repoID, fromTime, false)
	sess.OrderBy(rIssue + ".created_unix ASC")
	stats.OpenedPRs = make(PullRequestList, 0)
	if err = sess.Find(&stats.OpenedPRs); err != nil {
		return err
	}
	if err = stats.OpenedPRs.LoadAttributes(); err != nil {
		return err
	}

	// Opened pull request authors
	sess = pullRequestsForActivityStatement(repoID, fromTime, false)
	if _, err = sess.Select("count(distinct " + rIssue + ".poster_id) as `count`").
		Table(rPullRequest).Get(&count); err != nil {
		return err
	}
	stats.OpenedPRAuthorCount = count

	return nil
}

func pullRequestsForActivityStatement(repoID int64, fromTime time.Time, merged bool) *xorm.Session {
	var (
		rPullRequest = RealTableName("pull_request")
		rIssue       = RealTableName("issue")
	)
	sess := x.Where(rPullRequest+".base_repo_id=?", repoID).
		Join("INNER", rIssue, rPullRequest+".issue_id = "+rIssue+".id")

	if merged {
		sess.And(rPullRequest+".has_merged = ?", true)
		sess.And(rPullRequest+".merged_unix >= ?", fromTime.Unix())
	} else {
		sess.And(rIssue+".is_closed = ?", false)
		sess.And(rIssue+".created_unix >= ?", fromTime.Unix())
	}

	return sess
}

// FillIssues returns issue information for activity page
func (stats *ActivityStats) FillIssues(repoID int64, fromTime time.Time) error {
	var (
		err    error
		count  int64
		rIssue = RealTableName("issue")
	)

	// Closed issues
	sess := issuesForActivityStatement(repoID, fromTime, true, false)
	sess.OrderBy(rIssue + ".closed_unix DESC")
	stats.ClosedIssues = make(IssueList, 0)
	if err = sess.Find(&stats.ClosedIssues); err != nil {
		return err
	}

	// Closed issue authors
	sess = issuesForActivityStatement(repoID, fromTime, true, false)
	if _, err = sess.Select("count(distinct " + rIssue + ".poster_id) as `count`").
		Table(rIssue).
		Get(&count); err != nil {
		return err
	}
	stats.ClosedIssueAuthorCount = count

	// New issues
	sess = issuesForActivityStatement(repoID, fromTime, false, false)
	sess.OrderBy(rIssue + ".created_unix ASC")
	stats.OpenedIssues = make(IssueList, 0)
	if err = sess.Find(&stats.OpenedIssues); err != nil {
		return err
	}

	// Opened issue authors
	sess = issuesForActivityStatement(repoID, fromTime, false, false)
	if _, err = sess.Select("count(distinct " + rIssue + ".poster_id) as `count`").
		Table(rIssue).
		Get(&count); err != nil {
		return err
	}
	stats.OpenedIssueAuthorCount = count

	return nil
}

// FillUnresolvedIssues returns unresolved issue and pull request information for activity page
func (stats *ActivityStats) FillUnresolvedIssues(repoID int64, fromTime time.Time, issues, prs bool) error {
	// Check if we need to select anything
	if !issues && !prs {
		return nil
	}
	sess := issuesForActivityStatement(repoID, fromTime, false, true)
	if !issues || !prs {
		sess.And(RealTableName("issue")+".is_pull = ?", prs)
	}
	sess.OrderBy(RealTableName("issue") + ".updated_unix DESC")
	stats.UnresolvedIssues = make(IssueList, 0)
	return sess.Find(&stats.UnresolvedIssues)
}

func issuesForActivityStatement(repoID int64, fromTime time.Time, closed, unresolved bool) *xorm.Session {
	var rIssue = RealTableName("issue")
	sess := x.Where(rIssue+".repo_id = ?", repoID).
		And(rIssue+".is_closed = ?", closed)

	if !unresolved {
		sess.And(rIssue+".is_pull = ?", false)
		if closed {
			sess.And(rIssue+".closed_unix >= ?", fromTime.Unix())
		} else {
			sess.And(rIssue+".created_unix >= ?", fromTime.Unix())
		}
	} else {
		sess.And(rIssue+".created_unix < ?", fromTime.Unix())
		sess.And(rIssue+".updated_unix >= ?", fromTime.Unix())
	}

	return sess
}

// FillReleases returns release information for activity page
func (stats *ActivityStats) FillReleases(repoID int64, fromTime time.Time) error {
	var (
		err      error
		count    int64
		rRelease = RealTableName("release")
	)

	// Published releases list
	sess := releasesForActivityStatement(repoID, fromTime)
	sess.OrderBy(rRelease + ".created_unix DESC")
	stats.PublishedReleases = make([]*Release, 0)
	if err = sess.Find(&stats.PublishedReleases); err != nil {
		return err
	}

	// Published releases authors
	sess = releasesForActivityStatement(repoID, fromTime)
	if _, err = sess.Select("count(distinct " + rRelease + ".publisher_id) as `count`").
		Table(rRelease).
		Get(&count); err != nil {
		return err
	}
	stats.PublishedReleaseAuthorCount = count

	return nil
}

func releasesForActivityStatement(repoID int64, fromTime time.Time) *xorm.Session {
	var rRelease = RealTableName("release")
	return x.Where(rRelease+".repo_id = ?", repoID).
		And(rRelease+".is_draft = ?", false).
		And(rRelease+".created_unix >= ?", fromTime.Unix())
}
