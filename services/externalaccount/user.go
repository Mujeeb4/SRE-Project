// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package externalaccount

import (
	"strings"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/structs"

	"github.com/markbates/goth"
)

func toExternalLoginUser(user *models.User, gothUser goth.User) (*models.ExternalLoginUser, error) {
	loginSource, err := models.GetActiveOAuth2LoginSourceByName(gothUser.Provider)
	if err != nil {
		return nil, err
	}
	return &models.ExternalLoginUser{
		ExternalID:        gothUser.UserID,
		UserID:            user.ID,
		LoginSourceID:     loginSource.ID,
		RawData:           gothUser.RawData,
		Provider:          gothUser.Provider,
		Email:             gothUser.Email,
		Name:              gothUser.Name,
		FirstName:         gothUser.FirstName,
		LastName:          gothUser.LastName,
		NickName:          gothUser.NickName,
		Description:       gothUser.Description,
		AvatarURL:         gothUser.AvatarURL,
		Location:          gothUser.Location,
		AccessToken:       gothUser.AccessToken,
		AccessTokenSecret: gothUser.AccessTokenSecret,
		RefreshToken:      gothUser.RefreshToken,
		ExpiresAt:         gothUser.ExpiresAt,
	}, nil
}

// LinkAccountToUser link the gothUser to the user
func LinkAccountToUser(user *models.User, gothUser goth.User) error {
	externalLoginUser, err := toExternalLoginUser(user, gothUser)
	if err != nil {
		return err
	}

	if err := models.LinkExternalToUser(user, externalLoginUser); err != nil {
		return err
	}

	externalID := externalLoginUser.ExternalID

	var tp structs.GitServiceType
	for _, s := range structs.SupportedFullGitService {
		if strings.EqualFold(s.Name(), gothUser.Provider) {
			tp = s
			break
		}
	}

	if tp.Name() != "" {
		return models.UpdateMigrationsByType(tp, externalID, user.ID)
	}

	return nil
}

// UpdateExternalUser updates external user's information
func UpdateExternalUser(user *models.User, gothUser goth.User) error {
	externalLoginUser, err := toExternalLoginUser(user, gothUser)
	if err != nil {
		return err
	}

	return models.UpdateExternalUserByExternalID(externalLoginUser)
}
