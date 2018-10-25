// Copyright 2018 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"testing"

	"code.gitea.io/gitea/modules/setting"

	"github.com/stretchr/testify/assert"
)

func TestRegisterForm_IsDomainWhiteList_Empty(t *testing.T) {
	_ = setting.Service

	setting.Service.EmailDomainWhitelist = []string{}

	form := RegisterForm{}

	assert.True(t, form.IsEmaildomainwhitelisted())
}

func TestRegisterForm_IsDomainWhiteList_InvalidEmail(t *testing.T) {
	_ = setting.Service

	setting.Service.EmailDomainWhitelist = []string{"gitea.io"}

	tt := []struct {
		email string
	}{
		{"securitygieqqq"},
		{"hdudhdd"},
	}

	for _, v := range tt {
		form := RegisterForm{Email: v.email}

		assert.False(t, form.IsEmaildomainwhitelisted())
	}
}

func TestRegisterForm_IsDomainWhiteList_ValidEmail(t *testing.T) {
	_ = setting.Service

	setting.Service.EmailDomainWhitelist = []string{"gitea.io"}

	tt := []struct {
		email string
		valid bool
	}{
		{"security@gitea.io", true},
		{"hdudhdd", false},
		{"seee@example.com", false},
	}

	for _, v := range tt {
		form := RegisterForm{Email: v.email}

		assert.Equal(t, v.valid, form.IsEmaildomainwhitelisted())
	}
}
