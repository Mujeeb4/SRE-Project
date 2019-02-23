// Copyright 2015 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package git

import (
	"strings"
)

// Tree represents a flat directory listing.
type Tree struct {
	ID   SHA1
	repo *Repository

	// parent tree
	ptree *Tree

	entries       Entries
	entriesParsed bool

	entriesRecursive       Entries
	entriesRecursiveParsed bool
}

// NewTree create a new tree according the repository and commit id
func NewTree(repo *Repository, id SHA1) *Tree {
	return &Tree{
		ID:   id,
		repo: repo,
	}
}

// SubTree get a sub tree by the sub dir path
func (t *Tree) SubTree(rpath string, cache LsTreeCache) (*Tree, error) {
	if len(rpath) == 0 {
		return t, nil
	}

	paths := strings.Split(rpath, "/")
	var (
		err error
		g   = t
		p   = t
		te  *TreeEntry
	)
	for _, name := range paths {
		te, err = p.GetTreeEntryByPath(name, cache)
		if err != nil {
			return nil, err
		}

		g, err = t.repo.getTree(te.ID)
		if err != nil {
			return nil, err
		}
		g.ptree = p
		p = g
	}
	return g, nil
}

// ListEntries returns all entries of current tree.
func (t *Tree) ListEntries(cache LsTreeCache) (Entries, error) {
	if t.entriesParsed {
		return t.entries, nil
	}

	var err error
	id := t.ID.String()
	if cache != nil {
		t.entries, err = cache.Get(t.repo.Path, id)
		if err == nil && t.entries != nil {
			//log("Hit ls tree cache: %s, %s", t.repo.Path, id)
			t.entriesParsed = true
			return t.entries, nil
		}
	}

	stdout, err := NewCommand("ls-tree", id).RunInDirBytes(t.repo.Path)
	if err != nil {
		return nil, err
	}

	t.entries, err = parseTreeEntries(stdout, t)
	if err == nil {
		t.entriesParsed = true
		if cache != nil && t.entries != nil {
			cache.Put(t.repo.Path, id, t.entries)
		}
	}

	return t.entries, err
}

// ListEntriesRecursive returns all entries of current tree recursively including all subtrees
func (t *Tree) ListEntriesRecursive() (Entries, error) {
	if t.entriesRecursiveParsed {
		return t.entriesRecursive, nil
	}
	stdout, err := NewCommand("ls-tree", "-t", "-r", t.ID.String()).RunInDirBytes(t.repo.Path)
	if err != nil {
		return nil, err
	}

	t.entriesRecursive, err = parseTreeEntries(stdout, t)
	if err == nil {
		t.entriesRecursiveParsed = true
	}

	return t.entriesRecursive, err
}
