// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"fmt"
	"time"

	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/timeutil"
	"xorm.io/xorm"
)

func addSyncOnPushColForPushMirror(x *xorm.Engine) error {
	type PushMirror struct {
		ID         int64            `xorm:"pk autoincr"`
		RepoID     int64            `xorm:"INDEX"`
		Repo       *repo.Repository `xorm:"-"`
		RemoteName string

		SyncOnPush     bool
		Interval       time.Duration
		CreatedUnix    timeutil.TimeStamp `xorm:"created"`
		LastUpdateUnix timeutil.TimeStamp `xorm:"INDEX last_update"`
		LastError      string             `xorm:"text"`
	}

	if err := x.Sync2(new(PushMirror)); err != nil {
		return fmt.Errorf("sync2: %v", err)
	}

	return nil
}
