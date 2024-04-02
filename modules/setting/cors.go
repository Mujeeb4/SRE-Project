// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"time"

	"code.gitea.io/gitea/modules/log"
)

// CORSConfig defines CORS settings
var CORSConfig = struct {
	Enabled          bool
	AllowDomain      []string // FIXME: this option is from legacy code, it actually works as "AllowedOrigins". When refactoring in the future, the config option should also be renamed together.
	Methods          []string
	MaxAge           time.Duration
	AllowCredentials bool
	Headers          []string
}{
	AllowDomain:   []string{"*"},
	Methods:       []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
	Headers:       []string{"Content-Type", "User-Agent"},
	MaxAge:        10 * time.Minute,
}

func loadCorsFrom(rootCfg ConfigProvider) {
	mustMapSetting(rootCfg, "cors", &CORSConfig)
	if CORSConfig.Enabled {
		log.Info("CORS Service Enabled")
	}
}
