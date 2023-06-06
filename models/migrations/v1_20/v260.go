// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_20 //nolint

import (
	"xorm.io/xorm"
)

func AddTimeEstimateColumnToIssueTable(x *xorm.Engine) error {
	type Issue struct {
		TimeEstimate int64 `xorm:"NOT NULL DEFAULT 0"`
	}

	return x.Sync(new(Issue))
}

func AddColumnsToCommentTable(x *xorm.Engine) error {
	type Comment struct {
		TimeTracked  int64 `xorm:"NOT NULL DEFAULT 0"`
		TimeEstimate int64 `xorm:"NOT NULL DEFAULT 0"`
	}

	return x.Sync(new(Comment))
}
