// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package common

import (
	"fmt"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/charset"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/httpcache"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/typesniffer"
	"code.gitea.io/gitea/modules/util"
)

// ServeBlob download a git.Blob
func ServeBlob(ctx *context.Context, blob *git.Blob, lastModified time.Time) error {
	if httpcache.HandleGenericETagTimeCache(ctx.Req, ctx.Resp, `"`+blob.ID.String()+`"`, lastModified) {
		return nil
	}

	dataRc, err := blob.DataAsync()
	if err != nil {
		return err
	}
	defer func() {
		if err = dataRc.Close(); err != nil {
			log.Error("ServeBlob: Close: %v", err)
		}
	}()

	return ServeData(ctx, ctx.Repo.TreePath, blob.Size(), dataRc)
}

// ServeData download file from io.Reader
func ServeData(ctx *context.Context, filePath string, size int64, reader io.Reader) error {
	buf := make([]byte, 1024)
	n, err := util.ReadAtMost(reader, buf)
	if err != nil {
		return err
	}
	if n >= 0 {
		buf = buf[:n]
	}

	httpcache.AddCacheControlToHeader(ctx.Resp.Header(), 5*time.Minute)

	if size >= 0 {
		ctx.Resp.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	} else {
		log.Error("ServeData called to serve data: %s with size < 0: %d", filePath, size)
	}

	fileName := path.Base(filePath)
	st := typesniffer.DetectContentType(buf)
	mimeType := ""
	cs := ""

	if setting.MimeTypeMap.Enabled {
		fileExtension := strings.ToLower(filepath.Ext(fileName))
		mimeType = setting.MimeTypeMap.Map[fileExtension]
	}

	if mimeType == "" {
		if st.IsBrowsableType() {
			mimeType = st.GetContentType()
		} else if st.IsText() || ctx.FormBool("render") {
			mimeType = "text/plain"
		} else {
			mimeType = typesniffer.ApplicationOctetStream
		}
	}

	if st.IsText() || ctx.FormBool("render") {
		cs, err = charset.DetectEncoding(buf)
		if err != nil {
			log.Error("Detect raw file %s charset failed: %v, using by default utf-8", filePath, err)
			cs = "utf-8"
		}
	}

	if cs != "" {
		ctx.Resp.Header().Set("Content-Type", mimeType+"; charset="+strings.ToLower(cs))
	} else {
		ctx.Resp.Header().Set("Content-Type", mimeType)
	}
	ctx.Resp.Header().Set("X-Content-Type-Options", "nosniff")

	// serve types that can present a security risk with CSP
	if st.IsImage() || st.IsPDF() {
		ctx.Resp.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	}

	ctx.Resp.Header().Set("Content-Disposition", `inline; filename*=UTF-8''`+url.PathEscape(fileName))
	ctx.Resp.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")

	_, err = ctx.Resp.Write(buf)
	if err != nil {
		return err
	}
	_, err = io.Copy(ctx.Resp, reader)
	return err
}
