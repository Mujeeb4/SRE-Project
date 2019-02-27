// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package git_data

import (
	"code.gitea.io/git"
	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/sdk/gitea"
	"fmt"
)

// GetTreeBySHA get the GitTreeResponse of a repository using a sha hash.
func GetTreeBySHA(repo *models.Repository, sha string, page, perPage int, recursive bool) *gitea.GitTreeResponse {
	gitRepo, err := git.OpenRepository(repo.RepoPath())
	gitTree, err := gitRepo.GetTree(sha)
	if err != nil || gitTree == nil {
		return nil
	}
	tree := new(gitea.GitTreeResponse)
	tree.SHA = gitTree.ID.String()
	tree.URL = repo.APIURL() + "/git/trees/" + tree.SHA
	var entries git.Entries
	if recursive {
		entries, err = gitTree.ListEntriesRecursive()
	} else {
		entries, err = gitTree.ListEntries()
	}
	if err != nil {
		return tree
	}
	apiUrl := repo.APIURL()
	apiUrlLen := len(apiUrl)

	// 51 is len(sha1) + len("/git/blobs/"). 40 + 11.
	blobURL := make([]byte, apiUrlLen+51)
	copy(blobURL[:], apiUrl)
	copy(blobURL[apiUrlLen:], "/git/blobs/")

	// 51 is len(sha1) + len("/git/trees/"). 40 + 11.
	treeURL := make([]byte, apiUrlLen+51)
	copy(treeURL[:], apiUrl)
	copy(treeURL[apiUrlLen:], "/git/trees/")

	// 40 is the size of the sha1 hash in hexadecimal format.
	copyPos := len(treeURL) - 40

	if perPage <= 0 || perPage > setting.API.DefaultGitTreesPerPage {
		perPage = setting.API.DefaultGitTreesPerPage
	}
	if page <= 0 {
		page = 1
	}
	tree.Page = page
	tree.TotalCount = len(entries)
	rangeStart := perPage * (page - 1)
	if rangeStart >= len(entries) {
		return tree
	}
	var rangeEnd int
	if len(entries) > perPage {
		tree.Truncated = true
	}
	if rangeStart+perPage < len(entries) {
		rangeEnd = rangeStart + perPage
	} else {
		rangeEnd = len(entries)
	}
	tree.Entries = make([]gitea.GitEntry, rangeEnd-rangeStart)
	for e := rangeStart; e < rangeEnd; e++ {
		i := e - rangeStart
		tree.Entries[i].Path = entries[e].Name()
		tree.Entries[i].Mode = fmt.Sprintf("%06x", entries[e].Mode())
		tree.Entries[i].Type = string(entries[e].Type)
		tree.Entries[i].Size = entries[e].Size()
		tree.Entries[i].SHA = entries[e].ID.String()

		if entries[e].IsDir() {
			copy(treeURL[copyPos:], entries[e].ID.String())
			tree.Entries[i].URL = string(treeURL[:])
		} else {
			copy(blobURL[copyPos:], entries[e].ID.String())
			tree.Entries[i].URL = string(blobURL[:])
		}
	}
	return tree
}
