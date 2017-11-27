// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import "code.gitea.io/gitea/modules/setting"

// ___________.__             ___________                     __
// \__    ___/|__| _____   ___\__    ___/___________    ____ |  | __ ___________
// |    |   |  |/     \_/ __ \|    |  \_  __ \__  \ _/ ___\|  |/ // __ \_  __ \
// |    |   |  |  Y Y  \  ___/|    |   |  | \// __ \\  \___|    <\  ___/|  | \/
// |____|   |__|__|_|  /\___  >____|   |__|  (____  /\___  >__|_ \\___  >__|
// \/     \/                    \/     \/     \/    \/

// IsTimetrackerEnabled returns whether or not the timetracker is enabled. It returns the default value from config if an error occurs.
func (repo *Repository) IsTimetrackerEnabled() bool {
	var u *RepoUnit
	var err error
	if u, err = repo.GetUnit(UnitTypeIssues); err != nil {
		return setting.Service.DefaultEnableTimetracking
	}
	return u.IssuesConfig().EnableTimetracker
}

// AllowOnlyContributorsToTrackTime returns value of IssuesConfig or the default value
func (repo *Repository) AllowOnlyContributorsToTrackTime() bool {
	var u *RepoUnit
	var err error
	if u, err = repo.GetUnit(UnitTypeIssues); err != nil {
		return setting.Service.DefaultAllowOnlyContributorsToTrackTime
	}
	return u.IssuesConfig().AllowOnlyContributorsToTrackTime
}

/*
________                                   .___                   .__
\______ \   ____ ______   ____   ____    __| _/____   ____   ____ |__| ____   ______
 |    |  \_/ __ \\____ \_/ __ \ /    \  / __ |/ __ \ /    \_/ ___\|  |/ __ \ /  ___/
 |    `   \  ___/|  |_> >  ___/|   |  \/ /_/ \  ___/|   |  \  \___|  \  ___/ \___ \
/_______  /\___  >   __/ \___  >___|  /\____ |\___  >___|  /\___  >__|\___  >____  >
        \/     \/|__|        \/     \/      \/    \/     \/     \/        \/     \/
*/

// IsDependenciesEnabled returns if dependecies are enabled and returns the default setting if not set.
func (repo *Repository) IsDependenciesEnabled() bool {
	var u *RepoUnit
	var err error
	if u, err = repo.GetUnit(UnitTypeIssues); err != nil {
		return setting.Service.DefaultEnableDependencies
	}
	return u.IssuesConfig().EnableDependencies
}
