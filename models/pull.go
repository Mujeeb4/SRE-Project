// Copyright 2015 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"code.gitea.io/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/process"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/sync"
	api "code.gitea.io/sdk/gitea"
	"github.com/Unknwon/com"
	"github.com/go-xorm/xorm"
)

var pullRequestQueue = sync.NewUniqueQueue(setting.Repository.PullRequestQueueLength)

// PullRequestType defines pull request type
type PullRequestType int

// Enumerate all the pull request types
const (
	PullRequestGitea PullRequestType = iota
	PullRequestGit
)

// PullRequestStatus defines pull request status
type PullRequestStatus int

// Enumerate all the pull request status
const (
	PullRequestStatusConflict PullRequestStatus = iota
	PullRequestStatusChecking
	PullRequestStatusMergeable
)

// PullRequest represents relation between pull request and repositories.
type PullRequest struct {
	ID     int64 `xorm:"pk autoincr"`
	Type   PullRequestType
	Status PullRequestStatus

	IssueID int64  `xorm:"INDEX"`
	Issue   *Issue `xorm:"-"`
	Index   int64

	HeadRepoID   int64
	HeadRepo     *Repository `xorm:"-"`
	BaseRepoID   int64
	BaseRepo     *Repository `xorm:"-"`
	HeadUserName string
	HeadBranch   string
	BaseBranch   string
	MergeBase    string `xorm:"VARCHAR(40)"`

	HasMerged      bool
	MergedCommitID string `xorm:"VARCHAR(40)"`
	MergerID       int64
	Merger         *User     `xorm:"-"`
	Merged         time.Time `xorm:"-"`
	MergedUnix     int64
}

// BeforeUpdate is invoked from XORM before updating an object of this type.
func (pr *PullRequest) BeforeUpdate() {
	pr.MergedUnix = pr.Merged.Unix()
}

// AfterSet is invoked from XORM after setting the value of a field of this object.
// Note: don't try to get Issue because will end up recursive querying.
func (pr *PullRequest) AfterSet(colName string, _ xorm.Cell) {
	switch colName {
	case "merged_unix":
		if !pr.HasMerged {
			return
		}

		pr.Merged = time.Unix(pr.MergedUnix, 0).Local()
	}
}

// Note: don't try to get Issue because will end up recursive querying.
func (pr *PullRequest) loadAttributes(e Engine) (err error) {
	if pr.HasMerged && pr.Merger == nil {
		pr.Merger, err = getUserByID(e, pr.MergerID)
		if IsErrUserNotExist(err) {
			pr.MergerID = -1
			pr.Merger = NewGhostUser()
		} else if err != nil {
			return fmt.Errorf("getUserByID [%d]: %v", pr.MergerID, err)
		}
	}

	return nil
}

// LoadAttributes loads pull request attributes from database
func (pr *PullRequest) LoadAttributes() error {
	return pr.loadAttributes(x)
}

// LoadIssue loads issue information from database
func (pr *PullRequest) LoadIssue() (err error) {
	if pr.Issue != nil {
		return nil
	}

	pr.Issue, err = GetIssueByID(pr.IssueID)
	return err
}

// APIFormat assumes following fields have been assigned with valid values:
// Required - Issue
// Optional - Merger
func (pr *PullRequest) APIFormat() *api.PullRequest {
	var (
		baseBranch *Branch
		headBranch *Branch
		baseCommit *git.Commit
		headCommit *git.Commit
		err        error
	)
	apiIssue := pr.Issue.APIFormat()
	if pr.BaseRepo == nil || pr.HeadRepo == nil {
		log.Critical(log.CRITICAL, "Base/HeadRepo not set for PR!!!")
		return nil
	}
	if baseBranch, err = pr.BaseRepo.GetBranch(pr.BaseBranch); err != nil {
		return nil
	}
	if baseCommit, err = baseBranch.GetCommit(); err != nil {
		return nil
	}
	if headBranch, err = pr.HeadRepo.GetBranch(pr.HeadBranch); err != nil {
		return nil
	}
	if headCommit, err = headBranch.GetCommit(); err != nil {
		return nil
	}
	apiBaseBranchInfo := &api.PRBranchInfo{
		Name:       pr.BaseBranch,
		Ref:        pr.BaseBranch,
		Sha:        baseCommit.ID.String(),
		RepoID:     pr.BaseRepoID,
		Repository: pr.BaseRepo.APIFormat(nil),
	}
	apiHeadBranchInfo := &api.PRBranchInfo{
		Name:       pr.HeadBranch,
		Ref:        pr.HeadBranch,
		Sha:        headCommit.ID.String(),
		RepoID:     pr.HeadRepoID,
		Repository: pr.HeadRepo.APIFormat(nil),
	}
	apiPullRequest := &api.PullRequest{
		ID:        pr.ID,
		Index:     pr.Index,
		Poster:    apiIssue.Poster,
		Title:     apiIssue.Title,
		Body:      apiIssue.Body,
		Labels:    apiIssue.Labels,
		Milestone: apiIssue.Milestone,
		Assignee:  apiIssue.Assignee,
		State:     apiIssue.State,
		Comments:  apiIssue.Comments,
		HTMLURL:   pr.Issue.HTMLURL(),
		DiffURL:   pr.Issue.DiffURL(),
		PatchURL:  pr.Issue.PatchURL(),
		HasMerged: pr.HasMerged,
		Base:      apiBaseBranchInfo,
		Head:      apiHeadBranchInfo,
		MergeBase: pr.MergeBase,
	}

	if pr.Status != PullRequestStatusChecking {
		mergeable := pr.Status != PullRequestStatusConflict
		apiPullRequest.Mergeable = mergeable
	}
	if pr.HasMerged {
		apiPullRequest.Merged = &pr.Merged
		apiPullRequest.MergedCommitID = &pr.MergedCommitID
		apiPullRequest.MergedBy = pr.Merger.APIFormat()
	}

	return apiPullRequest
}

func (pr *PullRequest) getHeadRepo(e Engine) (err error) {
	pr.HeadRepo, err = getRepositoryByID(e, pr.HeadRepoID)
	if err != nil && !IsErrRepoNotExist(err) {
		return fmt.Errorf("getRepositoryByID(head): %v", err)
	}
	return nil
}

// GetHeadRepo loads the head repository
func (pr *PullRequest) GetHeadRepo() error {
	return pr.getHeadRepo(x)
}

// GetBaseRepo loads the target repository
func (pr *PullRequest) GetBaseRepo() (err error) {
	if pr.BaseRepo != nil {
		return nil
	}

	pr.BaseRepo, err = GetRepositoryByID(pr.BaseRepoID)
	if err != nil {
		return fmt.Errorf("GetRepositoryByID(base): %v", err)
	}
	return nil
}

// IsChecking returns true if this pull request is still checking conflict.
func (pr *PullRequest) IsChecking() bool {
	return pr.Status == PullRequestStatusChecking
}

// CanAutoMerge returns true if this pull request can be merged automatically.
func (pr *PullRequest) CanAutoMerge() bool {
	return pr.Status == PullRequestStatusMergeable
}

// Merge merges pull request to base repository.
// FIXME: add repoWorkingPull make sure two merges does not happen at same time.
func (pr *PullRequest) Merge(doer *User, baseGitRepo *git.Repository) (err error) {
	if err = pr.GetHeadRepo(); err != nil {
		return fmt.Errorf("GetHeadRepo: %v", err)
	} else if err = pr.GetBaseRepo(); err != nil {
		return fmt.Errorf("GetBaseRepo: %v", err)
	}

	defer func() {
		go HookQueue.Add(pr.BaseRepo.ID)
		go AddTestPullRequestTask(doer, pr.BaseRepo.ID, pr.BaseBranch, false)
	}()

	sess := x.NewSession()
	defer sessionRelease(sess)
	if err = sess.Begin(); err != nil {
		return err
	}

	if err = pr.Issue.changeStatus(sess, doer, pr.Issue.Repo, true); err != nil {
		return fmt.Errorf("Issue.changeStatus: %v", err)
	}

	headRepoPath := RepoPath(pr.HeadUserName, pr.HeadRepo.Name)
	headGitRepo, err := git.OpenRepository(headRepoPath)
	if err != nil {
		return fmt.Errorf("OpenRepository: %v", err)
	}

	// Clone base repo.
	tmpBasePath := path.Join(setting.AppDataPath, "tmp/repos", com.ToStr(time.Now().Nanosecond())+".git")

	if err := os.MkdirAll(path.Dir(tmpBasePath), os.ModePerm); err != nil {
		return fmt.Errorf("Fail to create dir %s: %v", tmpBasePath, err)
	}

	defer os.RemoveAll(path.Dir(tmpBasePath))

	var stderr string
	if _, stderr, err = process.ExecTimeout(5*time.Minute,
		fmt.Sprintf("PullRequest.Merge (git clone): %s", tmpBasePath),
		"git", "clone", baseGitRepo.Path, tmpBasePath); err != nil {
		return fmt.Errorf("git clone: %s", stderr)
	}

	// Check out base branch.
	if _, stderr, err = process.ExecDir(-1, tmpBasePath,
		fmt.Sprintf("PullRequest.Merge (git checkout): %s", tmpBasePath),
		"git", "checkout", pr.BaseBranch); err != nil {
		return fmt.Errorf("git checkout: %s", stderr)
	}

	// Add head repo remote.
	if _, stderr, err = process.ExecDir(-1, tmpBasePath,
		fmt.Sprintf("PullRequest.Merge (git remote add): %s", tmpBasePath),
		"git", "remote", "add", "head_repo", headRepoPath); err != nil {
		return fmt.Errorf("git remote add [%s -> %s]: %s", headRepoPath, tmpBasePath, stderr)
	}

	// Merge commits.
	if _, stderr, err = process.ExecDir(-1, tmpBasePath,
		fmt.Sprintf("PullRequest.Merge (git fetch): %s", tmpBasePath),
		"git", "fetch", "head_repo"); err != nil {
		return fmt.Errorf("git fetch [%s -> %s]: %s", headRepoPath, tmpBasePath, stderr)
	}

	if _, stderr, err = process.ExecDir(-1, tmpBasePath,
		fmt.Sprintf("PullRequest.Merge (git merge --no-ff --no-commit): %s", tmpBasePath),
		"git", "merge", "--no-ff", "--no-commit", "head_repo/"+pr.HeadBranch); err != nil {
		return fmt.Errorf("git merge --no-ff --no-commit [%s]: %v - %s", tmpBasePath, err, stderr)
	}

	sig := doer.NewGitSig()
	if _, stderr, err = process.ExecDir(-1, tmpBasePath,
		fmt.Sprintf("PullRequest.Merge (git merge): %s", tmpBasePath),
		"git", "commit", fmt.Sprintf("--author='%s <%s>'", sig.Name, sig.Email),
		"-m", fmt.Sprintf("Merge branch '%s' of %s/%s into %s", pr.HeadBranch, pr.HeadUserName, pr.HeadRepo.Name, pr.BaseBranch)); err != nil {
		return fmt.Errorf("git commit [%s]: %v - %s", tmpBasePath, err, stderr)
	}

	// Push back to upstream.
	if _, stderr, err = process.ExecDir(-1, tmpBasePath,
		fmt.Sprintf("PullRequest.Merge (git push): %s", tmpBasePath),
		"git", "push", baseGitRepo.Path, pr.BaseBranch); err != nil {
		return fmt.Errorf("git push: %s", stderr)
	}

	pr.MergedCommitID, err = headGitRepo.GetBranchCommitID(pr.HeadBranch)
	if err != nil {
		return fmt.Errorf("GetBranchCommit: %v", err)
	}

	pr.HasMerged = true
	pr.Merged = time.Now()
	pr.MergerID = doer.ID
	if _, err = sess.Id(pr.ID).AllCols().Update(pr); err != nil {
		return fmt.Errorf("update pull request: %v", err)
	}

	if err = sess.Commit(); err != nil {
		return fmt.Errorf("Commit: %v", err)
	}

	if err = MergePullRequestAction(doer, pr.Issue.Repo, pr.Issue); err != nil {
		log.Error(4, "MergePullRequestAction [%d]: %v", pr.ID, err)
	}

	// Reload pull request information.
	if err = pr.LoadAttributes(); err != nil {
		log.Error(4, "LoadAttributes: %v", err)
		return nil
	}
	if err = PrepareWebhooks(pr.Issue.Repo, HookEventPullRequest, &api.PullRequestPayload{
		Action:      api.HookIssueClosed,
		Index:       pr.Index,
		PullRequest: pr.APIFormat(),
		Repository:  pr.Issue.Repo.APIFormat(nil),
		Sender:      doer.APIFormat(),
	}); err != nil {
		log.Error(4, "PrepareWebhooks: %v", err)
		return nil
	}

	l, err := headGitRepo.CommitsBetweenIDs(pr.MergedCommitID, pr.MergeBase)
	if err != nil {
		log.Error(4, "CommitsBetweenIDs: %v", err)
		return nil
	}

	// TODO: when squash commits, no need to append merge commit.
	// It is possible that head branch is not fully sync with base branch for merge commits,
	// so we need to get latest head commit and append merge commit manully
	// to avoid strange diff commits produced.
	mergeCommit, err := baseGitRepo.GetBranchCommit(pr.BaseBranch)
	if err != nil {
		log.Error(4, "GetBranchCommit: %v", err)
		return nil
	}
	l.PushFront(mergeCommit)

	p := &api.PushPayload{
		Ref:        git.BRANCH_PREFIX + pr.BaseBranch,
		Before:     pr.MergeBase,
		After:      pr.MergedCommitID,
		CompareURL: setting.AppURL + pr.BaseRepo.ComposeCompareURL(pr.MergeBase, pr.MergedCommitID),
		Commits:    ListToPushCommits(l).ToAPIPayloadCommits(pr.BaseRepo.HTMLURL()),
		Repo:       pr.BaseRepo.APIFormat(nil),
		Pusher:     pr.HeadRepo.MustOwner().APIFormat(),
		Sender:     doer.APIFormat(),
	}
	if err = PrepareWebhooks(pr.BaseRepo, HookEventPush, p); err != nil {
		return fmt.Errorf("PrepareWebhooks: %v", err)
	}
	return nil
}

// patchConflicts is a list of conflict description from Git.
var patchConflicts = []string{
	"patch does not apply",
	"already exists in working directory",
	"unrecognized input",
	"error:",
}

// testPatch checks if patch can be merged to base repository without conflict.
// FIXME: make a mechanism to clean up stable local copies.
func (pr *PullRequest) testPatch() (err error) {
	if pr.BaseRepo == nil {
		pr.BaseRepo, err = GetRepositoryByID(pr.BaseRepoID)
		if err != nil {
			return fmt.Errorf("GetRepositoryByID: %v", err)
		}
	}

	patchPath, err := pr.BaseRepo.PatchPath(pr.Index)
	if err != nil {
		return fmt.Errorf("BaseRepo.PatchPath: %v", err)
	}

	// Fast fail if patch does not exist, this assumes data is cruppted.
	if !com.IsFile(patchPath) {
		log.Trace("PullRequest[%d].testPatch: ignored cruppted data", pr.ID)
		return nil
	}

	repoWorkingPool.CheckIn(com.ToStr(pr.BaseRepoID))
	defer repoWorkingPool.CheckOut(com.ToStr(pr.BaseRepoID))

	log.Trace("PullRequest[%d].testPatch (patchPath): %s", pr.ID, patchPath)

	if err := pr.BaseRepo.UpdateLocalCopyBranch(pr.BaseBranch); err != nil {
		return fmt.Errorf("UpdateLocalCopy: %v", err)
	}

	pr.Status = PullRequestStatusChecking
	_, stderr, err := process.ExecDir(-1, pr.BaseRepo.LocalCopyPath(),
		fmt.Sprintf("testPatch (git apply --check): %d", pr.BaseRepo.ID),
		"git", "apply", "--check", patchPath)
	if err != nil {
		for i := range patchConflicts {
			if strings.Contains(stderr, patchConflicts[i]) {
				log.Trace("PullRequest[%d].testPatch (apply): has conflict", pr.ID)
				fmt.Println(stderr)
				pr.Status = PullRequestStatusConflict
				return nil
			}
		}

		return fmt.Errorf("git apply --check: %v - %s", err, stderr)
	}
	return nil
}

// NewPullRequest creates new pull request with labels for repository.
func NewPullRequest(repo *Repository, pull *Issue, labelIDs []int64, uuids []string, pr *PullRequest, patch []byte) (err error) {
	sess := x.NewSession()
	defer sessionRelease(sess)
	if err = sess.Begin(); err != nil {
		return err
	}

	if err = newIssue(sess, NewIssueOptions{
		Repo:        repo,
		Issue:       pull,
		LableIDs:    labelIDs,
		Attachments: uuids,
		IsPull:      true,
	}); err != nil {
		return fmt.Errorf("newIssue: %v", err)
	}

	pr.Index = pull.Index
	if err = repo.SavePatch(pr.Index, patch); err != nil {
		return fmt.Errorf("SavePatch: %v", err)
	}

	pr.BaseRepo = repo
	if err = pr.testPatch(); err != nil {
		return fmt.Errorf("testPatch: %v", err)
	}
	// No conflict appears after test means mergeable.
	if pr.Status == PullRequestStatusChecking {
		pr.Status = PullRequestStatusMergeable
	}

	pr.IssueID = pull.ID
	if _, err = sess.Insert(pr); err != nil {
		return fmt.Errorf("insert pull repo: %v", err)
	}

	if err = sess.Commit(); err != nil {
		return fmt.Errorf("Commit: %v", err)
	}

	if err = NotifyWatchers(&Action{
		ActUserID:    pull.Poster.ID,
		ActUserName:  pull.Poster.Name,
		OpType:       ActionCreatePullRequest,
		Content:      fmt.Sprintf("%d|%s", pull.Index, pull.Title),
		RepoID:       repo.ID,
		RepoUserName: repo.Owner.Name,
		RepoName:     repo.Name,
		IsPrivate:    repo.IsPrivate,
	}); err != nil {
		log.Error(4, "NotifyWatchers: %v", err)
	} else if err = pull.MailParticipants(); err != nil {
		log.Error(4, "MailParticipants: %v", err)
	}

	pr.Issue = pull
	pull.PullRequest = pr
	if err = PrepareWebhooks(repo, HookEventPullRequest, &api.PullRequestPayload{
		Action:      api.HookIssueOpened,
		Index:       pull.Index,
		PullRequest: pr.APIFormat(),
		Repository:  repo.APIFormat(nil),
		Sender:      pull.Poster.APIFormat(),
	}); err != nil {
		log.Error(4, "PrepareWebhooks: %v", err)
	}
	go HookQueue.Add(repo.ID)

	return nil
}

// PullRequestsOptions holds the options for PRs
type PullRequestsOptions struct {
	Page        int
	State       string
	SortType    string
	Labels      []string
	MilestoneID int64
}

func listPullRequestStatement(baseRepoID int64, opts *PullRequestsOptions) *xorm.Session {
	sess := x.Where("pull_request.base_repo_id=?", baseRepoID)

	sess.Join("INNER", "issue", "pull_request.issue_id = issue.id")
	switch opts.State {
	case "closed", "open":
		sess.And("issue.is_closed=?", opts.State == "closed")
	}

	return sess
}

// PullRequests returns all pull requests for a base Repo by the given conditions
func PullRequests(baseRepoID int64, opts *PullRequestsOptions) ([]*PullRequest, int64, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}

	countSession := listPullRequestStatement(baseRepoID, opts)
	maxResults, err := countSession.Count(new(PullRequest))
	if err != nil {
		log.Error(4, "Count PRs", err)
		return nil, maxResults, err
	}

	prs := make([]*PullRequest, 0, ItemsPerPage)
	findSession := listPullRequestStatement(baseRepoID, opts)
	findSession.Limit(ItemsPerPage, (opts.Page-1)*ItemsPerPage)
	return prs, maxResults, findSession.Find(&prs)
}

// GetUnmergedPullRequest returnss a pull request that is open and has not been merged
// by given head/base and repo/branch.
func GetUnmergedPullRequest(headRepoID, baseRepoID int64, headBranch, baseBranch string) (*PullRequest, error) {
	pr := new(PullRequest)
	has, err := x.
		Where("head_repo_id=? AND head_branch=? AND base_repo_id=? AND base_branch=? AND has_merged=? AND issue.is_closed=?",
			headRepoID, headBranch, baseRepoID, baseBranch, false, false).
		Join("INNER", "issue", "issue.id=pull_request.issue_id").
		Get(pr)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrPullRequestNotExist{0, 0, headRepoID, baseRepoID, headBranch, baseBranch}
	}

	return pr, nil
}

// GetUnmergedPullRequestsByHeadInfo returnss all pull requests that are open and has not been merged
// by given head information (repo and branch).
func GetUnmergedPullRequestsByHeadInfo(repoID int64, branch string) ([]*PullRequest, error) {
	prs := make([]*PullRequest, 0, 2)
	return prs, x.
		Where("head_repo_id = ? AND head_branch = ? AND has_merged = ? AND issue.is_closed = ?",
			repoID, branch, false, false).
		Join("INNER", "issue", "issue.id = pull_request.issue_id").
		Find(&prs)
}

// GetUnmergedPullRequestsByBaseInfo returnss all pull requests that are open and has not been merged
// by given base information (repo and branch).
func GetUnmergedPullRequestsByBaseInfo(repoID int64, branch string) ([]*PullRequest, error) {
	prs := make([]*PullRequest, 0, 2)
	return prs, x.
		Where("base_repo_id=? AND base_branch=? AND has_merged=? AND issue.is_closed=?",
			repoID, branch, false, false).
		Join("INNER", "issue", "issue.id=pull_request.issue_id").
		Find(&prs)
}

// GetPullRequestByIndex returns a pull request by the given index
func GetPullRequestByIndex(repoID int64, index int64) (*PullRequest, error) {
	pr := &PullRequest{
		BaseRepoID: repoID,
		Index:      index,
	}

	has, err := x.Get(pr)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrPullRequestNotExist{0, repoID, index, 0, "", ""}
	}

	if err = pr.LoadAttributes(); err != nil {
		return nil, err
	}
	if err = pr.LoadIssue(); err != nil {
		return nil, err
	}

	return pr, nil
}

func getPullRequestByID(e Engine, id int64) (*PullRequest, error) {
	pr := new(PullRequest)
	has, err := e.Id(id).Get(pr)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrPullRequestNotExist{id, 0, 0, 0, "", ""}
	}
	return pr, pr.loadAttributes(e)
}

// GetPullRequestByID returns a pull request by given ID.
func GetPullRequestByID(id int64) (*PullRequest, error) {
	return getPullRequestByID(x, id)
}

func getPullRequestByIssueID(e Engine, issueID int64) (*PullRequest, error) {
	pr := &PullRequest{
		IssueID: issueID,
	}
	has, err := e.Get(pr)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrPullRequestNotExist{0, issueID, 0, 0, "", ""}
	}
	return pr, pr.loadAttributes(e)
}

// GetPullRequestByIssueID returns pull request by given issue ID.
func GetPullRequestByIssueID(issueID int64) (*PullRequest, error) {
	return getPullRequestByIssueID(x, issueID)
}

// Update updates all fields of pull request.
func (pr *PullRequest) Update() error {
	_, err := x.Id(pr.ID).AllCols().Update(pr)
	return err
}

// UpdateCols updates specific fields of pull request.
func (pr *PullRequest) UpdateCols(cols ...string) error {
	_, err := x.Id(pr.ID).Cols(cols...).Update(pr)
	return err
}

// UpdatePatch generates and saves a new patch.
func (pr *PullRequest) UpdatePatch() (err error) {
	if err = pr.GetHeadRepo(); err != nil {
		return fmt.Errorf("GetHeadRepo: %v", err)
	} else if pr.HeadRepo == nil {
		log.Trace("PullRequest[%d].UpdatePatch: ignored cruppted data", pr.ID)
		return nil
	}

	if err = pr.GetBaseRepo(); err != nil {
		return fmt.Errorf("GetBaseRepo: %v", err)
	}

	headGitRepo, err := git.OpenRepository(pr.HeadRepo.RepoPath())
	if err != nil {
		return fmt.Errorf("OpenRepository: %v", err)
	}

	// Add a temporary remote.
	tmpRemote := com.ToStr(time.Now().UnixNano())
	if err = headGitRepo.AddRemote(tmpRemote, RepoPath(pr.BaseRepo.MustOwner().Name, pr.BaseRepo.Name), true); err != nil {
		return fmt.Errorf("AddRemote: %v", err)
	}
	defer func() {
		headGitRepo.RemoveRemote(tmpRemote)
	}()
	remoteBranch := "remotes/" + tmpRemote + "/" + pr.BaseBranch
	pr.MergeBase, err = headGitRepo.GetMergeBase(remoteBranch, pr.HeadBranch)
	if err != nil {
		return fmt.Errorf("GetMergeBase: %v", err)
	} else if err = pr.Update(); err != nil {
		return fmt.Errorf("Update: %v", err)
	}

	patch, err := headGitRepo.GetPatch(pr.MergeBase, pr.HeadBranch)
	if err != nil {
		return fmt.Errorf("GetPatch: %v", err)
	}

	if err = pr.BaseRepo.SavePatch(pr.Index, patch); err != nil {
		return fmt.Errorf("BaseRepo.SavePatch: %v", err)
	}

	return nil
}

// PushToBaseRepo pushes commits from branches of head repository to
// corresponding branches of base repository.
// FIXME: Only push branches that are actually updates?
func (pr *PullRequest) PushToBaseRepo() (err error) {
	log.Trace("PushToBaseRepo[%d]: pushing commits to base repo 'refs/pull/%d/head'", pr.BaseRepoID, pr.Index)

	headRepoPath := pr.HeadRepo.RepoPath()
	headGitRepo, err := git.OpenRepository(headRepoPath)
	if err != nil {
		return fmt.Errorf("OpenRepository: %v", err)
	}

	tmpRemoteName := fmt.Sprintf("tmp-pull-%d", pr.ID)
	if err = headGitRepo.AddRemote(tmpRemoteName, pr.BaseRepo.RepoPath(), false); err != nil {
		return fmt.Errorf("headGitRepo.AddRemote: %v", err)
	}
	// Make sure to remove the remote even if the push fails
	defer headGitRepo.RemoveRemote(tmpRemoteName)

	headFile := fmt.Sprintf("refs/pull/%d/head", pr.Index)

	// Remove head in case there is a conflict.
	file := path.Join(pr.BaseRepo.RepoPath(), headFile)

	_ = os.Remove(file)

	if err = git.Push(headRepoPath, tmpRemoteName, fmt.Sprintf("%s:%s", pr.HeadBranch, headFile)); err != nil {
		return fmt.Errorf("Push: %v", err)
	}

	return nil
}

// AddToTaskQueue adds itself to pull request test task queue.
func (pr *PullRequest) AddToTaskQueue() {
	go pullRequestQueue.AddFunc(pr.ID, func() {
		pr.Status = PullRequestStatusChecking
		if err := pr.UpdateCols("status"); err != nil {
			log.Error(5, "AddToTaskQueue.UpdateCols[%d].(add to queue): %v", pr.ID, err)
		}
	})
}

// PullRequestList defines a list of pull requests
type PullRequestList []*PullRequest

func (prs PullRequestList) loadAttributes(e Engine) error {
	if len(prs) == 0 {
		return nil
	}

	// Load issues.
	issueIDs := make([]int64, 0, len(prs))
	for i := range prs {
		issueIDs = append(issueIDs, prs[i].IssueID)
	}
	issues := make([]*Issue, 0, len(issueIDs))
	if err := e.
		Where("id > 0").
		In("id", issueIDs).
		Find(&issues); err != nil {
		return fmt.Errorf("find issues: %v", err)
	}

	set := make(map[int64]*Issue)
	for i := range issues {
		set[issues[i].ID] = issues[i]
	}
	for i := range prs {
		prs[i].Issue = set[prs[i].IssueID]
	}
	return nil
}

// LoadAttributes load all the prs attributes
func (prs PullRequestList) LoadAttributes() error {
	return prs.loadAttributes(x)
}

func addHeadRepoTasks(prs []*PullRequest) {
	for _, pr := range prs {
		log.Trace("addHeadRepoTasks[%d]: composing new test task", pr.ID)
		if err := pr.UpdatePatch(); err != nil {
			log.Error(4, "UpdatePatch: %v", err)
			continue
		} else if err := pr.PushToBaseRepo(); err != nil {
			log.Error(4, "PushToBaseRepo: %v", err)
			continue
		}

		pr.AddToTaskQueue()
	}
}

// AddTestPullRequestTask adds new test tasks by given head/base repository and head/base branch,
// and generate new patch for testing as needed.
func AddTestPullRequestTask(doer *User, repoID int64, branch string, isSync bool) {
	log.Trace("AddTestPullRequestTask [head_repo_id: %d, head_branch: %s]: finding pull requests", repoID, branch)
	prs, err := GetUnmergedPullRequestsByHeadInfo(repoID, branch)
	if err != nil {
		log.Error(4, "Find pull requests [head_repo_id: %d, head_branch: %s]: %v", repoID, branch, err)
		return
	}

	if isSync {
		if err = PullRequestList(prs).LoadAttributes(); err != nil {
			log.Error(4, "PullRequestList.LoadAttributes: %v", err)
		}

		if err == nil {
			for _, pr := range prs {
				pr.Issue.PullRequest = pr
				if err = pr.Issue.LoadAttributes(); err != nil {
					log.Error(4, "LoadAttributes: %v", err)
					continue
				}
				if err = PrepareWebhooks(pr.Issue.Repo, HookEventPullRequest, &api.PullRequestPayload{
					Action:      api.HookIssueSynchronized,
					Index:       pr.Issue.Index,
					PullRequest: pr.Issue.PullRequest.APIFormat(),
					Repository:  pr.Issue.Repo.APIFormat(nil),
					Sender:      doer.APIFormat(),
				}); err != nil {
					log.Error(4, "PrepareWebhooks [pull_id: %v]: %v", pr.ID, err)
					continue
				}
				go HookQueue.Add(pr.Issue.Repo.ID)
			}
		}
	}

	addHeadRepoTasks(prs)

	log.Trace("AddTestPullRequestTask [base_repo_id: %d, base_branch: %s]: finding pull requests", repoID, branch)
	prs, err = GetUnmergedPullRequestsByBaseInfo(repoID, branch)
	if err != nil {
		log.Error(4, "Find pull requests [base_repo_id: %d, base_branch: %s]: %v", repoID, branch, err)
		return
	}
	for _, pr := range prs {
		pr.AddToTaskQueue()
	}
}

// ChangeUsernameInPullRequests changes the name of head_user_name
func ChangeUsernameInPullRequests(oldUserName, newUserName string) error {
	pr := PullRequest{
		HeadUserName: strings.ToLower(newUserName),
	}
	_, err := x.
		Cols("head_user_name").
		Where("head_user_name = ?", strings.ToLower(oldUserName)).
		Update(pr)
	return err
}

// checkAndUpdateStatus checks if pull request is possible to levaing checking status,
// and set to be either conflict or mergeable.
func (pr *PullRequest) checkAndUpdateStatus() {
	// Status is not changed to conflict means mergeable.
	if pr.Status == PullRequestStatusChecking {
		pr.Status = PullRequestStatusMergeable
	}

	// Make sure there is no waiting test to process before levaing the checking status.
	if !pullRequestQueue.Exist(pr.ID) {
		if err := pr.UpdateCols("status"); err != nil {
			log.Error(4, "Update[%d]: %v", pr.ID, err)
		}
	}
}

// TestPullRequests checks and tests untested patches of pull requests.
// TODO: test more pull requests at same time.
func TestPullRequests() {
	prs := make([]*PullRequest, 0, 10)
	x.Iterate(PullRequest{
		Status: PullRequestStatusChecking,
	},
		func(idx int, bean interface{}) error {
			pr := bean.(*PullRequest)

			if err := pr.GetBaseRepo(); err != nil {
				log.Error(3, "GetBaseRepo: %v", err)
				return nil
			}

			if err := pr.testPatch(); err != nil {
				log.Error(3, "testPatch: %v", err)
				return nil
			}
			prs = append(prs, pr)
			return nil
		})

	// Update pull request status.
	for _, pr := range prs {
		pr.checkAndUpdateStatus()
	}

	// Start listening on new test requests.
	for prID := range pullRequestQueue.Queue() {
		log.Trace("TestPullRequests[%v]: processing test task", prID)
		pullRequestQueue.Remove(prID)

		pr, err := GetPullRequestByID(com.StrTo(prID).MustInt64())
		if err != nil {
			log.Error(4, "GetPullRequestByID[%d]: %v", prID, err)
			continue
		} else if err = pr.testPatch(); err != nil {
			log.Error(4, "testPatch[%d]: %v", pr.ID, err)
			continue
		}

		pr.checkAndUpdateStatus()
	}
}

// InitTestPullRequests runs the task to test all the checking status pull requests
func InitTestPullRequests() {
	go TestPullRequests()
}
