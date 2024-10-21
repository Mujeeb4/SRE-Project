// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"sync"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting/config"
)

type PictureStruct struct {
	DisableGravatar       *config.Value[bool]
	EnableFederatedAvatar *config.Value[bool]
}

type OpenWithEditorApp struct {
	DisplayName string
	OpenURL     string
}

type OpenWithEditorAppsType []OpenWithEditorApp

func (t OpenWithEditorAppsType) ToTextareaString() string {
	ret := ""
	for _, app := range t {
		ret += app.DisplayName + " = " + app.OpenURL + "\n"
	}
	return ret
}

func DefaultOpenWithEditorApps() OpenWithEditorAppsType {
	return OpenWithEditorAppsType{
		{
			DisplayName: "VS Code",
			OpenURL:     "vscode://vscode.git/clone?url={url}",
		},
		{
			DisplayName: "VSCodium",
			OpenURL:     "vscodium://vscode.git/clone?url={url}",
		},
		{
			DisplayName: "Intellij IDEA",
			OpenURL:     "jetbrains://idea/checkout/git?idea.required.plugins.id=Git4Idea&checkout.repo={url}",
		},
	}
}

type RepositoryStruct struct {
	OpenWithEditorApps *config.Value[OpenWithEditorAppsType]
}

type Explore struct {
	RequireSigninView        *config.Value[bool]
	DisableUsersPage         *config.Value[bool]
	DisableOrganizationsPage *config.Value[bool]
	DisableCodePage          *config.Value[bool]
}

type ServiceStruct struct {
	Explore *Explore
}

type ConfigStruct struct {
	Picture    *PictureStruct
	Repository *RepositoryStruct
	Service    *ServiceStruct
}

var (
	defaultConfig     *ConfigStruct
	defaultConfigOnce sync.Once
)

func initDefaultConfig() {
	config.SetCfgSecKeyGetter(&cfgSecKeyGetter{})
	defaultConfig = &ConfigStruct{
		Picture: &PictureStruct{
			DisableGravatar:       config.ValueJSON[bool]("picture.disable_gravatar").WithFileConfig(config.CfgSecKey{Sec: "picture", Key: "DISABLE_GRAVATAR"}),
			EnableFederatedAvatar: config.ValueJSON[bool]("picture.enable_federated_avatar").WithFileConfig(config.CfgSecKey{Sec: "picture", Key: "ENABLE_FEDERATED_AVATAR"}),
		},
		Repository: &RepositoryStruct{
			OpenWithEditorApps: config.ValueJSON[OpenWithEditorAppsType]("repository.open-with.editor-apps"),
		},
		Service: &ServiceStruct{
			Explore: &Explore{
				RequireSigninView:        config.ValueJSON[bool]("service.explore.require_signin_view").WithFileConfig(config.CfgSecKey{Sec: "service.explore", Key: "REQUIRE_SIGNIN_VIEW"}),
				DisableUsersPage:         config.ValueJSON[bool]("service.explore.disable_users_page").WithFileConfig(config.CfgSecKey{Sec: "service.explore", Key: "DISABLE_USERS_PAGE"}),
				DisableOrganizationsPage: config.ValueJSON[bool]("service.explore.disable_organizations_page").WithFileConfig(config.CfgSecKey{Sec: "service.explore", Key: "DISABLE_ORGANIZATIONS_PAGE"}),
				DisableCodePage:          config.ValueJSON[bool]("service.explore.disable_code_page").WithFileConfig(config.CfgSecKey{Sec: "service.explore", Key: "DISABLE_CODE_PAGE"}),
			},
		},
	}
}

func Config() *ConfigStruct {
	defaultConfigOnce.Do(initDefaultConfig)
	return defaultConfig
}

type cfgSecKeyGetter struct{}

func (c cfgSecKeyGetter) GetValue(sec, key string) (v string, has bool) {
	if key == "" {
		return "", false
	}
	cfgSec, err := CfgProvider.GetSection(sec)
	if err != nil {
		log.Error("Unable to get config section: %q", sec)
		return "", false
	}
	cfgKey := ConfigSectionKey(cfgSec, key)
	if cfgKey == nil {
		return "", false
	}
	return cfgKey.Value(), true
}
