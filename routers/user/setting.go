// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package user

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"image/png"
	"io/ioutil"
	"strings"

	"github.com/Unknwon/com"
	"github.com/Unknwon/i18n"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/tstranex/u2f"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/auth"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

const (
	tplSettingsProfile      base.TplName = "user/settings/profile"
	tplSettingsAccount      base.TplName = "user/settings/account"
	tplSettingsSecurity     base.TplName = "user/settings/security"
	tplSettingsTwofaEnroll  base.TplName = "user/settings/twofa_enroll"
	tplSettingsApplications base.TplName = "user/settings/applications"
	tplSettingsKeys         base.TplName = "user/settings/keys"
	tplSettingsOrganization base.TplName = "user/settings/organization"
	tplSettingsRepositories base.TplName = "user/settings/repos"
)

// Settings render user's profile page
func Settings(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsProfile"] = true
	ctx.HTML(200, tplSettingsProfile)
}

func handleUsernameChange(ctx *context.Context, newName string) {
	// Non-local users are not allowed to change their username.
	if len(newName) == 0 || !ctx.User.IsLocal() {
		return
	}

	// Check if user name has been changed
	if ctx.User.LowerName != strings.ToLower(newName) {
		if err := models.ChangeUserName(ctx.User, newName); err != nil {
			switch {
			case models.IsErrUserAlreadyExist(err):
				ctx.Flash.Error(ctx.Tr("form.username_been_taken"))
				ctx.Redirect(setting.AppSubURL + "/user/settings")
			case models.IsErrEmailAlreadyUsed(err):
				ctx.Flash.Error(ctx.Tr("form.email_been_used"))
				ctx.Redirect(setting.AppSubURL + "/user/settings")
			case models.IsErrNameReserved(err):
				ctx.Flash.Error(ctx.Tr("user.form.name_reserved", newName))
				ctx.Redirect(setting.AppSubURL + "/user/settings")
			case models.IsErrNamePatternNotAllowed(err):
				ctx.Flash.Error(ctx.Tr("user.form.name_pattern_not_allowed", newName))
				ctx.Redirect(setting.AppSubURL + "/user/settings")
			default:
				ctx.ServerError("ChangeUserName", err)
			}
			return
		}
		log.Trace("User name changed: %s -> %s", ctx.User.Name, newName)
	}

	// In case it's just a case change
	ctx.User.Name = newName
	ctx.User.LowerName = strings.ToLower(newName)
}

// SettingsPost response for change user's profile
func SettingsPost(ctx *context.Context, form auth.UpdateProfileForm) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsProfile"] = true

	if ctx.HasError() {
		ctx.HTML(200, tplSettingsProfile)
		return
	}

	handleUsernameChange(ctx, form.Name)
	if ctx.Written() {
		return
	}

	ctx.User.FullName = form.FullName
	ctx.User.Email = form.Email
	ctx.User.KeepEmailPrivate = form.KeepEmailPrivate
	ctx.User.Website = form.Website
	ctx.User.Location = form.Location
	ctx.User.Language = form.Language
	if err := models.UpdateUserSetting(ctx.User); err != nil {
		if _, ok := err.(models.ErrEmailAlreadyUsed); ok {
			ctx.Flash.Error(ctx.Tr("form.email_been_used"))
			ctx.Redirect(setting.AppSubURL + "/user/settings")
			return
		}
		ctx.ServerError("UpdateUser", err)
		return
	}

	// Update the language to the one we just set
	ctx.SetCookie("lang", ctx.User.Language, nil, setting.AppSubURL)

	log.Trace("User settings updated: %s", ctx.User.Name)
	ctx.Flash.Success(i18n.Tr(ctx.User.Language, "settings.update_profile_success"))
	ctx.Redirect(setting.AppSubURL + "/user/settings")
}

// UpdateAvatarSetting update user's avatar
// FIXME: limit size.
func UpdateAvatarSetting(ctx *context.Context, form auth.AvatarForm, ctxUser *models.User) error {
	ctxUser.UseCustomAvatar = form.Source == auth.AvatarLocal
	if len(form.Gravatar) > 0 {
		ctxUser.Avatar = base.EncodeMD5(form.Gravatar)
		ctxUser.AvatarEmail = form.Gravatar
	}

	if form.Avatar != nil {
		fr, err := form.Avatar.Open()
		if err != nil {
			return fmt.Errorf("Avatar.Open: %v", err)
		}
		defer fr.Close()

		data, err := ioutil.ReadAll(fr)
		if err != nil {
			return fmt.Errorf("ioutil.ReadAll: %v", err)
		}
		if !base.IsImageFile(data) {
			return errors.New(ctx.Tr("settings.uploaded_avatar_not_a_image"))
		}
		if err = ctxUser.UploadAvatar(data); err != nil {
			return fmt.Errorf("UploadAvatar: %v", err)
		}
	} else {
		// No avatar is uploaded but setting has been changed to enable,
		// generate a random one when needed.
		if ctxUser.UseCustomAvatar && !com.IsFile(ctxUser.CustomAvatarPath()) {
			if err := ctxUser.GenerateRandomAvatar(); err != nil {
				log.Error(4, "GenerateRandomAvatar[%d]: %v", ctxUser.ID, err)
			}
		}
	}

	if err := models.UpdateUserCols(ctxUser, "avatar", "avatar_email", "use_custom_avatar"); err != nil {
		return fmt.Errorf("UpdateUser: %v", err)
	}

	return nil
}

// SettingsAvatarPost response for change user's avatar request
func SettingsAvatarPost(ctx *context.Context, form auth.AvatarForm) {
	if err := UpdateAvatarSetting(ctx, form, ctx.User); err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("settings.update_avatar_success"))
	}

	ctx.Redirect(setting.AppSubURL + "/user/settings")
}

// SettingsDeleteAvatar render delete avatar page
func SettingsDeleteAvatar(ctx *context.Context) {
	if err := ctx.User.DeleteAvatar(); err != nil {
		ctx.Flash.Error(err.Error())
	}

	ctx.Redirect(setting.AppSubURL + "/user/settings")
}

// SettingsAccount renders change user's password, user's email and user suicide page
func SettingsAccount(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsAccount"] = true
	ctx.Data["Email"] = ctx.User.Email

	emails, err := models.GetEmailAddresses(ctx.User.ID)
	if err != nil {
		ctx.ServerError("GetEmailAddresses", err)
		return
	}
	ctx.Data["Emails"] = emails

	ctx.HTML(200, tplSettingsAccount)
}

// SettingsAccountPost response for change user's password
func SettingsAccountPost(ctx *context.Context, form auth.ChangePasswordForm) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsAccount"] = true

	if ctx.HasError() {
		ctx.HTML(200, tplSettingsAccount)
		return
	}

	if len(form.Password) < setting.MinPasswordLength {
		ctx.Flash.Error(ctx.Tr("auth.password_too_short", setting.MinPasswordLength))
	} else if ctx.User.IsPasswordSet() && !ctx.User.ValidatePassword(form.OldPassword) {
		ctx.Flash.Error(ctx.Tr("settings.password_incorrect"))
	} else if form.Password != form.Retype {
		ctx.Flash.Error(ctx.Tr("form.password_not_match"))
	} else {
		var err error
		if ctx.User.Salt, err = models.GetUserSalt(); err != nil {
			ctx.ServerError("UpdateUser", err)
			return
		}
		ctx.User.HashPassword(form.Password)
		if err := models.UpdateUserCols(ctx.User, "salt", "passwd"); err != nil {
			ctx.ServerError("UpdateUser", err)
			return
		}
		log.Trace("User password updated: %s", ctx.User.Name)
		ctx.Flash.Success(ctx.Tr("settings.change_password_success"))
	}

	ctx.Redirect(setting.AppSubURL + "/user/settings/account")
}

// SettingsEmailPost response for change user's email
func SettingsEmailPost(ctx *context.Context, form auth.AddEmailForm) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsAccount"] = true

	// Make emailaddress primary.
	if ctx.Query("_method") == "PRIMARY" {
		if err := models.MakeEmailPrimary(&models.EmailAddress{ID: ctx.QueryInt64("id")}); err != nil {
			ctx.ServerError("MakeEmailPrimary", err)
			return
		}

		log.Trace("Email made primary: %s", ctx.User.Name)
		ctx.Redirect(setting.AppSubURL + "/user/settings/account")
		return
	}

	// Add Email address.
	emails, err := models.GetEmailAddresses(ctx.User.ID)
	if err != nil {
		ctx.ServerError("GetEmailAddresses", err)
		return
	}
	ctx.Data["Emails"] = emails

	if ctx.HasError() {
		ctx.HTML(200, tplSettingsAccount)
		return
	}

	email := &models.EmailAddress{
		UID:         ctx.User.ID,
		Email:       form.Email,
		IsActivated: !setting.Service.RegisterEmailConfirm,
	}
	if err := models.AddEmailAddress(email); err != nil {
		if models.IsErrEmailAlreadyUsed(err) {
			ctx.RenderWithErr(ctx.Tr("form.email_been_used"), tplSettingsAccount, &form)
			return
		}
		ctx.ServerError("AddEmailAddress", err)
		return
	}

	// Send confirmation email
	if setting.Service.RegisterEmailConfirm {
		models.SendActivateEmailMail(ctx.Context, ctx.User, email)

		if err := ctx.Cache.Put("MailResendLimit_"+ctx.User.LowerName, ctx.User.LowerName, 180); err != nil {
			log.Error(4, "Set cache(MailResendLimit) fail: %v", err)
		}
		ctx.Flash.Info(ctx.Tr("settings.add_email_confirmation_sent", email.Email, base.MinutesToFriendly(setting.Service.ActiveCodeLives, ctx.Locale.Language())))
	} else {
		ctx.Flash.Success(ctx.Tr("settings.add_email_success"))
	}

	log.Trace("Email address added: %s", email.Email)
	ctx.Redirect(setting.AppSubURL + "/user/settings/account")
}

// DeleteEmail response for delete user's email
func DeleteEmail(ctx *context.Context) {
	if err := models.DeleteEmailAddress(&models.EmailAddress{ID: ctx.QueryInt64("id"), UID: ctx.User.ID}); err != nil {
		ctx.ServerError("DeleteEmail", err)
		return
	}
	log.Trace("Email address deleted: %s", ctx.User.Name)

	ctx.Flash.Success(ctx.Tr("settings.email_deletion_success"))
	ctx.JSON(200, map[string]interface{}{
		"redirect": setting.AppSubURL + "/user/settings/account",
	})
}

// SettingsDelete render user suicide page and response for delete user himself
func SettingsDelete(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsAccount"] = true

	if _, err := models.UserSignIn(ctx.User.Name, ctx.Query("password")); err != nil {
		if models.IsErrUserNotExist(err) {
			ctx.RenderWithErr(ctx.Tr("form.enterred_invalid_password"), tplSettingsAccount, nil)
		} else {
			ctx.ServerError("UserSignIn", err)
		}
		return
	}

	if err := models.DeleteUser(ctx.User); err != nil {
		switch {
		case models.IsErrUserOwnRepos(err):
			ctx.Flash.Error(ctx.Tr("form.still_own_repo"))
			ctx.Redirect(setting.AppSubURL + "/user/settings/account")
		case models.IsErrUserHasOrgs(err):
			ctx.Flash.Error(ctx.Tr("form.still_has_org"))
			ctx.Redirect(setting.AppSubURL + "/user/settings/account")
		default:
			ctx.ServerError("DeleteUser", err)
		}
	} else {
		log.Trace("Account deleted: %s", ctx.User.Name)
		ctx.Redirect(setting.AppSubURL + "/")
	}
}

// SettingsSecurity render change user's password page and 2FA
func SettingsSecurity(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsSecurity"] = true

	enrolled := true
	_, err := models.GetTwoFactorByUID(ctx.User.ID)
	if err != nil {
		if models.IsErrTwoFactorNotEnrolled(err) {
			enrolled = false
		} else {
			ctx.ServerError("SettingsTwoFactor", err)
			return
		}
	}
	ctx.Data["TwofaEnrolled"] = enrolled
	if enrolled {
		ctx.Data["U2FRegistrations"], err = models.GetU2FRegistrationsByUID(ctx.User.ID)
		if err != nil {
			ctx.ServerError("GetU2FRegistrationsByUID", err)
			return
		}
		ctx.Data["RequireU2F"] = true
	}

	accountLinks, err := models.ListAccountLinks(ctx.User)
	if err != nil {
		ctx.ServerError("ListAccountLinks", err)
		return
	}

	// map the provider display name with the LoginSource
	sources := make(map[*models.LoginSource]string)
	for _, externalAccount := range accountLinks {
		if loginSource, err := models.GetLoginSourceByID(externalAccount.LoginSourceID); err == nil {
			var providerDisplayName string
			if loginSource.IsOAuth2() {
				providerTechnicalName := loginSource.OAuth2().Provider
				providerDisplayName = models.OAuth2Providers[providerTechnicalName].DisplayName
			} else {
				providerDisplayName = loginSource.Name
			}
			sources[loginSource] = providerDisplayName
		}
	}
	ctx.Data["AccountLinks"] = sources

	if ctx.Query("openid.return_to") != "" {
		settingsOpenIDVerify(ctx)
		return
	}

	openid, err := models.GetUserOpenIDs(ctx.User.ID)
	if err != nil {
		ctx.ServerError("GetUserOpenIDs", err)
		return
	}
	ctx.Data["OpenIDs"] = openid

	ctx.HTML(200, tplSettingsSecurity)
}

// SettingsDeleteAccountLink delete a single account link
func SettingsDeleteAccountLink(ctx *context.Context) {
	if _, err := models.RemoveAccountLink(ctx.User, ctx.QueryInt64("loginSourceID")); err != nil {
		ctx.Flash.Error("RemoveAccountLink: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("settings.remove_account_link_success"))
	}

	ctx.JSON(200, map[string]interface{}{
		"redirect": setting.AppSubURL + "/user/settings/security",
	})
}

// SettingsApplications render manage access token page
func SettingsApplications(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsApplications"] = true

	tokens, err := models.ListAccessTokens(ctx.User.ID)
	if err != nil {
		ctx.ServerError("ListAccessTokens", err)
		return
	}
	ctx.Data["Tokens"] = tokens

	ctx.HTML(200, tplSettingsApplications)
}

// SettingsApplicationsPost response for add user's access token
func SettingsApplicationsPost(ctx *context.Context, form auth.NewAccessTokenForm) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsApplications"] = true

	if ctx.HasError() {
		tokens, err := models.ListAccessTokens(ctx.User.ID)
		if err != nil {
			ctx.ServerError("ListAccessTokens", err)
			return
		}
		ctx.Data["Tokens"] = tokens
		ctx.HTML(200, tplSettingsApplications)
		return
	}

	t := &models.AccessToken{
		UID:  ctx.User.ID,
		Name: form.Name,
	}
	if err := models.NewAccessToken(t); err != nil {
		ctx.ServerError("NewAccessToken", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.generate_token_success"))
	ctx.Flash.Info(t.Sha1)

	ctx.Redirect(setting.AppSubURL + "/user/settings/applications")
}

// SettingsDeleteApplication response for delete user access token
func SettingsDeleteApplication(ctx *context.Context) {
	if err := models.DeleteAccessTokenByID(ctx.QueryInt64("id"), ctx.User.ID); err != nil {
		ctx.Flash.Error("DeleteAccessTokenByID: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("settings.delete_token_success"))
	}

	ctx.JSON(200, map[string]interface{}{
		"redirect": setting.AppSubURL + "/user/settings/applications",
	})
}

// SettingsKeys render user's SSH/GPG public keys page
func SettingsKeys(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsKeys"] = true
	ctx.Data["DisableSSH"] = setting.SSH.Disabled

	keys, err := models.ListPublicKeys(ctx.User.ID)
	if err != nil {
		ctx.ServerError("ListPublicKeys", err)
		return
	}
	ctx.Data["Keys"] = keys

	gpgkeys, err := models.ListGPGKeys(ctx.User.ID)
	if err != nil {
		ctx.ServerError("ListGPGKeys", err)
		return
	}
	ctx.Data["GPGKeys"] = gpgkeys

	ctx.HTML(200, tplSettingsKeys)
}

// SettingsKeysPost response for change user's SSH/GPG keys
func SettingsKeysPost(ctx *context.Context, form auth.AddKeyForm) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsKeys"] = true

	keys, err := models.ListPublicKeys(ctx.User.ID)
	if err != nil {
		ctx.ServerError("ListPublicKeys", err)
		return
	}
	ctx.Data["Keys"] = keys

	gpgkeys, err := models.ListGPGKeys(ctx.User.ID)
	if err != nil {
		ctx.ServerError("ListGPGKeys", err)
		return
	}
	ctx.Data["GPGKeys"] = gpgkeys

	if ctx.HasError() {
		ctx.HTML(200, tplSettingsKeys)
		return
	}
	switch form.Type {
	case "gpg":
		key, err := models.AddGPGKey(ctx.User.ID, form.Content)
		if err != nil {
			ctx.Data["HasGPGError"] = true
			switch {
			case models.IsErrGPGKeyParsing(err):
				ctx.Flash.Error(ctx.Tr("form.invalid_gpg_key", err.Error()))
				ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
			case models.IsErrGPGKeyIDAlreadyUsed(err):
				ctx.Data["Err_Content"] = true
				ctx.RenderWithErr(ctx.Tr("settings.gpg_key_id_used"), tplSettingsKeys, &form)
			case models.IsErrGPGNoEmailFound(err):
				ctx.Data["Err_Content"] = true
				ctx.RenderWithErr(ctx.Tr("settings.gpg_no_key_email_found"), tplSettingsKeys, &form)
			default:
				ctx.ServerError("AddPublicKey", err)
			}
			return
		}
		ctx.Flash.Success(ctx.Tr("settings.add_gpg_key_success", key.KeyID))
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
	case "ssh":
		content, err := models.CheckPublicKeyString(form.Content)
		if err != nil {
			if models.IsErrSSHDisabled(err) {
				ctx.Flash.Info(ctx.Tr("settings.ssh_disabled"))
			} else if models.IsErrKeyUnableVerify(err) {
				ctx.Flash.Info(ctx.Tr("form.unable_verify_ssh_key"))
			} else {
				ctx.Flash.Error(ctx.Tr("form.invalid_ssh_key", err.Error()))
			}
			ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
			return
		}

		if _, err = models.AddPublicKey(ctx.User.ID, form.Title, content); err != nil {
			ctx.Data["HasSSHError"] = true
			switch {
			case models.IsErrKeyAlreadyExist(err):
				ctx.Data["Err_Content"] = true
				ctx.RenderWithErr(ctx.Tr("settings.ssh_key_been_used"), tplSettingsKeys, &form)
			case models.IsErrKeyNameAlreadyUsed(err):
				ctx.Data["Err_Title"] = true
				ctx.RenderWithErr(ctx.Tr("settings.ssh_key_name_used"), tplSettingsKeys, &form)
			default:
				ctx.ServerError("AddPublicKey", err)
			}
			return
		}
		ctx.Flash.Success(ctx.Tr("settings.add_key_success", form.Title))
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")

	default:
		ctx.Flash.Warning("Function not implemented")
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
	}

}

// DeleteKey response for delete user's SSH/GPG key
func DeleteKey(ctx *context.Context) {

	switch ctx.Query("type") {
	case "gpg":
		if err := models.DeleteGPGKey(ctx.User, ctx.QueryInt64("id")); err != nil {
			ctx.Flash.Error("DeleteGPGKey: " + err.Error())
		} else {
			ctx.Flash.Success(ctx.Tr("settings.gpg_key_deletion_success"))
		}
	case "ssh":
		if err := models.DeletePublicKey(ctx.User, ctx.QueryInt64("id")); err != nil {
			ctx.Flash.Error("DeletePublicKey: " + err.Error())
		} else {
			ctx.Flash.Success(ctx.Tr("settings.ssh_key_deletion_success"))
		}
	default:
		ctx.Flash.Warning("Function not implemented")
		ctx.Redirect(setting.AppSubURL + "/user/settings/keys")
	}
	ctx.JSON(200, map[string]interface{}{
		"redirect": setting.AppSubURL + "/user/settings/keys",
	})
}

// SettingsTwoFactorRegenerateScratch regenerates the user's 2FA scratch code.
func SettingsTwoFactorRegenerateScratch(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsSecurity"] = true

	t, err := models.GetTwoFactorByUID(ctx.User.ID)
	if err != nil {
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}

	if err = t.GenerateScratchToken(); err != nil {
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}

	if err = models.UpdateTwoFactor(t); err != nil {
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.twofa_scratch_token_regenerated", t.ScratchToken))
	ctx.Redirect(setting.AppSubURL + "/user/settings/security")
}

// SettingsTwoFactorDisable deletes the user's 2FA settings.
func SettingsTwoFactorDisable(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsSecurity"] = true

	t, err := models.GetTwoFactorByUID(ctx.User.ID)
	if err != nil {
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}

	if err = models.DeleteTwoFactorByID(t.ID, ctx.User.ID); err != nil {
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.twofa_disabled"))
	ctx.Redirect(setting.AppSubURL + "/user/settings/security")
}

func twofaGenerateSecretAndQr(ctx *context.Context) bool {
	var otpKey *otp.Key
	var err error
	uri := ctx.Session.Get("twofaUri")
	if uri != nil {
		otpKey, err = otp.NewKeyFromURL(uri.(string))
	}
	if otpKey == nil {
		err = nil // clear the error, in case the URL was invalid
		otpKey, err = totp.Generate(totp.GenerateOpts{
			Issuer:      setting.AppName + " (" + strings.TrimRight(setting.AppURL, "/") + ")",
			AccountName: ctx.User.Name,
		})
		if err != nil {
			ctx.ServerError("SettingsTwoFactor", err)
			return false
		}
	}

	ctx.Data["TwofaSecret"] = otpKey.Secret()
	img, err := otpKey.Image(320, 240)
	if err != nil {
		ctx.ServerError("SettingsTwoFactor", err)
		return false
	}

	var imgBytes bytes.Buffer
	if err = png.Encode(&imgBytes, img); err != nil {
		ctx.ServerError("SettingsTwoFactor", err)
		return false
	}

	ctx.Data["QrUri"] = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(imgBytes.Bytes()))
	ctx.Session.Set("twofaSecret", otpKey.Secret())
	ctx.Session.Set("twofaUri", otpKey.String())
	return true
}

// SettingsTwoFactorEnroll shows the page where the user can enroll into 2FA.
func SettingsTwoFactorEnroll(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsSecurity"] = true

	t, err := models.GetTwoFactorByUID(ctx.User.ID)
	if t != nil {
		// already enrolled
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}
	if err != nil && !models.IsErrTwoFactorNotEnrolled(err) {
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}

	if !twofaGenerateSecretAndQr(ctx) {
		return
	}

	ctx.HTML(200, tplSettingsTwofaEnroll)
}

// SettingsTwoFactorEnrollPost handles enrolling the user into 2FA.
func SettingsTwoFactorEnrollPost(ctx *context.Context, form auth.TwoFactorAuthForm) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsSecurity"] = true

	t, err := models.GetTwoFactorByUID(ctx.User.ID)
	if t != nil {
		// already enrolled
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}
	if err != nil && !models.IsErrTwoFactorNotEnrolled(err) {
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}

	if ctx.HasError() {
		if !twofaGenerateSecretAndQr(ctx) {
			return
		}
		ctx.HTML(200, tplSettingsTwofaEnroll)
		return
	}

	secret := ctx.Session.Get("twofaSecret").(string)
	if !totp.Validate(form.Passcode, secret) {
		if !twofaGenerateSecretAndQr(ctx) {
			return
		}
		ctx.Flash.Error(ctx.Tr("settings.passcode_invalid"))
		ctx.HTML(200, tplSettingsTwofaEnroll)
		return
	}

	t = &models.TwoFactor{
		UID: ctx.User.ID,
	}
	err = t.SetSecret(secret)
	if err != nil {
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}
	err = t.GenerateScratchToken()
	if err != nil {
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}

	if err = models.NewTwoFactor(t); err != nil {
		ctx.ServerError("SettingsTwoFactor", err)
		return
	}

	ctx.Session.Delete("twofaSecret")
	ctx.Session.Delete("twofaUri")
	ctx.Flash.Success(ctx.Tr("settings.twofa_enrolled", t.ScratchToken))
	ctx.Redirect(setting.AppSubURL + "/user/settings/security")
}

// U2FRegister initializes the u2f registration procedure
func U2FRegister(ctx *context.Context, form auth.U2FRegistrationForm) {
	if form.Name == "" {
		ctx.Error(409)
		return
	}
	challenge, err := u2f.NewChallenge(setting.U2F.AppID, setting.U2F.TrustedFacets)
	if err != nil {
		ctx.ServerError("NewChallenge", err)
		return
	}
	err = ctx.Session.Set("u2fChallenge", challenge)
	if err != nil {
		ctx.ServerError("Session.Set", err)
		return
	}
	regs, err := models.GetU2FRegistrationsByUID(ctx.User.ID)
	if err != nil {
		ctx.ServerError("GetU2FRegistrationsByUID", err)
		return
	}
	for _, reg := range regs {
		if reg.Name == form.Name {
			ctx.Error(409, "Name already taken")
			return
		}
	}
	ctx.Session.Set("u2fName", form.Name)
	ctx.JSON(200, u2f.NewWebRegisterRequest(challenge, regs.ToRegistrations()))
}

// U2FRegisterPost receives the response of the security key
func U2FRegisterPost(ctx *context.Context, response u2f.RegisterResponse) {
	challSess := ctx.Session.Get("u2fChallenge")
	u2fName := ctx.Session.Get("u2fName")
	if challSess == nil || u2fName == nil {
		ctx.ServerError("U2FRegisterPost", errors.New("not in U2F session"))
		return
	}
	challenge := challSess.(*u2f.Challenge)
	name := u2fName.(string)
	config := &u2f.Config{
		// Chrome 66+ doesn't return the device's attestation
		// certificate by default.
		SkipAttestationVerify: true,
	}
	reg, err := u2f.Register(response, *challenge, config)
	if err != nil {
		ctx.ServerError("u2f.Register", err)
		return
	}
	if _, err = models.CreateRegistration(ctx.User, name, reg); err != nil {
		ctx.ServerError("u2f.Register", err)
		return
	}
	ctx.Status(200)
}

// U2FDelete deletes an security key by id
func U2FDelete(ctx *context.Context, form auth.U2FDeleteForm) {
	reg, err := models.GetU2FRegistrationByID(form.ID)
	if err != nil {
		if models.IsErrU2FRegistrationNotExist(err) {
			ctx.Status(200)
			return
		}
		ctx.ServerError("GetU2FRegistrationByID", err)
		return
	}
	if reg.UserID != ctx.User.ID {
		ctx.Status(401)
		return
	}
	if err := models.DeleteRegistration(reg); err != nil {
		ctx.ServerError("DeleteRegistration", err)
		return
	}
	ctx.JSON(200, map[string]interface{}{
		"redirect": setting.AppSubURL + "/user/settings/security",
	})
	return
}

// SettingsOrganization render all the organization of the user
func SettingsOrganization(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsOrganization"] = true
	orgs, err := models.GetOrgsByUserID(ctx.User.ID, ctx.IsSigned)
	if err != nil {
		ctx.ServerError("GetOrgsByUserID", err)
		return
	}
	ctx.Data["Orgs"] = orgs
	ctx.HTML(200, tplSettingsOrganization)
}

// SettingsRepos display a list of all repositories of the user
func SettingsRepos(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings")
	ctx.Data["PageIsSettingsRepos"] = true
	ctxUser := ctx.User

	var err error
	if err = ctxUser.GetRepositories(1, setting.UI.User.RepoPagingNum); err != nil {
		ctx.ServerError("GetRepositories", err)
		return
	}
	repos := ctxUser.Repos

	for i := range repos {
		if repos[i].IsFork {
			err := repos[i].GetBaseRepo()
			if err != nil {
				ctx.ServerError("GetBaseRepo", err)
				return
			}
			err = repos[i].BaseRepo.GetOwner()
			if err != nil {
				ctx.ServerError("GetOwner", err)
				return
			}
		}
	}

	ctx.Data["Owner"] = ctxUser
	ctx.Data["Repos"] = repos

	ctx.HTML(200, tplSettingsRepositories)
}
