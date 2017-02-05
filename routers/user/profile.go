// Copyright 2015 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package user

import (
	"fmt"
	"path"
	"strings"

	"github.com/Unknwon/paginater"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers/repo"
)

const (
	tplFollowers base.TplName = "user/meta/followers"
	tplStars     base.TplName = "user/meta/stars"
)

// GetUserByName get user by name
func GetUserByName(ctx *context.Context, name string) *models.User {
	user, err := models.GetUserByName(name)
	if err != nil {
		if models.IsErrUserNotExist(err) {
			ctx.Handle(404, "GetUserByName", nil)
		} else {
			ctx.Handle(500, "GetUserByName", err)
		}
		return nil
	}
	return user
}

// GetUserByParams returns user whose name is presented in URL paramenter.
func GetUserByParams(ctx *context.Context) *models.User {
	return GetUserByName(ctx, ctx.Params(":username"))
}

// Profile render user's profile page
func Profile(ctx *context.Context) {
	uname := ctx.Params(":username")
	// Special handle for FireFox requests favicon.ico.
	if uname == "favicon.ico" {
		ctx.ServeFile(path.Join(setting.StaticRootPath, "public/img/favicon.png"))
		return
	} else if strings.HasSuffix(uname, ".png") {
		ctx.Error(404)
		return
	}

	isShowKeys := false
	if strings.HasSuffix(uname, ".keys") {
		isShowKeys = true
	}

	ctxUser := GetUserByName(ctx, strings.TrimSuffix(uname, ".keys"))
	if ctx.Written() {
		return
	}

	// Show SSH keys.
	if isShowKeys {
		ShowSSHKeys(ctx, ctxUser.ID)
		return
	}

	if ctxUser.IsOrganization() {
		showOrgProfile(ctx)
		return
	}

	ctx.Data["Title"] = ctxUser.DisplayName()
	ctx.Data["PageIsUserProfile"] = true
	ctx.Data["Owner"] = ctxUser
	showPrivate := ctx.IsSigned && (ctx.User.IsAdmin || ctx.User.ID == ctxUser.ID)

	orgs, err := models.GetOrgsByUserID(ctxUser.ID, showPrivate)
	if err != nil {
		ctx.Handle(500, "GetOrgsByUserIDDesc", err)
		return
	}

	ctx.Data["Orgs"] = orgs

	tab := ctx.Query("tab")
	ctx.Data["TabName"] = tab
	switch tab {
	case "activity":
		retrieveFeeds(ctx, ctxUser, -1, 0, !showPrivate)
		if ctx.Written() {
			return
		}
	case "stars":
		page := ctx.QueryInt("page")
		if page <= 0 {
			page = 1
		}

		repos, err := ctxUser.GetStarredRepos(showPrivate, page, setting.UI.User.RepoPagingNum, "")
		if err != nil {
			ctx.Handle(500, "GetStarredRepos", err)
			return
		}

		counts, err := ctxUser.GetStarredRepoCount(showPrivate)
		if err != nil {
			ctx.Handle(500, "GetStarredRepoCount", err)
			return
		}

		ctx.Data["Repos"] = repos
		ctx.Data["Page"] = paginater.New(int(counts), setting.UI.User.RepoPagingNum, page, 5)
		ctx.Data["Total"] = int(counts)
		ctx.Data["Tabs"] = "stars"
	default:
		page := ctx.QueryInt("page")
		if page <= 0 {
			page = 1
		}

		var (
			repos   []*models.Repository
			count   int64
			err     error
			orderBy string
		)
		switch ctx.Query("sort") {
		case "newest":
			orderBy = "created_unix DESC"
		case "oldest":
			orderBy = "created_unix ASC"
		case "recentupdate":
			orderBy = "updated_unix DESC"
		case "leastupdate":
			orderBy = "updated_unix ASC"
		case "reversealphabetically":
			orderBy = "name DESC"
		case "alphabetically":
			orderBy = "name ASC"
		default:
			orderBy = "updated_unix DESC"
		}

		keyword := ctx.Query("q")
		if len(keyword) == 0 {
			repos, err = models.GetUserRepositories(ctxUser.ID, showPrivate, page, setting.UI.User.RepoPagingNum, orderBy)
			if err != nil {
				ctx.Handle(500, "GetRepositories", err)
				return
			}
			ctx.Data["Repos"] = repos
			ctx.Data["Page"] = paginater.New(ctxUser.NumRepos, setting.UI.User.RepoPagingNum, page, 5)
			ctx.Data["Total"] = ctxUser.NumRepos
		} else {
			repos, count, err = models.SearchRepositoryByName(&models.SearchRepoOptions{
				Keyword:  keyword,
				OwnerID:  ctxUser.ID,
				OrderBy:  orderBy,
				Private:  ctx.IsSigned && ctx.User.ID == ctxUser.ID,
				Page:     page,
				PageSize: setting.UI.User.RepoPagingNum,
			})
			if err != nil {
				ctx.Handle(500, "SearchRepositoryByName", err)
				return
			}

			ctx.Data["Repos"] = repos
			ctx.Data["Page"] = paginater.New(int(count), setting.UI.User.RepoPagingNum, page, 5)
			ctx.Data["Total"] = count
		}

		// set default sort value.
		if ctx.Query("sort") == "" {
			ctx.Data["SortType"] = "recentupdate"
		} else {
			ctx.Data["SortType"] = ctx.Query("sort")
		}

		ctx.Data["Keyword"] = keyword
	}

	ctx.HTML(200, tplProfile)
}

// Followers render user's followers page
func Followers(ctx *context.Context) {
	u := GetUserByParams(ctx)
	if ctx.Written() {
		return
	}
	ctx.Data["Title"] = u.DisplayName()
	ctx.Data["CardsTitle"] = ctx.Tr("user.followers")
	ctx.Data["PageIsFollowers"] = true
	ctx.Data["Owner"] = u
	repo.RenderUserCards(ctx, u.NumFollowers, u.GetFollowers, tplFollowers)
}

// Following render user's followering page
func Following(ctx *context.Context) {
	u := GetUserByParams(ctx)
	if ctx.Written() {
		return
	}
	ctx.Data["Title"] = u.DisplayName()
	ctx.Data["CardsTitle"] = ctx.Tr("user.following")
	ctx.Data["PageIsFollowing"] = true
	ctx.Data["Owner"] = u
	repo.RenderUserCards(ctx, u.NumFollowing, u.GetFollowing, tplFollowers)
}

// Action response for follow/unfollow user request
func Action(ctx *context.Context) {
	u := GetUserByParams(ctx)
	if ctx.Written() {
		return
	}

	var err error
	switch ctx.Params(":action") {
	case "follow":
		err = models.FollowUser(ctx.User.ID, u.ID)
	case "unfollow":
		err = models.UnfollowUser(ctx.User.ID, u.ID)
	}

	if err != nil {
		ctx.Handle(500, fmt.Sprintf("Action (%s)", ctx.Params(":action")), err)
		return
	}

	redirectTo := ctx.Query("redirect_to")
	if len(redirectTo) == 0 {
		redirectTo = u.HomeLink()
	}
	ctx.Redirect(redirectTo)
}
