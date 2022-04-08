// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	perm_model "code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
)

func TestRepoPermissionPublicNonOrgRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// public non-organization repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4}).(*repo_model.Repository)
	assert.NoError(t, repo.LoadUnits(db.DefaultContext))

	// plain user
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)
<<<<<<< HEAD
	perm, err := GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	perm, err := access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// change to collaborator
	assert.NoError(t, AddCollaborator(repo, user))
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	perm, err = access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// collaborator
	collaborator := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, collaborator)
=======
	perm, err = access_model.GetUserRepoPermission(repo, collaborator)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// owner
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, owner)
=======
	perm, err = access_model.GetUserRepoPermission(repo, owner)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// admin
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, admin)
=======
	perm, err = access_model.GetUserRepoPermission(repo, admin)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}
}

func TestRepoPermissionPrivateNonOrgRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// private non-organization repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2}).(*repo_model.Repository)
	assert.NoError(t, repo.LoadUnits(db.DefaultContext))

	// plain user
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4}).(*user_model.User)
<<<<<<< HEAD
	perm, err := GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	perm, err := access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.False(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// change to collaborator to default write access
	assert.NoError(t, AddCollaborator(repo, user))
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	perm, err = access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

<<<<<<< HEAD
	assert.NoError(t, ChangeCollaborationAccessMode(repo, user.ID, perm_model.AccessModeRead))
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	assert.NoError(t, repo_model.ChangeCollaborationAccessMode(repo, user.ID, perm_model.AccessModeRead))
	perm, err = access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// owner
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, owner)
=======
	perm, err = access_model.GetUserRepoPermission(repo, owner)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// admin
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, admin)
=======
	perm, err = access_model.GetUserRepoPermission(repo, admin)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}
}

func TestRepoPermissionPublicOrgRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// public organization repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 32}).(*repo_model.Repository)
	assert.NoError(t, repo.LoadUnits(db.DefaultContext))

	// plain user
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5}).(*user_model.User)
<<<<<<< HEAD
	perm, err := GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	perm, err := access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// change to collaborator to default write access
	assert.NoError(t, AddCollaborator(repo, user))
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	perm, err = access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

<<<<<<< HEAD
	assert.NoError(t, ChangeCollaborationAccessMode(repo, user.ID, perm_model.AccessModeRead))
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	assert.NoError(t, repo_model.ChangeCollaborationAccessMode(repo, user.ID, perm_model.AccessModeRead))
	perm, err = access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// org member team owner
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, owner)
=======
	perm, err = access_model.GetUserRepoPermission(repo, owner)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// org member team tester
	member := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 15}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, member)
=======
	perm, err = access_model.GetUserRepoPermission(repo, member)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
	}
	assert.True(t, perm.CanWrite(unit.TypeIssues))
	assert.False(t, perm.CanWrite(unit.TypeCode))

	// admin
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, admin)
=======
	perm, err = access_model.GetUserRepoPermission(repo, admin)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}
}

func TestRepoPermissionPrivateOrgRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// private organization repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 24}).(*repo_model.Repository)
	assert.NoError(t, repo.LoadUnits(db.DefaultContext))

	// plain user
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5}).(*user_model.User)
<<<<<<< HEAD
	perm, err := GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	perm, err := access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.False(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// change to collaborator to default write access
	assert.NoError(t, AddCollaborator(repo, user))
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	perm, err = access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

<<<<<<< HEAD
	assert.NoError(t, ChangeCollaborationAccessMode(repo, user.ID, perm_model.AccessModeRead))
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, user)
=======
	assert.NoError(t, repo_model.ChangeCollaborationAccessMode(repo, user.ID, perm_model.AccessModeRead))
	perm, err = access_model.GetUserRepoPermission(repo, user)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// org member team owner
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 15}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, owner)
=======
	perm, err = access_model.GetUserRepoPermission(repo, owner)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// update team information and then check permission
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 5}).(*organization.Team)
	err = organization.UpdateTeamUnits(team, nil)
	assert.NoError(t, err)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, owner)
=======
	perm, err = access_model.GetUserRepoPermission(repo, owner)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// org member team tester
	tester := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, tester)
=======
	perm, err = access_model.GetUserRepoPermission(repo, tester)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	assert.True(t, perm.CanWrite(unit.TypeIssues))
	assert.False(t, perm.CanWrite(unit.TypeCode))
	assert.False(t, perm.CanRead(unit.TypeCode))

	// org member team reviewer
	reviewer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, reviewer)
=======
	perm, err = access_model.GetUserRepoPermission(repo, reviewer)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	assert.False(t, perm.CanRead(unit.TypeIssues))
	assert.False(t, perm.CanWrite(unit.TypeCode))
	assert.True(t, perm.CanRead(unit.TypeCode))

	// admin
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1}).(*user_model.User)
<<<<<<< HEAD
	perm, err = GetUserRepoPermission(db.DefaultContext, repo, admin)
=======
	perm, err = access_model.GetUserRepoPermission(repo, admin)
>>>>>>> ce90db7ed (Move access and repo permission to models/perm/access)
	assert.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}
}
