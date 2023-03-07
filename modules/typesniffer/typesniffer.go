// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package typesniffer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"code.gitea.io/gitea/modules/util"
)

// Use at most this many bytes to determine Content Type.
const sniffLen = 1024

const (
	// SvgMimeType MIME type of SVG images.
	SvgMimeType = "image/svg+xml"
	// ApplicationOctetStream MIME type of binary files.
	ApplicationOctetStream = "application/octet-stream"
)

var (
	svgTagRegex      = regexp.MustCompile(`(?si)\A\s*(?:(<!--.*?-->|<!DOCTYPE\s+svg([\s:]+.*?>|>))\s*)*<svg[\s>\/]`)
	svgTagInXMLRegex = regexp.MustCompile(`(?si)\A<\?xml\b.*?\?>\s*(?:(<!--.*?-->|<!DOCTYPE\s+svg([\s:]+.*?>|>))\s*)*<svg[\s>\/]`)
)

// SniffedType contains information about a blobs type.
type SniffedType struct {
	contentType string
}

// IsText etects if content format is plain text.
func (ct SniffedType) IsText() bool {
	return strings.Contains(ct.contentType, "text/")
}

// IsImage detects if data is an image format
func (ct SniffedType) IsImage() bool {
	return strings.Contains(ct.contentType, "image/")
}

// IsSvgImage detects if data is an SVG image format
func (ct SniffedType) IsSvgImage() bool {
	return strings.Contains(ct.contentType, SvgMimeType)
}

// IsPDF detects if data is a PDF format
func (ct SniffedType) IsPDF() bool {
	return strings.Contains(ct.contentType, "application/pdf")
}

// IsVideo detects if data is an video format
func (ct SniffedType) IsVideo() bool {
	return strings.Contains(ct.contentType, "video/")
}

// IsAudio detects if data is an video format
func (ct SniffedType) IsAudio() bool {
	return strings.Contains(ct.contentType, "audio/")
}

// IsRepresentableAsText returns true if file content can be represented as
// plain text or is empty.
func (ct SniffedType) IsRepresentableAsText() bool {
	return ct.IsText() || ct.IsSvgImage()
}

// IsBrowsableType returns whether a non-text type can be displayed in a browser
func (ct SniffedType) IsBrowsableBinaryType() bool {
	return ct.IsImage() || ct.IsSvgImage() || ct.IsPDF() || ct.IsVideo() || ct.IsAudio()
}

// GetMimeType returns the mime type
func (ct SniffedType) GetMimeType() string {
	return strings.SplitN(ct.contentType, ";", 2)[0]
}

// DetectContentType extends http.DetectContentType with more content types. Defaults to text/unknown if input is empty.
func DetectContentType(data []byte) SniffedType {
	if len(data) == 0 {
		return SniffedType{"text/unknown"}
	}

	ct := http.DetectContentType(data)

	if len(data) > sniffLen {
		data = data[:sniffLen]
	}

	if (strings.Contains(ct, "text/plain") || strings.Contains(ct, "text/html")) && svgTagRegex.Match(data) ||
		strings.Contains(ct, "text/xml") && svgTagInXMLRegex.Match(data) {
		// SVG is unsupported. https://github.com/golang/go/issues/15888
		ct = SvgMimeType
	}

	if strings.HasPrefix(ct, "audio/") && bytes.HasPrefix(data, []byte("ID3")) {
		// the MP3 detection is quite inaccurate, any content with "ID3" prefix will result in "audio/mpeg"
		// so remove the "ID3" prefix and detect again, if result is text, then it must be text content.
		ct2 := http.DetectContentType(data[3:])
		if strings.HasPrefix(ct2, "text/") {
			ct = ct2
		}
	}

	return SniffedType{ct}
}

// DetectContentTypeFromReader guesses the content type contained in the reader.
func DetectContentTypeFromReader(r io.Reader) (SniffedType, error) {
	buf := make([]byte, sniffLen)
	n, err := util.ReadAtMost(r, buf)
	if err != nil {
		return SniffedType{}, fmt.Errorf("DetectContentTypeFromReader io error: %w", err)
	}
	buf = buf[:n]

	return DetectContentType(buf), nil
}
