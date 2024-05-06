// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import "code.gitea.io/gitea/models/db"

// Strings for sorting result
const (
	SearchOrderByAlphabetically        db.SearchOrderBy = "owner_name ASC, name ASC"
	SearchOrderByAlphabeticallyReverse db.SearchOrderBy = "owner_name DESC, name DESC"
	SearchOrderByLeastUpdated          db.SearchOrderBy = db.SearchOrderByLeastUpdated
	SearchOrderByRecentUpdated         db.SearchOrderBy = db.SearchOrderByRecentUpdated
	SearchOrderByOldest                db.SearchOrderBy = db.SearchOrderByOldest
	SearchOrderByNewest                db.SearchOrderBy = db.SearchOrderByNewest
	SearchOrderBySize                  db.SearchOrderBy = db.SearchOrderBySize
	SearchOrderBySizeReverse           db.SearchOrderBy = db.SearchOrderBySizeReverse
	SearchOrderByID                    db.SearchOrderBy = db.SearchOrderByID
	SearchOrderByIDReverse             db.SearchOrderBy = db.SearchOrderByIDReverse
	SearchOrderByStars                 db.SearchOrderBy = db.SearchOrderByStars
	SearchOrderByStarsReverse          db.SearchOrderBy = db.SearchOrderByStarsReverse
	SearchOrderByForks                 db.SearchOrderBy = db.SearchOrderByForks
	SearchOrderByForksReverse          db.SearchOrderBy = db.SearchOrderByForksReverse
	SearchOrderByGitSize               db.SearchOrderBy = db.SearchOrderByGitSize
	SearchOrderByGitSizeReverse        db.SearchOrderBy = db.SearchOrderByGitSizeReverse
	SearchOrderByLFSSize               db.SearchOrderBy = db.SearchOrderByLFSSize
	SearchOrderByLFSSizeReverse        db.SearchOrderBy = db.SearchOrderByLFSSizeReverse
)

// SearchOrderByMap represents all possible search order
var SearchOrderByMap = map[string]map[string]db.SearchOrderBy{
	"asc": {
		"alpha":     SearchOrderByAlphabetically,
		"created":   SearchOrderByOldest,
		"updated":   SearchOrderByLeastUpdated,
		"size":      SearchOrderBySize,
		"git_size":  SearchOrderByGitSize,
		"lfs_size":  SearchOrderByLFSSize,
		"id":        SearchOrderByID,
		"num_stars": SearchOrderByStars,
		"num_forks": SearchOrderByForks,
	},
	"desc": {
		"alpha":     SearchOrderByAlphabeticallyReverse,
		"created":   SearchOrderByNewest,
		"updated":   SearchOrderByRecentUpdated,
		"size":      SearchOrderBySizeReverse,
		"git_size":  SearchOrderByGitSizeReverse,
		"lfs_size":  SearchOrderByLFSSizeReverse,
		"id":        SearchOrderByIDReverse,
		"num_stars": SearchOrderByStarsReverse,
		"num_forks": SearchOrderByForksReverse,
	},
}

// SearchOrderByFlatMap is similar to SearchOrderByMap but use human language keywords
// to decide between asc and desc
var SearchOrderByFlatMap = map[string]db.SearchOrderBy{
	"newest":                SearchOrderByMap["desc"]["created"],
	"oldest":                SearchOrderByMap["asc"]["created"],
	"leastupdate":           SearchOrderByMap["asc"]["updated"],
	"reversealphabetically": SearchOrderByMap["desc"]["alpha"],
	"alphabetically":        SearchOrderByMap["asc"]["alpha"],
	"reversesize":           SearchOrderByMap["desc"]["size"],
	"size":                  SearchOrderByMap["asc"]["size"],
	"reversegitsize":        SearchOrderByMap["desc"]["git_size"],
	"gitsize":               SearchOrderByMap["asc"]["git_size"],
	"reverselfssize":        SearchOrderByMap["desc"]["lfs_size"],
	"lfssize":               SearchOrderByMap["asc"]["lfs_size"],
	"moststars":             SearchOrderByMap["desc"]["num_stars"],
	"feweststars":           SearchOrderByMap["asc"]["num_stars"],
	"mostforks":             SearchOrderByMap["desc"]["num_forks"],
	"fewestforks":           SearchOrderByMap["asc"]["num_forks"],
}
