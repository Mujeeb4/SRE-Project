// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package sso

import (
	"reflect"
	"sort"
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"

	"gitea.com/macaron/macaron"
	"gitea.com/macaron/session"
)

var (
	ssoMethods []SingleSignOn
)

// Methods returns the instances of all registered SSO methods
func Methods() []SingleSignOn {
	return ssoMethods
}

// MethodsByPriority returns the instances of all registered SSO methods, ordered by ascending priority
func MethodsByPriority() []SingleSignOn {
	methods := Methods()
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Priority() < methods[j].Priority()
	})
	return methods
}

// Register adds the specified instance to the list of available SSO methods
func Register(method SingleSignOn) {
	ssoMethods = append(ssoMethods, method)
}

// Init should be called exactly once when the application starts to allow SSO plugins
// to allocate necessary resources
func Init() {
	for _, method := range Methods() {
		err := method.Init()
		if err != nil {
			log.Error("Could not initialize '%s' SSO method, error: %s", reflect.TypeOf(method).String(), err)
		}
	}
}

// Free should be called exactly once when the application is terminating to allow SSO plugins
// to release necessary resources
func Free() {
	for _, method := range Methods() {
		err := method.Free()
		if err != nil {
			log.Error("Could not free '%s' SSO method, error: %s", reflect.TypeOf(method).String(), err)
		}
	}
}

// SessionUser returns the user object corresponding to the "uid" session variable.
func SessionUser(sess session.Store) *models.User {
	// Get user ID
	uid := sess.Get("uid")
	if uid == nil {
		return nil
	}
	id, ok := uid.(int64)
	if !ok {
		return nil
	}

	// Get user object
	user, err := models.GetUserByID(id)
	if err != nil {
		if !models.IsErrUserNotExist(err) {
			log.Error("GetUserById: %v", err)
		}
		return nil
	}
	return user
}

// isAPIPath returns true if the specified URL is an API path
func isAPIPath(ctx *macaron.Context) bool {
	return strings.HasPrefix(ctx.Req.URL.Path, "/api/")
}

// isAttachmentDownload check if request is a file download (GET) with URL to an attachment
func isAttachmentDownload(ctx *macaron.Context) bool {
	return strings.HasPrefix(ctx.Req.URL.Path, "/attachments/") && ctx.Req.Method == "GET"
}
