// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_20 //nolint

import "xorm.io/xorm"

func AddStatusCheckPatternToProtectedBranch(x *xorm.Engine) error {
	type ProtectedBranch struct {
		StatusCheckPattern string `xorm:"TEXT"`
	}

	return x.Sync(new(ProtectedBranch))
}
