// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package markup

import (
	"testing"

	"code.gitea.io/gitea/modules/setting"

	"github.com/stretchr/testify/assert"
)

func TestCamoHandleLink(t *testing.T) {
	setting.AppURL = "https://gitea.com"
	// Test media proxy
	setting.CamoEnabled = true
	setting.CamoServerURL = "https://image.proxy"
	setting.CamoHMACKey = "geheim"

	assert.Equal(t,
		"https://gitea.com/img.jpg",
		camoHandleLink("https://gitea.com/img.jpg"))
	assert.Equal(t,
		"https://testimages.org/img.jpg",
		camoHandleLink("https://testimages.org/img.jpg"))
	assert.Equal(t,
		"https://image.proxy/eivin43gJwGVIjR9MiYYtFIk0mw/aHR0cDovL3Rlc3RpbWFnZXMub3JnL2ltZy5qcGc",
		camoHandleLink("http://testimages.org/img.jpg"))

	setting.CamoAllways = true
	assert.Equal(t,
		"https://gitea.com/img.jpg",
		camoHandleLink("https://gitea.com/img.jpg"))
	assert.Equal(t,
		"https://image.proxy/tkdlvmqpbIr7SjONfHNgEU622y0/aHR0cHM6Ly90ZXN0aW1hZ2VzLm9yZy9pbWcuanBn",
		camoHandleLink("https://testimages.org/img.jpg"))
	assert.Equal(t,
		"https://image.proxy/eivin43gJwGVIjR9MiYYtFIk0mw/aHR0cDovL3Rlc3RpbWFnZXMub3JnL2ltZy5qcGc",
		camoHandleLink("http://testimages.org/img.jpg"))

	// Restore previous settings
	setting.CamoEnabled = false
}
