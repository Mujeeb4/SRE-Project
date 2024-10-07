// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Package private contains all internal routes. The package name "internal" isn't usable because Golang reserves it for disabling cross-package usage.
package private

import (
	"net/http"
	"strings"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/private"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/common"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/lfs"

	"gitea.com/go-chi/binding"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
)

// CheckInternalToken check internal token is set
func CheckInternalToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		tokens := req.Header.Get("Authorization")
		fields := strings.SplitN(tokens, " ", 2)
		if setting.InternalToken == "" {
			log.Warn(`The INTERNAL_TOKEN setting is missing from the configuration file: %q, internal API can't work.`, setting.CustomConf)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if len(fields) != 2 || fields[0] != "Bearer" || fields[1] != setting.InternalToken {
			log.Debug("Forbidden attempt to access internal url: Authorization header: %s", tokens)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		} else {
			next.ServeHTTP(w, req)
		}
	})
}

// bind binding an obj to a handler
func bind[T any](_ T) any {
	return func(ctx *context.PrivateContext) {
		theObj := new(T) // create a new form obj for every request but not use obj directly
		binding.Bind(ctx.Req, theObj)
		web.SetForm(ctx, theObj)
	}
}

// SwapAuthToken swaps Authorization header with X-Auth header
func swapAuthToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.Header.Set("Authorization", req.Header.Get("X-Auth"))
		next.ServeHTTP(w, req)
	})
}

// Routes registers all internal APIs routes to web application.
// These APIs will be invoked by internal commands for example `gitea serv` and etc.
func Routes() *web.Router {
	r := web.NewRouter()
	r.Use(context.PrivateContexter())
	r.Use(CheckInternalToken)
	// Log the real ip address of the request from SSH is really helpful for diagnosing sometimes.
	// Since internal API will be sent only from Gitea sub commands and it's under control (checked by InternalToken), we can trust the headers.
	r.Use(chi_middleware.RealIP)

	r.Post("/ssh/authorized_keys", AuthorizedPublicKeyByContent)
	r.Post("/ssh/{id}/update/{repoid}", UpdatePublicKeyInRepo)
	r.Post("/ssh/log", bind(private.SSHLogOption{}), SSHLog)
	r.Post("/hook/pre-receive/{owner}/{repo}", RepoAssignment, bind(private.HookOptions{}), HookPreReceive)
	r.Post("/hook/post-receive/{owner}/{repo}", context.OverrideContext, bind(private.HookOptions{}), HookPostReceive)
	r.Post("/hook/proc-receive/{owner}/{repo}", context.OverrideContext, RepoAssignment, bind(private.HookOptions{}), HookProcReceive)
	r.Post("/hook/set-default-branch/{owner}/{repo}/{branch}", RepoAssignment, SetDefaultBranch)
	r.Get("/serv/none/{keyid}", ServNoCommand)
	r.Get("/serv/command/{keyid}/{owner}/{repo}", ServCommand)
	r.Post("/manager/shutdown", Shutdown)
	r.Post("/manager/restart", Restart)
	r.Post("/manager/reload-templates", ReloadTemplates)
	r.Post("/manager/flush-queues", bind(private.FlushOptions{}), FlushQueues)
	r.Post("/manager/pause-logging", PauseLogging)
	r.Post("/manager/resume-logging", ResumeLogging)
	r.Post("/manager/release-and-reopen-logging", ReleaseReopenLogging)
	r.Post("/manager/set-log-sql", SetLogSQL)
	r.Post("/manager/add-logger", bind(private.LoggerOptions{}), AddLogger)
	r.Post("/manager/remove-logger/{logger}/{writer}", RemoveLogger)
	r.Get("/manager/processes", Processes)
	r.Post("/mail/send", SendEmail)
	r.Post("/restore_repo", RestoreRepo)
	r.Post("/actions/generate_actions_runner_token", GenerateActionsRunnerToken)

	r.Group("/repo/{username}/{reponame}", func() {
		r.Group("/info/lfs", func() {
			r.Post("/objects/batch", lfs.CheckAcceptMediaType, lfs.BatchHandler)
			r.Put("/objects/{oid}/{size}", lfs.UploadHandler)
			r.Get("/objects/{oid}/{filename}", lfs.DownloadHandler)
			r.Get("/objects/{oid}", lfs.DownloadHandler)
			r.Post("/verify", lfs.CheckAcceptMediaType, lfs.VerifyHandler)
			r.Group("/locks", func() {
				r.Get("/", lfs.GetListLockHandler)
				r.Post("/", lfs.PostLockHandler)
				r.Post("/verify", lfs.VerifyLockHandler)
				r.Post("/{lid}/unlock", lfs.UnLockHandler)
			}, lfs.CheckAcceptMediaType)
			r.Any("/*", func(ctx *context.Context) {
				ctx.NotFound("", nil)
			})
		}, swapAuthToken)
	}, common.Sessioner(), context.Contexter())
	// end "/repo/{username}/{reponame}": git (LFS) API mirror

	return r
}
