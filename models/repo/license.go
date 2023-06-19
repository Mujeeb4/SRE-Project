// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"context"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(RepoLicense))
}

type RepoLicense struct { //revive:disable-line:exported
	ID          int64 `xorm:"pk autoincr"`
	RepoID      int64 `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CommitID    string
	License     string             `xorm:"VARCHAR(50) UNIQUE(s) INDEX NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX CREATED"`
	UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
}

// LanguageStatList defines a list of language statistics
type RepoLicenseList []*RepoLicense //revive:disable-line:exported

func (rll RepoLicenseList) StringList() []string {
	var licenses []string
	for _, rl := range rll {
		licenses = append(licenses, rl.License)
	}
	return licenses
}

// GetRepoLicenses returns the license statistics for a repository
func GetRepoLicenses(ctx context.Context, repo *Repository) (RepoLicenseList, error) {
	licenses := make(RepoLicenseList, 0)
	if err := db.GetEngine(ctx).Where("`repo_id` = ?", repo.ID).Asc("`license`").Find(&licenses); err != nil {
		return nil, err
	}
	return licenses, nil
}

// UpdateRepoLicenses updates the license statistics for repository
func UpdateRepoLicenses(ctx context.Context, repo *Repository, commitID string, licenses []string) error {
	oldLicenses, err := GetRepoLicenses(ctx, repo)
	if err != nil {
		return err
	}
	for _, license := range licenses {
		upd := false
		for _, o := range oldLicenses {
			// Update already existing license
			if o.License == license {
				if _, err := db.GetEngine(ctx).ID(o.ID).Cols("`commit_id`").Update(o); err != nil {
					return err
				}
				upd = true
				break
			}
		}
		// Insert new license
		if !upd {
			if err := db.Insert(ctx, &RepoLicense{
				RepoID:   repo.ID,
				CommitID: commitID,
				License:  license,
			}); err != nil {
				return err
			}
		}
	}
	// Delete old languages
	licenseToDelete := make([]int64, 0, len(oldLicenses))
	for _, o := range oldLicenses {
		if o.CommitID != commitID {
			licenseToDelete = append(licenseToDelete, o.ID)
		}
	}
	if len(licenseToDelete) > 0 {
		if _, err := db.GetEngine(ctx).In("`id`", licenseToDelete).Delete(&RepoLicense{}); err != nil {
			return err
		}
	}

	return nil
}

// CopyLicense Copy originalRepo license information to destRepo (use for forked repo)
func CopyLicense(originalRepo, destRepo *Repository) error {
	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		return err
	}
	defer committer.Close()

	repoLicenses, err := GetRepoLicenses(ctx, originalRepo)
	if err != nil {
		return err
	}
	if len(repoLicenses) > 0 {
		time := timeutil.TimeStampNow()
		for i := range repoLicenses {
			repoLicenses[i].ID = 0
			repoLicenses[i].RepoID = destRepo.ID
			repoLicenses[i].CreatedUnix = time
			repoLicenses[i].UpdatedUnix = time
		}
		if err := db.Insert(ctx, &repoLicenses); err != nil {
			return err
		}
	}
	return committer.Commit()
}
