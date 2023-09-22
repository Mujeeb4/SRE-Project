// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db_test

import (
	"testing"

	"code.gitea.io/gitea/internal/models/unittest"

	_ "code.gitea.io/gitea/internal/models"
	_ "code.gitea.io/gitea/internal/models/repo"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{})
}
