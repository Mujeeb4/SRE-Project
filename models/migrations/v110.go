// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"code.gitea.io/gitea/models"

	"xorm.io/xorm"
)

func addBranchProtectionCanPushAndEnableWhitelist(x *xorm.Engine) error {

	type ProtectedBranch struct {
		CanPush                  bool  `xorm:"NOT NULL DEFAULT false"`
		EnableWhitelist          bool  `xorm:"NOT NULL DEFAULT false"`
		EnableApprovalsWhitelist bool  `xorm:"NOT NULL DEFAULT false"`
		RequiredApprovals        int64 `xorm:"NOT NULL DEFAULT 0"`
	}

	type Review struct {
		ID       int64 `xorm:"pk autoincr"`
		Official bool  `xorm:"NOT NULL DEFAULT false"`
	}

	sess := x.NewSession()
	defer sess.Close()

	if err := sess.Sync2(new(ProtectedBranch)); err != nil {
		return err
	}

	if err := sess.Sync2(new(Review)); err != nil {
		return err
	}

	if _, err := sess.Exec("UPDATE `protected_branch` SET `can_push` = `enable_whitelist`"); err != nil {
		return err
	}
	if _, err := sess.Exec("UPDATE `protected_branch` SET `enable_approvals_whitelist` = ? WHERE `required_approvals` > ?", true, 0); err != nil {
		return err
	}

	var pageSize int64 = 20
	totallPRs, err := x.Count(new(models.PullRequest))
	if err != nil {
		return err
	}
	var totalPages int64
	totalPages = totallPRs / pageSize

	// Find latest review of each user in each pull request, and set official field if appropriate
	reviews := []*models.Review{}
	var page int64
	for page = 0; page <= totalPages; page++ {
		if err := sess.Sql("SELECT * FROM review WHERE id IN (SELECT max(id) as id FROM review WHERE issue_id > ? AND issue_id <= ? AND type in (?, ?) GROUP BY issue_id, reviewer_id)",
			page*pageSize, (page+1)*pageSize, models.ReviewTypeApprove, models.ReviewTypeReject).
			Find(&reviews); err != nil {
			return err
		}

		for _, review := range reviews {
			if err := review.LoadAttributes(); err != nil {
				return err
			}
			official, err := models.IsOfficialReviewer(review.Issue, review.Reviewer)
			if err != nil {
				return err
			}
			review.Official = official

			if _, err := sess.ID(review.ID).Cols("official").Update(review); err != nil {
				return err
			}
		}

	}

	return sess.Commit()
}
