// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mail

import (
	"code.gitea.io/git"
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/notification/base"
)

type mailNotifier struct {
}

var (
	_ base.Notifier = &mailNotifier{}
)

// NewNotifier create a new mailNotifier notifier
func NewNotifier() base.Notifier {
	return &mailNotifier{}
}

func (m *mailNotifier) Run() {
}

func (m *mailNotifier) NotifyCreateIssueComment(doer *models.User, repo *models.Repository,
	issue *models.Issue, comment *models.Comment) {
}

func (m *mailNotifier) NotifyNewIssue(issue *models.Issue) {
	if err := issue.MailParticipants(); err != nil {
		log.Error(4, "MailParticipants: %v", err)
	}
}

func (m *mailNotifier) NotifyIssueChangeStatus(doer *models.User, issue *models.Issue, isClosed bool) {
	if err := issue.MailParticipants(); err != nil {
		log.Error(4, "MailParticipants: %v", err)
	}
}

func (m *mailNotifier) NotifyNewPullRequest(pr *models.PullRequest) {
	if err := pr.Issue.MailParticipants(); err != nil {
		log.Error(4, "MailParticipants: %v", err)
	}
}

func (m *mailNotifier) NotifyMergePullRequest(pr *models.PullRequest, doer *models.User, baseRepo *git.Repository) {
}

func (m *mailNotifier) NotifyUpdateComment(doer *models.User, c *models.Comment, oldContent string) {
}

func (m *mailNotifier) NotifyDeleteComment(doer *models.User, c *models.Comment) {
}

func (m *mailNotifier) NotifyDeleteRepository(doer *models.User, repo *models.Repository) {
}

func (m *mailNotifier) NotifyForkRepository(doer *models.User, oldRepo, repo *models.Repository) {
}

func (m *mailNotifier) NotifyNewRelease(rel *models.Release) {
}

func (m *mailNotifier) NotifyUpdateRelease(doer *models.User, rel *models.Release) {
}

func (m *mailNotifier) NotifyDeleteRelease(doer *models.User, rel *models.Release) {
}

func (m *mailNotifier) NotifyIssueChangeMilestone(doer *models.User, issue *models.Issue) {
}

func (m *mailNotifier) NotifyIssueChangeContent(doer *models.User, issue *models.Issue, oldContent string) {
}

func (m *mailNotifier) NotifyIssueChangeAssignee(doer *models.User, issue *models.Issue, removed bool) {
}

func (m *mailNotifier) NotifyIssueClearLabels(doer *models.User, issue *models.Issue) {
}

func (m *mailNotifier) NotifyIssueChangeTitle(doer *models.User, issue *models.Issue, oldTitle string) {
}

func (m *mailNotifier) NotifyIssueChangeLabels(doer *models.User, issue *models.Issue,
	addedLabels []*models.Label, removedLabels []*models.Label) {
}

func (m *mailNotifier) NotifyCreateRepository(doer *models.User, u *models.User, repo *models.Repository) {
}

func (m *mailNotifier) NotifyMigrateRepository(doer *models.User, u *models.User, repo *models.Repository) {
}
