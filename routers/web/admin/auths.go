// Copyright 2014 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/modules/auth/pam"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	auth_service "code.gitea.io/gitea/services/auth"
	"code.gitea.io/gitea/services/auth/source/ldap"
	"code.gitea.io/gitea/services/auth/source/oauth2"
	pam_service "code.gitea.io/gitea/services/auth/source/pam"
	"code.gitea.io/gitea/services/auth/source/smtp"
	"code.gitea.io/gitea/services/auth/source/sspi"
	"code.gitea.io/gitea/services/forms"

	"xorm.io/xorm/convert"
)

const (
	tplAuths    base.TplName = "admin/auth/list"
	tplAuthNew  base.TplName = "admin/auth/new"
	tplAuthEdit base.TplName = "admin/auth/edit"
)

var (
	separatorAntiPattern = regexp.MustCompile(`[^\w-\.]`)
	langCodePattern      = regexp.MustCompile(`^[a-z]{2}-[A-Z]{2}$`)
)

// Authentications show authentication config page
func Authentications(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.authentication")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminAuthentications"] = true

	var err error
	ctx.Data["Sources"], err = auth.Sources()
	if err != nil {
		ctx.ServerError("auth.Sources", err)
		return
	}

	ctx.Data["Total"] = auth.CountSources()
	ctx.HTML(http.StatusOK, tplAuths)
}

type dropdownItem struct {
	Name string
	Type interface{}
}

var (
	authSources = func() []dropdownItem {
		items := []dropdownItem{
			{auth.LDAP.String(), auth.LDAP},
			{auth.DLDAP.String(), auth.DLDAP},
			{auth.SMTP.String(), auth.SMTP},
			{auth.OAuth2.String(), auth.OAuth2},
			{auth.SSPI.String(), auth.SSPI},
		}
		if pam.Supported {
			items = append(items, dropdownItem{auth.Names[auth.PAM], auth.PAM})
		}
		return items
	}()

	securityProtocols = []dropdownItem{
		{ldap.SecurityProtocolNames[ldap.SecurityProtocolUnencrypted], ldap.SecurityProtocolUnencrypted},
		{ldap.SecurityProtocolNames[ldap.SecurityProtocolLDAPS], ldap.SecurityProtocolLDAPS},
		{ldap.SecurityProtocolNames[ldap.SecurityProtocolStartTLS], ldap.SecurityProtocolStartTLS},
	}
)

// NewAuthSource render adding a new auth source page
func NewAuthSource(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.auths.new")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminAuthentications"] = true

	ctx.Data["TypeNames"] = auth.Names
	ctx.Data["AuthSources"] = authSources
	ctx.Data["SecurityProtocols"] = securityProtocols
	ctx.Data["SMTPAuths"] = smtp.Authenticators
	oauth2providers := oauth2.GetOAuth2Providers()
	ctx.Data["OAuth2Providers"] = oauth2providers

	ctx.Data["Form"] = forms.AuthenticationForm{
		Type:                     auth.LDAP.Int(),
		SMTPAuth:                 "PLAIN",
		IsActive:                 true,
		IsSyncEnabled:            true,
		Oauth2Provider:           oauth2providers[0].Name(), // only the first as default
		SSPIAutoCreateUsers:      true,
		SSPIAutoActivateUsers:    true,
		SSPIStripDomainNames:     true,
		SSPISeparatorReplacement: "_",
	}

	ctx.HTML(http.StatusOK, tplAuthNew)
}

func parseLDAPConfig(form forms.AuthenticationForm) *ldap.Source {
	var pageSize uint32
	if form.UsePagedSearch {
		pageSize = uint32(form.SearchPageSize)
	}
	return &ldap.Source{
		Name:                  form.Name,
		Host:                  form.Host,
		Port:                  form.Port,
		SecurityProtocol:      ldap.SecurityProtocol(form.SecurityProtocol),
		SkipVerify:            form.SkipVerify,
		BindDN:                form.BindDN,
		UserDN:                form.UserDN,
		BindPassword:          form.BindPassword,
		UserBase:              form.UserBase,
		AttributeUsername:     form.AttributeUsername,
		AttributeName:         form.AttributeName,
		AttributeSurname:      form.AttributeSurname,
		AttributeMail:         form.AttributeMail,
		AttributesInBind:      form.AttributesInBind,
		AttributeSSHPublicKey: form.AttributeSSHPublicKey,
		AttributeAvatar:       form.AttributeAvatar,
		SearchPageSize:        pageSize,
		Filter:                form.Filter,
		GroupsEnabled:         form.GroupsEnabled,
		GroupDN:               form.GroupDN,
		GroupFilter:           form.GroupFilter,
		GroupMemberUID:        form.GroupMemberUID,
		GroupTeamMap:          form.GroupTeamMap,
		GroupTeamMapRemoval:   form.GroupTeamMapRemoval,
		UserUID:               form.UserUID,
		AdminFilter:           form.AdminFilter,
		RestrictedFilter:      form.RestrictedFilter,
		AllowDeactivateAll:    form.AllowDeactivateAll,
		Enabled:               true,
		SkipLocalTwoFA:        form.SkipLocalTwoFA,
	}
}

func parseSMTPConfig(form forms.AuthenticationForm) *smtp.Source {
	return &smtp.Source{
		Auth:           form.SMTPAuth,
		Host:           form.SMTPHost,
		Port:           form.SMTPPort,
		AllowedDomains: form.AllowedDomains,
		ForceSMTPS:     form.ForceSMTPS,
		SkipVerify:     form.SkipVerify,
		HeloHostname:   form.HeloHostname,
		DisableHelo:    form.DisableHelo,
		SkipLocalTwoFA: form.SkipLocalTwoFA,
	}
}

func parseOAuth2Config(form forms.AuthenticationForm) *oauth2.Source {
	var customURLMapping *oauth2.CustomURLMapping
	if form.Oauth2UseCustomURL {
		customURLMapping = &oauth2.CustomURLMapping{
			TokenURL:   form.Oauth2TokenURL,
			AuthURL:    form.Oauth2AuthURL,
			ProfileURL: form.Oauth2ProfileURL,
			EmailURL:   form.Oauth2EmailURL,
			Tenant:     form.Oauth2Tenant,
		}
	} else {
		customURLMapping = nil
	}
	var scopes []string
	for _, s := range strings.Split(form.Oauth2Scopes, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			scopes = append(scopes, s)
		}
	}

	return &oauth2.Source{
		Provider:                      form.Oauth2Provider,
		ClientID:                      form.Oauth2Key,
		ClientSecret:                  form.Oauth2Secret,
		OpenIDConnectAutoDiscoveryURL: form.OpenIDConnectAutoDiscoveryURL,
		CustomURLMapping:              customURLMapping,
		IconURL:                       form.Oauth2IconURL,
		Scopes:                        scopes,
		RequiredClaimName:             form.Oauth2RequiredClaimName,
		RequiredClaimValue:            form.Oauth2RequiredClaimValue,
		SkipLocalTwoFA:                form.SkipLocalTwoFA,
		GroupClaimName:                form.Oauth2GroupClaimName,
		RestrictedGroup:               form.Oauth2RestrictedGroup,
		AdminGroup:                    form.Oauth2AdminGroup,
		GroupTeamMap:                  form.Oauth2GroupTeamMap,
		GroupTeamMapRemoval:           form.Oauth2GroupTeamMapRemoval,
	}
}

func parseSSPIConfig(ctx *context.Context, form forms.AuthenticationForm) (*sspi.Source, error) {
	if util.IsEmptyString(form.SSPISeparatorReplacement) {
		ctx.Data["Err_SSPISeparatorReplacement"] = true
		return nil, errors.New(ctx.Tr("form.SSPISeparatorReplacement") + ctx.Tr("form.require_error"))
	}
	if separatorAntiPattern.MatchString(form.SSPISeparatorReplacement) {
		ctx.Data["Err_SSPISeparatorReplacement"] = true
		return nil, errors.New(ctx.Tr("form.SSPISeparatorReplacement") + ctx.Tr("form.alpha_dash_dot_error"))
	}

	if form.SSPIDefaultLanguage != "" && !langCodePattern.MatchString(form.SSPIDefaultLanguage) {
		ctx.Data["Err_SSPIDefaultLanguage"] = true
		return nil, errors.New(ctx.Tr("form.lang_select_error"))
	}

	return &sspi.Source{
		AutoCreateUsers:      form.SSPIAutoCreateUsers,
		AutoActivateUsers:    form.SSPIAutoActivateUsers,
		StripDomainNames:     form.SSPIStripDomainNames,
		SeparatorReplacement: form.SSPISeparatorReplacement,
		DefaultLanguage:      form.SSPIDefaultLanguage,
	}, nil
}

// NewAuthSourcePost response for adding an auth source
func NewAuthSourcePost(ctx *context.Context) {
	form := *web.GetForm(ctx).(*forms.AuthenticationForm)
	ctx.Data["Title"] = ctx.Tr("admin.auths.new")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminAuthentications"] = true

	ctx.Data["AuthSources"] = authSources
	ctx.Data["SecurityProtocols"] = securityProtocols
	ctx.Data["SMTPAuths"] = smtp.Authenticators
	oauth2providers := oauth2.GetOAuth2Providers()
	ctx.Data["OAuth2Providers"] = oauth2providers

	hasTLS := false
	var config convert.Conversion
	switch auth.Type(form.Type) {
	case auth.LDAP, auth.DLDAP:
		config = parseLDAPConfig(form)
		hasTLS = ldap.SecurityProtocol(form.SecurityProtocol) > ldap.SecurityProtocolUnencrypted
	case auth.SMTP:
		config = parseSMTPConfig(form)
		hasTLS = true
	case auth.PAM:
		config = &pam_service.Source{
			ServiceName:    form.PAMServiceName,
			EmailDomain:    form.PAMEmailDomain,
			SkipLocalTwoFA: form.SkipLocalTwoFA,
		}
	case auth.OAuth2:
		config = parseOAuth2Config(form)
		oauth2Config := config.(*oauth2.Source)
		if oauth2Config.Provider == "openidConnect" {
			discoveryURL, err := url.Parse(oauth2Config.OpenIDConnectAutoDiscoveryURL)
			if err != nil || (discoveryURL.Scheme != "http" && discoveryURL.Scheme != "https") {
				ctx.Data["Err_DiscoveryURL"] = true
				ctx.RenderWithErr(ctx.Tr("admin.auths.invalid_openIdConnectAutoDiscoveryURL"), tplAuthNew, form)
				return
			}
		}
	case auth.SSPI:
		var err error
		config, err = parseSSPIConfig(ctx, form)
		if err != nil {
			ctx.RenderWithErr(err.Error(), tplAuthNew, form)
			return
		}
		existing, err := auth.SourcesByType(auth.SSPI)
		if err != nil || len(existing) > 0 {
			ctx.Data["Err_Type"] = true
			ctx.RenderWithErr(ctx.Tr("admin.auths.login_source_of_type_exist"), tplAuthNew, form)
			return
		}
	default:
		ctx.Error(http.StatusBadRequest)
		return
	}
	ctx.Data["HasTLS"] = hasTLS

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplAuthNew)
		return
	}

	if err := auth.CreateSource(&auth.Source{
		Type:          auth.Type(form.Type),
		Name:          form.Name,
		IsActive:      form.IsActive,
		IsSyncEnabled: form.IsSyncEnabled,
		Cfg:           config,
	}); err != nil {
		if auth.IsErrSourceAlreadyExist(err) {
			ctx.Data["Err_Name"] = true
			ctx.RenderWithErr(ctx.Tr("admin.auths.login_source_exist", err.(auth.ErrSourceAlreadyExist).Name), tplAuthNew, form)
		} else if oauth2.IsErrOpenIDConnectInitialize(err) {
			ctx.Data["Err_DiscoveryURL"] = true
			unwrapped := err.(oauth2.ErrOpenIDConnectInitialize).Unwrap()
			ctx.RenderWithErr(ctx.Tr("admin.auths.unable_to_initialize_openid", unwrapped), tplAuthNew, form)
		} else {
			ctx.ServerError("auth.CreateSource", err)
		}
		return
	}

	log.Trace("Authentication created by admin(%s): %s", ctx.Doer.Name, form.Name)

	ctx.Flash.Success(ctx.Tr("admin.auths.new_success", form.Name))
	ctx.Redirect(setting.AppSubURL + "/admin/auths")
}

func parseSource(ctx *context.Context) *auth.Source {
	source, err := auth.GetSourceByID(ctx.ParamsInt64(":authid"))
	if err != nil {
		ctx.ServerError("auth.GetSourceByID", err)
		return nil
	}
	ctx.Data["HasTLS"] = source.HasTLS()
	ctx.Data["Type"] = source.Type

	form := forms.AuthenticationForm{
		ID:            source.ID,
		Type:          source.Type.Int(),
		Name:          source.Name,
		IsActive:      source.IsActive,
		IsSyncEnabled: source.IsSyncEnabled,
	}

	if source.Cfg != nil {
		switch source.Type {
		case auth.LDAP, auth.DLDAP:
			cfg := source.Cfg.(*ldap.Source)
			if cfg.SearchPageSize > 0 {
				form.UsePagedSearch = true
			}
			form.Host = cfg.Host
			form.Port = cfg.Port
			form.SecurityProtocol = cfg.SecurityProtocol.Int()
			form.SkipVerify = cfg.SkipVerify
			form.BindDN = cfg.BindDN
			form.BindPassword = cfg.BindPassword
			form.UserBase = cfg.UserBase
			form.AttributeUsername = cfg.AttributeUsername
			form.AttributeName = cfg.AttributeName
			form.AttributeSurname = cfg.AttributeSurname
			form.AttributeMail = cfg.AttributeMail
			form.AttributesInBind = cfg.AttributesInBind
			form.AttributeSSHPublicKey = cfg.AttributeSSHPublicKey
			form.AttributeAvatar = cfg.AttributeAvatar
			form.SearchPageSize = int(cfg.SearchPageSize)
			form.Filter = cfg.Filter
			form.GroupsEnabled = cfg.GroupsEnabled
			form.GroupDN = cfg.GroupDN
			form.GroupFilter = cfg.GroupFilter
			form.GroupMemberUID = cfg.GroupMemberUID
			form.GroupTeamMap = cfg.GroupTeamMap
			form.GroupTeamMapRemoval = cfg.GroupTeamMapRemoval
			form.UserUID = cfg.UserUID
			form.AdminFilter = cfg.AdminFilter
			form.RestrictedFilter = cfg.RestrictedFilter
			form.AllowDeactivateAll = cfg.AllowDeactivateAll
			//form.Enabled=cfg.Enabled
			form.SkipLocalTwoFA = cfg.SkipLocalTwoFA
		case auth.SMTP:
			cfg := source.Cfg.(*smtp.Source)
			form.SMTPAuth = cfg.Auth
			form.SMTPHost = cfg.Host
			form.SMTPPort = cfg.Port
			form.AllowedDomains = cfg.AllowedDomains
			form.ForceSMTPS = cfg.ForceSMTPS
			form.SkipVerify = cfg.SkipVerify
			form.HeloHostname = cfg.HeloHostname
			form.DisableHelo = cfg.DisableHelo
			form.SkipLocalTwoFA = cfg.SkipLocalTwoFA
		case auth.PAM:
			cfg := source.Cfg.(*pam_service.Source)
			form.PAMServiceName = cfg.ServiceName
			form.PAMEmailDomain = cfg.EmailDomain
			form.SkipLocalTwoFA = cfg.SkipLocalTwoFA
		case auth.OAuth2:
			cfg := source.Cfg.(*oauth2.Source)
			form.Oauth2Provider = cfg.Provider
			form.Oauth2Key = cfg.ClientID
			form.Oauth2Secret = cfg.ClientSecret
			form.OpenIDConnectAutoDiscoveryURL = cfg.OpenIDConnectAutoDiscoveryURL
			if cfg.CustomURLMapping != nil {
				form.Oauth2UseCustomURL = true
				form.Oauth2TokenURL = cfg.CustomURLMapping.TokenURL
				form.Oauth2AuthURL = cfg.CustomURLMapping.AuthURL
				form.Oauth2ProfileURL = cfg.CustomURLMapping.ProfileURL
				form.Oauth2EmailURL = cfg.CustomURLMapping.EmailURL
				form.Oauth2Tenant = cfg.CustomURLMapping.Tenant
			}
			form.Oauth2IconURL = cfg.IconURL
			form.Oauth2Scopes = strings.Join(cfg.Scopes, ",")
			form.Oauth2RequiredClaimName = cfg.RequiredClaimName
			form.Oauth2RequiredClaimValue = cfg.RequiredClaimValue
			form.SkipLocalTwoFA = cfg.SkipLocalTwoFA
			form.Oauth2GroupClaimName = cfg.GroupClaimName
			form.Oauth2RestrictedGroup = cfg.RestrictedGroup
			form.Oauth2AdminGroup = cfg.AdminGroup
			form.Oauth2GroupTeamMap = cfg.GroupTeamMap
			form.Oauth2GroupTeamMapRemoval = cfg.GroupTeamMapRemoval
		case auth.SSPI:
			cfg := source.Cfg.(*sspi.Source)
			form.SSPIAutoCreateUsers = cfg.AutoCreateUsers
			form.SSPIAutoActivateUsers = cfg.AutoActivateUsers
			form.SSPIStripDomainNames = cfg.StripDomainNames
			form.SSPISeparatorReplacement = cfg.SeparatorReplacement
			form.SSPIDefaultLanguage = cfg.DefaultLanguage
		default:
			ctx.Error(http.StatusBadRequest)
			return nil
		}
	}
	ctx.Data["Form"] = form

	return source
}

// EditAuthSource render editing auth source page
func EditAuthSource(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.auths.edit")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminAuthentications"] = true

	ctx.Data["SecurityProtocols"] = securityProtocols
	ctx.Data["SMTPAuths"] = smtp.Authenticators
	oauth2providers := oauth2.GetOAuth2Providers()
	ctx.Data["OAuth2Providers"] = oauth2providers
	ctx.Data["SourceTypeNames"] = auth.Names

	parseSource(ctx)
	if ctx.Written() {
		return
	}

	ctx.HTML(http.StatusOK, tplAuthEdit)
}

// EditAuthSourcePost response for editing auth source
func EditAuthSourcePost(ctx *context.Context) {
	form := *web.GetForm(ctx).(*forms.AuthenticationForm)
	ctx.Data["Title"] = ctx.Tr("admin.auths.edit")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminAuthentications"] = true

	ctx.Data["SecurityProtocols"] = securityProtocols
	ctx.Data["SMTPAuths"] = smtp.Authenticators
	oauth2providers := oauth2.GetOAuth2Providers()
	ctx.Data["OAuth2Providers"] = oauth2providers
	ctx.Data["SourceTypeNames"] = auth.Names

	source := parseSource(ctx)
	if ctx.Written() {
		return
	}

	var err error
	var config convert.Conversion
	switch auth.Type(form.Type) {
	case auth.LDAP, auth.DLDAP:
		config = parseLDAPConfig(form)
	case auth.SMTP:
		config = parseSMTPConfig(form)
	case auth.PAM:
		config = &pam_service.Source{
			ServiceName: form.PAMServiceName,
			EmailDomain: form.PAMEmailDomain,
		}
	case auth.OAuth2:
		config = parseOAuth2Config(form)
		oauth2Config := config.(*oauth2.Source)
		if oauth2Config.Provider == "openidConnect" {
			discoveryURL, err := url.Parse(oauth2Config.OpenIDConnectAutoDiscoveryURL)
			if err != nil || (discoveryURL.Scheme != "http" && discoveryURL.Scheme != "https") {
				ctx.Data["Err_DiscoveryURL"] = true
				ctx.RenderWithErr(ctx.Tr("admin.auths.invalid_openIdConnectAutoDiscoveryURL"), tplAuthEdit, form)
				return
			}
		}
	case auth.SSPI:
		config, err = parseSSPIConfig(ctx, form)
		if err != nil {
			ctx.RenderWithErr(err.Error(), tplAuthEdit, form)
			return
		}
	default:
		ctx.Error(http.StatusBadRequest)
		return
	}

	source.Name = form.Name
	source.IsActive = form.IsActive
	source.IsSyncEnabled = form.IsSyncEnabled
	source.Cfg = config
	if err := auth.UpdateSource(source); err != nil {
		if auth.IsErrSourceAlreadyExist(err) {
			ctx.Data["Err_Name"] = true
			ctx.RenderWithErr(ctx.Tr("admin.auths.login_source_exist", err.(auth.ErrSourceAlreadyExist).Name), tplAuthEdit, form)
		} else if oauth2.IsErrOpenIDConnectInitialize(err) {
			ctx.Flash.Error(err.Error(), true)
			ctx.Data["Err_DiscoveryURL"] = true
			ctx.HTML(http.StatusOK, tplAuthEdit)
		} else {
			ctx.ServerError("UpdateSource", err)
		}
		return
	}
	log.Trace("Authentication changed by admin(%s): %d", ctx.Doer.Name, source.ID)

	ctx.Flash.Success(ctx.Tr("admin.auths.update_success"))
	ctx.Redirect(setting.AppSubURL + "/admin/auths/" + strconv.FormatInt(form.ID, 10))
}

// DeleteAuthSource response for deleting an auth source
func DeleteAuthSource(ctx *context.Context) {
	source, err := auth.GetSourceByID(ctx.ParamsInt64(":authid"))
	if err != nil {
		ctx.ServerError("auth.GetSourceByID", err)
		return
	}

	if err = auth_service.DeleteSource(source); err != nil {
		if auth.IsErrSourceInUse(err) {
			ctx.Flash.Error(ctx.Tr("admin.auths.still_in_used"))
		} else {
			ctx.Flash.Error(fmt.Sprintf("auth_service.DeleteSource: %v", err))
		}
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"redirect": setting.AppSubURL + "/admin/auths/" + url.PathEscape(ctx.Params(":authid")),
		})
		return
	}
	log.Trace("Authentication deleted by admin(%s): %d", ctx.Doer.Name, source.ID)

	ctx.Flash.Success(ctx.Tr("admin.auths.deletion_success"))
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": setting.AppSubURL + "/admin/auths",
	})
}
