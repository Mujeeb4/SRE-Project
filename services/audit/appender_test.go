// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package audit

import (
	"strings"
	"testing"
	"time"

	repository_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
)

func TestWriteEventAsJSON(t *testing.T) {
	r := &repository_model.Repository{ID: 3, Name: "TestRepo", OwnerName: "TestUser"}
	m := &repository_model.PushMirror{ID: 4}
	doer := &user_model.User{ID: 2, Name: "Doer"}

	e := BuildEvent(
		RepositoryMirrorPushAdd,
		doer,
		r,
		m,
		"Added push mirror for repository %s.",
		r.FullName(),
	)
	e.Time = time.Time{}

	sb := strings.Builder{}
	assert.NoError(t, WriteEventAsJSON(&sb, e))
	assert.Equal(
		t,
		`{"action":"repository:mirror:push:add","doer":{"type":"user","primary_key":2,"friendly_name":"Doer"},"scope":{"type":"repository","primary_key":3,"friendly_name":"TestUser/TestRepo"},"target":{"type":"push_mirror","primary_key":4,"friendly_name":""},"message":"Added push mirror for repository TestUser/TestRepo.","time":"0001-01-01T00:00:00Z"}`+"\n",
		sb.String(),
	)
}
