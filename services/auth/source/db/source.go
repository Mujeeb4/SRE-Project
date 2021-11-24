// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package db

import (
	"code.gitea.io/gitea/models/login"
	user_model "code.gitea.io/gitea/models/user"
)

// Source is a password authentication service
type Source struct{}

// FromDB fills up an OAuth2Config from serialized format.
func (source *Source) FromDB(_ []byte) error {
	return nil
}

// ToDB exports an SMTPConfig to a serialized format.
func (source *Source) ToDB() ([]byte, error) {
	return nil, nil
}

// Authenticate queries if login/password is valid against the PAM,
// and create a local user if success when enabled.
func (source *Source) Authenticate(user *user_model.User, login, password string) (*user_model.User, error) {
	return Authenticate(user, login, password)
}

func init() {
	login.RegisterTypeConfig(login.NoType, &Source{})
	login.RegisterTypeConfig(login.Plain, &Source{})
}
