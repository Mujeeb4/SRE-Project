// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"time"

	"code.gitea.io/gitea/modules/log"
)

var (
	// UILocation is the location on the UI, so that we can display the time on UI.
	DefaultUILocation = time.Local
)

func loadTimeFrom(rootCfg ConfigProvider) {
	zone := rootCfg.Section("time").Key("DEFAULT_UI_LOCATION").String()
	if zone != "" {
		var err error
		DefaultUILocation, err = time.LoadLocation(zone)
		if err != nil {
			log.Fatal("Load time zone failed: %v", err)
		} else {
			log.Info("Default UI Location is %v", zone)
		}
	}
	if DefaultUILocation == nil {
		DefaultUILocation = time.Local
	}
}
