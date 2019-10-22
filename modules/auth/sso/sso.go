package sso

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

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
		if !method.IsEnabled() {
			continue
		}
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
		if !method.IsEnabled() {
			continue
		}
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
func isAPIPath(url string) bool {
	return strings.HasPrefix(url, "/api/")
}

// isPublicResource checks if the url is of a public resource file that should be served
// without authentication (eg. the Web App Manifest, the Service Worker script or the favicon)
func isPublicResource(ctx *macaron.Context) bool {
	path := strings.TrimSuffix(ctx.Req.URL.Path, "/")
	return path == "/robots.txt" ||
		path == "/favicon.ico" ||
		path == "/favicon.png" ||
		path == "/manifest.json" ||
		path == "/serviceworker.js"
}

// isPublicPage checks if the url is of a public page that should not require authentication
func isPublicPage(ctx *macaron.Context) bool {
	path := strings.TrimSuffix(ctx.Req.URL.Path, "/")
	homePage := strings.TrimSuffix(setting.AppSubURL, "/")
	currentURL := homePage + path
	return currentURL == homePage ||
		path == "/user/login" ||
		path == "/user/login/openid" ||
		path == "/user/sign_up" ||
		path == "/user/forgot_password" ||
		path == "/user/openid/connect" ||
		path == "/user/openid/register" ||
		strings.HasPrefix(path, "/user/oauth2") ||
		path == "/user/link_account" ||
		path == "/user/link_account_signin" ||
		path == "/user/link_account_signup" ||
		path == "/user/two_factor" ||
		path == "/user/two_factor/scratch" ||
		path == "/user/u2f" ||
		path == "/user/u2f/challenge" ||
		path == "/user/u2f/sign" ||
		(!setting.Service.RequireSignInView && (path == "/explore/repos" ||
			path == "/explore/users" ||
			path == "/explore/organizations" ||
			path == "/explore/code"))
}

func handleSignIn(ctx *macaron.Context, sess session.Store, user *models.User) {
	_ = sess.Delete("openid_verified_uri")
	_ = sess.Delete("openid_signin_remember")
	_ = sess.Delete("openid_determined_email")
	_ = sess.Delete("openid_determined_username")
	_ = sess.Delete("twofaUid")
	_ = sess.Delete("twofaRemember")
	_ = sess.Delete("u2fChallenge")
	_ = sess.Delete("linkAccount")
	err := sess.Set("uid", user.ID)
	if err != nil {
		log.Error(fmt.Sprintf("Error setting session: %v", err))
	}
	err = sess.Set("uname", user.Name)
	if err != nil {
		log.Error(fmt.Sprintf("Error setting session: %v", err))
	}

	// Language setting of the user overwrites the one previously set
	// If the user does not have a locale set, we save the current one.
	if len(user.Language) == 0 {
		user.Language = ctx.Locale.Language()
		if err := models.UpdateUserCols(user, "language"); err != nil {
			log.Error(fmt.Sprintf("Error updating user language [user: %d, locale: %s]", user.ID, user.Language))
			return
		}
	}

	ctx.SetCookie("lang", user.Language, nil, setting.AppSubURL, setting.SessionConfig.Domain, setting.SessionConfig.Secure, true)

	// Clear whatever CSRF has right now, force to generate a new one
	ctx.SetCookie(setting.CSRFCookieName, "", -1, setting.AppSubURL, setting.SessionConfig.Domain, setting.SessionConfig.Secure, true)
}

// addFlashErr adds an error message to the Flash object mapped to a macaron.Context
func addFlashErr(ctx *macaron.Context, err string) {
	fv := ctx.GetVal(reflect.TypeOf(&session.Flash{}))
	if !fv.IsValid() {
		return
	}
	flash, ok := fv.Interface().(*session.Flash)
	if !ok {
		return
	}
	flash.Error(err)
	ctx.Data["Flash"] = flash
}
