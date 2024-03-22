// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package internal

import (
	"math"

	"code.gitea.io/gitea/models/db"
)

// ParsePaginator parses a db.Paginator into a skip and limit
func ParsePaginator(listOptions *db.ListOptions, max ...int) (int, int) {
	// Use a very large number to indicate no limit
	unlimited := math.MaxInt32
	if len(max) > 0 {
		// Some indexer engines have a limit on the page size, respect that
		unlimited = max[0]
	}

	if listOptions == nil || listOptions.IsListAll() {
		// It shouldn't happen. In actual usage scenarios, there should not be requests to search all.
		// But if it does happen, respect it and return "unlimited".
		// And it's also useful for testing.
		return 0, unlimited
	}

	if listOptions.PageSize == 0 {
		// Do not return any results when searching, it's used to get the total count only.
		return 0, 0
	}

	return listOptions.GetSkipTake()
}
