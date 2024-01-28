// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitrepo

import (
	"code.gitea.io/gitea/modules/git"
)

// RunGitCmdStdString runs the command with options and returns stdout/stderr as string. and store stderr to returned error (err combined with stderr).
func RunGitCmdStdString(repo Repository, c *git.Command, opts *git.RunOpts) (stdout, stderr string, runErr git.RunStdError) {
	if opts.Dir != "" {
		// we must panic here, otherwise there would be bugs if developers set Dir by mistake, and it would be very difficult to debug
		panic("dir field must be empty when using RunStdBytes")
	}
	opts.Dir = repoPath(repo)
	return c.RunStdString(opts)
}

// RunGitCmdStdString runs the command with options and returns stdout/stderr as string. and store stderr to returned error (err combined with stderr).
func RunGitCmdStdStringWiki(repo Repository, c *git.Command, opts *git.RunOpts) (stdout, stderr string, runErr git.RunStdError) {
	if opts.Dir != "" {
		// we must panic here, otherwise there would be bugs if developers set Dir by mistake, and it would be very difficult to debug
		panic("dir field must be empty when using RunStdBytes")
	}
	opts.Dir = wikiPath(repo)
	return c.RunStdString(opts)
}

// RunGitCmdStdBytes runs the command with options and returns stdout/stderr as bytes. and store stderr to returned error (err combined with stderr).
func RunGitCmdStdBytes(repo Repository, c *git.Command, opts *git.RunOpts) (stdout, stderr []byte, runErr git.RunStdError) {
	if opts.Dir != "" {
		// we must panic here, otherwise there would be bugs if developers set Dir by mistake, and it would be very difficult to debug
		panic("dir field must be empty when using RunStdBytes")
	}
	opts.Dir = repoPath(repo)
	return c.RunStdBytes(opts)
}

// RunGitCmd runs the command with the RunOpts
func RunGitCmd(repo Repository, c *git.Command, opts *git.RunOpts) error {
	if opts.Dir != "" {
		// we must panic here, otherwise there would be bugs if developers set Dir by mistake, and it would be very difficult to debug
		panic("dir field must be empty when using RunStdBytes")
	}
	opts.Dir = repoPath(repo)
	return c.Run(opts)
}

// RunGitCmdWiki runs the command with the RunOpts
func RunGitCmdWiki(repo Repository, c *git.Command, opts *git.RunOpts) error {
	if opts.Dir != "" {
		// we must panic here, otherwise there would be bugs if developers set Dir by mistake, and it would be very difficult to debug
		panic("dir field must be empty when using RunStdBytes")
	}
	opts.Dir = repoPath(repo)
	return c.Run(opts)
}
