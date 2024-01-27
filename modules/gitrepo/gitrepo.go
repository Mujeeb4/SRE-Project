// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitrepo

import (
	"context"
	"io"
	"path"
	"path/filepath"
	"strings"

	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
)

type Repository interface {
	FullName() string // should return "<user>/<repo>"
}

func repoPath(fullName string) string {
	ownerName, repoName := path.Split(fullName)
	return filepath.Join(setting.RepoRootPath, strings.ToLower(ownerName), strings.ToLower(repoName)+".git")
}

func wikiPath(fullName string) string {
	ownerName, repoName := path.Split(fullName)
	return filepath.Join(setting.RepoRootPath, strings.ToLower(ownerName), strings.ToLower(repoName)+".wiki.git")
}

// OpenRepository opens the repository at the given relative path with the provided context.
func OpenRepository(ctx context.Context, repo Repository) (*git.Repository, error) {
	return git.OpenRepository(ctx, repoPath(repo.FullName()))
}

func OpenWikiRepository(ctx context.Context, repo Repository) (*git.Repository, error) {
	return git.OpenRepository(ctx, wikiPath(repo.FullName()))
}

// contextKey is a value for use with context.WithValue.
type contextKey struct {
	name string
}

// RepositoryContextKey is a context key. It is used with context.Value() to get the current Repository for the context
var RepositoryContextKey = &contextKey{"repository"}

// RepositoryFromContext attempts to get the repository from the context
func repositoryFromContext(ctx context.Context, repo Repository) *git.Repository {
	value := ctx.Value(RepositoryContextKey)
	if value == nil {
		return nil
	}

	if gitRepo, ok := value.(*git.Repository); ok && gitRepo != nil {
		if gitRepo.Path == repoPath(repo.FullName()) {
			return gitRepo
		}
	}

	return nil
}

type nopCloser func()

func (nopCloser) Close() error { return nil }

// RepositoryFromContextOrOpen attempts to get the repository from the context or just opens it
func RepositoryFromContextOrOpen(ctx context.Context, repo Repository) (*git.Repository, io.Closer, error) {
	gitRepo := repositoryFromContext(ctx, repo)
	if gitRepo != nil {
		return gitRepo, nopCloser(nil), nil
	}

	gitRepo, err := OpenRepository(ctx, repo)
	return gitRepo, gitRepo, err
}

// repositoryFromContextPath attempts to get the repository from the context
func repositoryFromContextPath(ctx context.Context, path string) *git.Repository {
	value := ctx.Value(RepositoryContextKey)
	if value == nil {
		return nil
	}

	if repo, ok := value.(*git.Repository); ok && repo != nil {
		if repo.Path == path {
			return repo
		}
	}

	return nil
}

// RepositoryFromContextOrOpenPath attempts to get the repository from the context or just opens it
// Deprecated: Use RepositoryFromContextOrOpen instead
func RepositoryFromContextOrOpenPath(ctx context.Context, path string) (*git.Repository, io.Closer, error) {
	gitRepo := repositoryFromContextPath(ctx, path)
	if gitRepo != nil {
		return gitRepo, nopCloser(nil), nil
	}

	gitRepo, err := git.OpenRepository(ctx, path)
	return gitRepo, gitRepo, err
}
