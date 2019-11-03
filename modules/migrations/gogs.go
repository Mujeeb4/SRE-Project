// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/migrations/base"
	"code.gitea.io/gitea/modules/structs"

	"github.com/gogs/go-gogs-client"
)

var (
	_ base.Downloader        = &GogsDownloader{}
	_ base.DownloaderFactory = &GogsDownloaderFactory{}
)

func init() {
	RegisterDownloaderFactory(&GogsDownloaderFactory{})
}

// GogsDownloaderFactory defines a gogs downloader factory
type GogsDownloaderFactory struct {
}

// Match returns ture if the migration remote URL matched this downloader factory
func (f *GogsDownloaderFactory) Match(opts base.MigrateOptions) (bool, error) {
	if opts.GitServiceType == structs.GogsService {
		return true, nil
	}
	return false, nil
}

// New returns a Downloader related to this factory according MigrateOptions
func (f *GogsDownloaderFactory) New(opts base.MigrateOptions) (base.Downloader, error) {
	u, err := url.Parse(opts.CloneAddr)
	if err != nil {
		return nil, err
	}

	baseURL := u.Scheme + "://" + u.Host
	fields := strings.Split(u.Path, "/")
	oldOwner := fields[1]
	oldName := strings.TrimSuffix(fields[2], ".git")

	log.Trace("Create gogs downloader: %s/%s", oldOwner, oldName)

	return NewGogsDownloader(baseURL, opts.AuthUsername, opts.AuthPassword, oldOwner, oldName), nil
}

// GitServiceType returns the type of git service
func (f *GogsDownloaderFactory) GitServiceType() structs.GitServiceType {
	return structs.GogsService
}

// GogsDownloader implements a Downloader interface to get repository informations
// from gogs via API
type GogsDownloader struct {
	client    *gogs.Client
	baseURL   string
	repoOwner string
	repoName  string
	userName  string
	password  string
}

// NewGogsDownloader creates a gogs Downloader via gogs API
func NewGogsDownloader(baseURL, userName, password, repoOwner, repoName string) *GogsDownloader {
	var downloader = GogsDownloader{
		baseURL:   baseURL,
		userName:  userName,
		password:  password,
		repoOwner: repoOwner,
		repoName:  repoName,
	}

	var client *gogs.Client
	if userName != "" {
		if password == "" {
			client = gogs.NewClient(baseURL, userName)
		} else {
			client = gogs.NewClient(baseURL, "")
			client.SetHTTPClient(&http.Client{
				Transport: &http.Transport{
					Proxy: func(req *http.Request) (*url.URL, error) {
						req.SetBasicAuth(userName, password)
						return nil, nil
					},
				},
			})
		}
	}
	downloader.client = client
	return &downloader
}

// GetRepoInfo returns a repository information
func (g *GogsDownloader) GetRepoInfo() (*base.Repository, error) {
	gr, err := g.client.GetRepo(g.repoOwner, g.repoName)
	if err != nil {
		return nil, err
	}

	// convert github repo to stand Repo
	return &base.Repository{
		Owner:       g.repoOwner,
		Name:        g.repoName,
		IsPrivate:   gr.Private,
		Description: gr.Description,
		CloneURL:    gr.CloneURL,
	}, nil
}

// GetTopics return github topics
func (g *GogsDownloader) GetTopics() ([]string, error) {
	return []string{}, nil
}

// GetMilestones returns milestones
func (g *GogsDownloader) GetMilestones() ([]*base.Milestone, error) {
	var perPage = 100
	var milestones = make([]*base.Milestone, 0, perPage)

	ms, err := g.client.ListRepoMilestones(g.repoOwner, g.repoName)
	if err != nil {
		return nil, err
	}

	t := time.Now()

	for _, m := range ms {
		milestones = append(milestones, &base.Milestone{
			Title:       m.Title,
			Description: m.Description,
			Deadline:    m.Deadline,
			State:       string(m.State),
			Created:     t,
			Updated:     &t,
			Closed:      m.Closed,
		})
	}

	return milestones, nil
}

func convertGogsLabel(label *gogs.Label) *base.Label {
	return &base.Label{
		Name:  label.Name,
		Color: label.Color,
	}
}

// GetLabels returns labels
func (g *GogsDownloader) GetLabels() ([]*base.Label, error) {
	var perPage = 100
	var labels = make([]*base.Label, 0, perPage)
	ls, err := g.client.ListRepoLabels(g.repoOwner, g.repoName)
	if err != nil {
		return nil, err
	}

	for _, label := range ls {
		labels = append(labels, convertGogsLabel(label))
	}

	return labels, nil
}

// GetReleases returns releases
// FIXME: gogs API haven't support get releases
func (g *GogsDownloader) GetReleases() ([]*base.Release, error) {
	return nil, ErrNotSupported
}

// GetIssues returns issues according start and limit, perPage is not supported
func (g *GogsDownloader) GetIssues(page, perPage int) ([]*base.Issue, bool, error) {
	var allIssues = make([]*base.Issue, 0, perPage)

	issues, err := g.client.ListRepoIssues(g.repoOwner, g.repoName, gogs.ListIssueOption{
		Page: page,
	})
	if err != nil {
		return nil, false, fmt.Errorf("error while listing repos: %v", err)
	}
	for _, issue := range issues {
		if issue.PullRequest != nil {
			continue
		}

		var milestone string
		if issue.Milestone != nil {
			milestone = issue.Milestone.Title
		}
		var labels = make([]*base.Label, 0, len(issue.Labels))
		for _, l := range issue.Labels {
			labels = append(labels, convertGogsLabel(l))
		}

		var closed *time.Time
		if issue.State == gogs.STATE_CLOSED {
			// gogs client haven't provide closed, so we use updated instead
			closed = &issue.Updated
		}

		allIssues = append(allIssues, &base.Issue{
			Title:       issue.Title,
			Number:      issue.Index,
			PosterName:  issue.Poster.Login,
			PosterEmail: issue.Poster.Email,
			Content:     issue.Body,
			Milestone:   milestone,
			State:       string(issue.State),
			Created:     issue.Created,
			Labels:      labels,
			Closed:      closed,
		})
	}

	return allIssues, len(issues) == 0, nil
}

// GetComments returns comments according issueNumber
func (g *GogsDownloader) GetComments(issueNumber int64) ([]*base.Comment, error) {
	var allComments = make([]*base.Comment, 0, 100)

	comments, err := g.client.ListIssueComments(g.repoOwner, g.repoName, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("error while listing repos: %v", err)
	}
	for _, comment := range comments {
		allComments = append(allComments, &base.Comment{
			PosterName:  comment.Poster.Login,
			PosterEmail: comment.Poster.Email,
			Content:     comment.Body,
			Created:     comment.Created,
			Updated:     comment.Updated,
		})
	}

	return allComments, nil
}

// GetPullRequests returns pull requests according page and perPage
func (g *GogsDownloader) GetPullRequests(page, perPage int) ([]*base.PullRequest, error) {
	return nil, ErrNotSupported
}
