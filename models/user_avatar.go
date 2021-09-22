// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"crypto/md5"
	"fmt"
	"image/png"
	"io"
	"strconv"

	"code.gitea.io/gitea/models/avatars"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/avatar"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/storage"
)

// CustomAvatarRelativePath returns user custom avatar relative path.
func (u *User) CustomAvatarRelativePath() string {
	return u.Avatar
}

// GenerateRandomAvatar generates a random avatar for user.
func (u *User) GenerateRandomAvatar() error {
	return u.generateRandomAvatar(db.DefaultContext().Engine())
}

func (u *User) generateRandomAvatar(e db.Engine) error {
	seed := u.Email
	if len(seed) == 0 {
		seed = u.Name
	}

	img, err := avatar.RandomImage([]byte(seed))
	if err != nil {
		return fmt.Errorf("RandomImage: %v", err)
	}

	u.Avatar = avatars.HashEmail(seed)

	// Don't share the images so that we can delete them easily
	if err := storage.SaveFrom(storage.Avatars, u.CustomAvatarRelativePath(), func(w io.Writer) error {
		if err := png.Encode(w, img); err != nil {
			log.Error("Encode: %v", err)
		}
		return err
	}); err != nil {
		return fmt.Errorf("Failed to create dir %s: %v", u.CustomAvatarRelativePath(), err)
	}

	if _, err := e.ID(u.ID).Cols("avatar").Update(u); err != nil {
		return err
	}

	log.Info("New random avatar created: %d", u.ID)
	return nil
}

// AvatarLinkWithSize returns a link to the user's avatar with size
func (u *User) AvatarLinkWithSize(size int) string {
	if u.ID == -1 {
		// ghost user
		return avatars.DefaultAvatarLink()
	}

	var useLocalAvatar bool

	switch {
	case u.UseCustomAvatar:
		useLocalAvatar = true
	case setting.DisableGravatar, setting.OfflineMode:
		useLocalAvatar = true
		if u.Avatar == "" {
			if err := u.GenerateRandomAvatar(); err != nil {
				log.Error("GenerateRandomAvatar: %v", err)
			}
		}
	default:
		useLocalAvatar = false
	}

	if useLocalAvatar {
		if u.Avatar == "" {
			return avatars.DefaultAvatarLink()
		}
		if size > 0 {
			return setting.AppSubURL + "/avatars/" + u.Avatar + "?size=" + strconv.Itoa(size)
		}
		return setting.AppSubURL + "/avatars/" + u.Avatar
	}

	return avatars.GenerateEmailAvatarFastLink(u.AvatarEmail, size)
}

// AvatarLinkDefaultSize returns a avatar link with default size
func (u *User) AvatarLinkDefaultSize() string {
	return u.AvatarLinkWithSize(avatars.DefaultAvatarSize)
}

// UploadAvatar saves custom avatar for user.
// FIXME: split uploads to different subdirs in case we have massive users.
func (u *User) UploadAvatar(data []byte) error {
	m, err := avatar.Prepare(data)
	if err != nil {
		return err
	}

	sess := db.DefaultContext().NewSession()
	defer sess.Close()
	if err = sess.Begin(); err != nil {
		return err
	}

	u.UseCustomAvatar = true
	// Different users can upload same image as avatar
	// If we prefix it with u.ID, it will be separated
	// Otherwise, if any of the users delete his avatar
	// Other users will lose their avatars too.
	u.Avatar = fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d-%x", u.ID, md5.Sum(data)))))
	if err = updateUserCols(sess, u, "use_custom_avatar", "avatar"); err != nil {
		return fmt.Errorf("updateUser: %v", err)
	}

	if err := storage.SaveFrom(storage.Avatars, u.CustomAvatarRelativePath(), func(w io.Writer) error {
		if err := png.Encode(w, *m); err != nil {
			log.Error("Encode: %v", err)
		}
		return err
	}); err != nil {
		return fmt.Errorf("Failed to create dir %s: %v", u.CustomAvatarRelativePath(), err)
	}

	return sess.Commit()
}

// DeleteAvatar deletes the user's custom avatar.
func (u *User) DeleteAvatar() error {
	aPath := u.CustomAvatarRelativePath()
	log.Trace("DeleteAvatar[%d]: %s", u.ID, aPath)
	if len(u.Avatar) > 0 {
		if err := storage.Avatars.Delete(aPath); err != nil {
			return fmt.Errorf("Failed to remove %s: %v", aPath, err)
		}
	}

	u.UseCustomAvatar = false
	u.Avatar = ""
	if _, err := db.DefaultContext().Engine().ID(u.ID).Cols("avatar, use_custom_avatar").Update(u); err != nil {
		return fmt.Errorf("UpdateUser: %v", err)
	}
	return nil
}
