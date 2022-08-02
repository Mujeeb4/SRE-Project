// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"fmt"

	"xorm.io/xorm"
)

func addMilestoneLabels(x *xorm.Engine) error {
	type MilestoneLabel struct {
		ID          int64 `xorm:"pk autoincr"`
		MilestoneID int64 `xorm:"UNIQUE(milestoneid_labelid)"`
		LabelID     int64 `xorm:"UNIQUE(milestoneid_labelid)"`
	}

	if err := x.Sync2(new(MilestoneLabel)); err != nil {
		return fmt.Errorf("Sync2: %v", err)
	}
	return nil
}
