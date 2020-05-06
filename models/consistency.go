// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"xorm.io/builder"
)

// consistencyCheckable a type that can be tested for database consistency
type consistencyCheckable interface {
	checkForConsistency(t *testing.T)
}

// CheckConsistencyForAll test that the entire database is consistent
func CheckConsistencyForAll(t *testing.T) {
	CheckConsistencyFor(t,
		&User{},
		&Repository{},
		&Issue{},
		&PullRequest{},
		&Milestone{},
		&Label{},
		&Team{},
		&Action{})
}

// CheckConsistencyFor test that all matching database entries are consistent
func CheckConsistencyFor(t *testing.T, beansToCheck ...interface{}) {
	for _, bean := range beansToCheck {
		sliceType := reflect.SliceOf(reflect.TypeOf(bean))
		sliceValue := reflect.MakeSlice(sliceType, 0, 10)

		ptrToSliceValue := reflect.New(sliceType)
		ptrToSliceValue.Elem().Set(sliceValue)

		assert.NoError(t, x.Table(bean).Find(ptrToSliceValue.Interface()))
		sliceValue = ptrToSliceValue.Elem()

		for i := 0; i < sliceValue.Len(); i++ {
			entity := sliceValue.Index(i).Interface()
			checkable, ok := entity.(consistencyCheckable)
			if !ok {
				t.Errorf("Expected %+v (of type %T) to be checkable for consistency",
					entity, entity)
			} else {
				checkable.checkForConsistency(t)
			}
		}
	}
}

// getCount get the count of database entries matching bean
func getCount(t *testing.T, e Engine, bean interface{}) int64 {
	count, err := e.Count(bean)
	assert.NoError(t, err)
	return count
}

// assertCount test the count of database entries matching bean
func assertCount(t *testing.T, bean interface{}, expected int) {
	assert.EqualValues(t, expected, getCount(t, x, bean),
		"Failed consistency test, the counted bean (of type %T) was %+v", bean, bean)
}

func (user *User) checkForConsistency(t *testing.T) {
	assertCount(t, &Repository{OwnerID: user.ID}, user.NumRepos)
	assertCount(t, &Star{UID: user.ID}, user.NumStars)
	assertCount(t, &OrgUser{OrgID: user.ID}, user.NumMembers)
	assertCount(t, &Team{OrgID: user.ID}, user.NumTeams)
	assertCount(t, &Follow{UserID: user.ID}, user.NumFollowing)
	assertCount(t, &Follow{FollowID: user.ID}, user.NumFollowers)
	if user.Type != UserTypeOrganization {
		assert.EqualValues(t, 0, user.NumMembers)
		assert.EqualValues(t, 0, user.NumTeams)
	}
}

func (repo *Repository) checkForConsistency(t *testing.T) {
	assert.Equal(t, repo.LowerName, strings.ToLower(repo.Name), "repo: %+v", repo)
	assertCount(t, &Star{RepoID: repo.ID}, repo.NumStars)
	assertCount(t, &Milestone{RepoID: repo.ID}, repo.NumMilestones)
	assertCount(t, &Repository{ForkID: repo.ID}, repo.NumForks)
	if repo.IsFork {
		AssertExistsAndLoadBean(t, &Repository{ID: repo.ForkID})
	}

	actual := getCount(t, x.Where("Mode<>?", RepoWatchModeDont), &Watch{RepoID: repo.ID})
	assert.EqualValues(t, repo.NumWatches, actual,
		"Unexpected number of watches for repo %+v", repo)

	actual = getCount(t, x.Where("is_pull=?", false), &Issue{RepoID: repo.ID})
	assert.EqualValues(t, repo.NumIssues, actual,
		"Unexpected number of issues for repo %+v", repo)

	actual = getCount(t, x.Where("is_pull=? AND is_closed=?", false, true), &Issue{RepoID: repo.ID})
	assert.EqualValues(t, repo.NumClosedIssues, actual,
		"Unexpected number of closed issues for repo %+v", repo)

	actual = getCount(t, x.Where("is_pull=?", true), &Issue{RepoID: repo.ID})
	assert.EqualValues(t, repo.NumPulls, actual,
		"Unexpected number of pulls for repo %+v", repo)

	actual = getCount(t, x.Where("is_pull=? AND is_closed=?", true, true), &Issue{RepoID: repo.ID})
	assert.EqualValues(t, repo.NumClosedPulls, actual,
		"Unexpected number of closed pulls for repo %+v", repo)

	actual = getCount(t, x.Where("is_closed=?", true), &Milestone{RepoID: repo.ID})
	assert.EqualValues(t, repo.NumClosedMilestones, actual,
		"Unexpected number of closed milestones for repo %+v", repo)
}

func (issue *Issue) checkForConsistency(t *testing.T) {
	actual := getCount(t, x.Where("type=?", CommentTypeComment), &Comment{IssueID: issue.ID})
	assert.EqualValues(t, issue.NumComments, actual,
		"Unexpected number of comments for issue %+v", issue)
	if issue.IsPull {
		pr := AssertExistsAndLoadBean(t, &PullRequest{IssueID: issue.ID}).(*PullRequest)
		assert.EqualValues(t, pr.Index, issue.Index)
	}
}

func (pr *PullRequest) checkForConsistency(t *testing.T) {
	issue := AssertExistsAndLoadBean(t, &Issue{ID: pr.IssueID}).(*Issue)
	assert.True(t, issue.IsPull)
	assert.EqualValues(t, issue.Index, pr.Index)
}

func (milestone *Milestone) checkForConsistency(t *testing.T) {
	assertCount(t, &Issue{MilestoneID: milestone.ID}, milestone.NumIssues)

	actual := getCount(t, x.Where("is_closed=?", true), &Issue{MilestoneID: milestone.ID})
	assert.EqualValues(t, milestone.NumClosedIssues, actual,
		"Unexpected number of closed issues for milestone %+v", milestone)
}

func (label *Label) checkForConsistency(t *testing.T) {
	issueLabels := make([]*IssueLabel, 0, 10)
	assert.NoError(t, x.Find(&issueLabels, &IssueLabel{LabelID: label.ID}))
	assert.EqualValues(t, label.NumIssues, len(issueLabels),
		"Unexpected number of issue for label %+v", label)

	issueIDs := make([]int64, len(issueLabels))
	for i, issueLabel := range issueLabels {
		issueIDs[i] = issueLabel.IssueID
	}

	expected := int64(0)
	if len(issueIDs) > 0 {
		expected = getCount(t, x.In("id", issueIDs).Where("is_closed=?", true), &Issue{})
	}
	assert.EqualValues(t, expected, label.NumClosedIssues,
		"Unexpected number of closed issues for label %+v", label)
}

func (team *Team) checkForConsistency(t *testing.T) {
	assertCount(t, &TeamUser{TeamID: team.ID}, team.NumMembers)
	assertCount(t, &TeamRepo{TeamID: team.ID}, team.NumRepos)
}

func (action *Action) checkForConsistency(t *testing.T) {
	repo := AssertExistsAndLoadBean(t, &Repository{ID: action.RepoID}).(*Repository)
	assert.Equal(t, repo.IsPrivate, action.IsPrivate, "action: %+v", action)
}

// CountCorruptLabels return count of labels witch are broken and not accessible via ui anymore
func CountCorruptLabels() (int64, error) {
	noref, err := x.Table("label").Where("repo_id=? AND org_id=?", 0, 0).Count("label.id")
	if err != nil {
		return 0, err
	}
	norepo, err := x.Table("label").
		Join("LEFT", "repository", "label.repo_id=repository.id").
		Where("repository.id is NULL AND label.repo_id > 0").
		Count("id")
	if err != nil {
		return 0, err
	}
	noorg, err := x.Table("label").
		Join("LEFT", "repository", "label.repo_id=user.id").
		Where("user.id is NULL AND label.org_id > 0").
		Count("id")
	if err != nil {
		return 0, err
	}
	if err != nil {
		return 0, err
	}

	return noref + norepo + noorg, nil
}

// DeleteCorruptLabels delete labels witch are broken and not accessible via ui anymore
func DeleteCorruptLabels() error {
	// delete labels with no reference
	if _, err := x.Table("label").Where("repo_id=? AND org_id=?", 0, 0).Delete(new(Label)); err != nil {
		return err
	}

	// delete labels with none existing repos
	if _, err := x.In("id", builder.Select("label.id").
		From("label").
		Join("LEFT", "repository", "label.repo_id=repository.id").
		Where(builder.IsNull{"repository.id"}).And(builder.Gt{"label.repo_id": 0})).
		Delete(Label{}); err != nil {
		return err
	}

	// delete labels with none existing orgs
	if _, err := x.In("id", builder.Select("label.id").
		From("label").
		Join("LEFT", "user", "label.org_id=user.id").
		Where(builder.IsNull{"user.id"}).And(builder.Gt{"label.org_id": 0})).
		Delete(Label{}); err != nil {
		return err
	}

	return nil
}

// CountCorruptIssues count issues without a repo
func CountCorruptIssues() (int64, error) {
	return x.Table("issue").
		Join("LEFT", "repository", "issue.repo_id=repository.id").
		Where("repository.id is NULL").
		Count("id")
}

// DeleteCorruptIssues delete issues without a repo
func DeleteCorruptIssues() error {
	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}

	var ids []int64
	var err error

	if err = sess.Table("issue").Select("issue.id").
		Join("LEFT", "repository", "issue.repo_id=repository.id").
		Where("repository.id is NULL").
		Find(&ids); err != nil {
		return err
	}

	deleteCond := builder.Select("id").From("issue").Where(builder.In("id", ids))
	var attachmentPaths []string
	if attachmentPaths, err = deleteIssuesByBuilder(sess, deleteCond); err != nil {
		return err
	}

	if _, err = sess.In("id", ids).Delete(&Issue{}); err != nil {
		return err
	}

	if err = sess.Commit(); err != nil {
		return err
	}

	// Remove issue attachment files.
	for i := range attachmentPaths {
		removeAllWithNotice(x, "Delete issue attachment", attachmentPaths[i])
	}
	return nil
}

// CountCorruptObject count subjects with have no existing refobject anymore
func CountCorruptObject(subject, refobject, joinCond string) (int64, error) {
	return x.Table(subject).
		Join("LEFT", refobject, joinCond).
		Where(refobject + ".id is NULL").
		Count("id")
}

// DeleteCorruptObject delete subjects with have no existing refobject anymore
func DeleteCorruptObject(subject, refobject, joinCond string) error {
	_, err := x.In("id", builder.Select(subject+".id").
		From(subject).
		Join("LEFT", refobject, joinCond).
		Where(builder.IsNull{refobject + ".id"})).
		Delete(subject)
	return err
}
