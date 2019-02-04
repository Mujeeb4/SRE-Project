// +build bindata

// Copyright 2016 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package public

import (
	"github.com/gobuffalo/packr/v2"
	"gopkg.in/macaron.v1"
)

// Static implements the macaron static handler for serving assets.
func Static(opts *Options) macaron.Handler {
	box := packr.New("public", "../../public")
	opts.FileSystem = box
	// we don't need to pass the directory, because the directory var is only
	// used when in the options there is no FileSystem.
	return opts.staticHandler("")
}
