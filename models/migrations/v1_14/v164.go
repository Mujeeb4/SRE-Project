// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_14 //nolint

import (
	"fmt"

	"xorm.io/xorm"
)

// OAuth2Grant here is a snapshot of models.OAuth2Grant for this version
// of the database, as it does not appear to have been added as a part
// of a previous migration.
type OAuth2Grant struct {
	ID            int64  `xorm:"pk autoincr"`
	UserID        int64  `xorm:"INDEX unique(user_application)"`
	ApplicationID int64  `xorm:"INDEX unique(user_application)"`
	Counter       int64  `xorm:"NOT NULL DEFAULT 1"`
	Scope         string `xorm:"TEXT"`
	Nonce         string `xorm:"TEXT"`
	CreatedUnix   int64  `xorm:"created"`
	UpdatedUnix   int64  `xorm:"updated"`
}

// TableName sets the database table name to be the correct one, as the
// autogenerated table name for this struct is "o_auth2_grant".
func (grant *OAuth2Grant) TableName() string {
	return "oauth2_grant"
}

func AddScopeAndNonceColumnsToOAuth2Grant(x *xorm.Engine) error {
	if err := x.Sync(new(OAuth2Grant)); err != nil {
		return fmt.Errorf("Sync: %w", err)
	}
	return nil
}
