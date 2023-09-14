// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_21 //nolint

import (
	"xorm.io/xorm"
)

type BadgeUnique struct {
	Slug string `xorm:"UNIQUE"`
}

func (BadgeUnique) TableName() string {
	return "badge"
}

func UseSlugInsteadOfIDForBadges(x *xorm.Engine) error {
	type Badge struct {
		Slug string
	}

	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}
	err := sess.Sync(new(Badge))
	if err != nil {
		return err
	}

	_, err = sess.Exec("UPDATE `badge` SET `slug` = `id`")
	if err != nil {
		return err
	}

	err = sess.Sync(new(BadgeUnique))
	if err != nil {
		return err
	}

	return sess.Commit()
}
