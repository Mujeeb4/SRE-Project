// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package audit

import (
	"context"
	"io"

	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util/rotatingfilewriter"
)

type Appender interface {
	Record(context.Context, *Event)
	Close() error
	ReleaseReopen() error
}

// LogAppender writes an info log entry for every audit event
type LogAppender struct{}

func (a *LogAppender) Record(ctx context.Context, e *Event) {
	log.Info("%s", e.Message)
}

func (a *LogAppender) Close() error {
	return nil
}

func (a *LogAppender) ReleaseReopen() error {
	return nil
}

// File writes json object for every audit event
type FileAppender struct {
	rfw *rotatingfilewriter.RotatingFileWriter
}

func NewFileAppender(filename string, opts *rotatingfilewriter.Options) (*FileAppender, error) {
	rfw, err := rotatingfilewriter.Open(filename, opts)
	if err != nil {
		return nil, err
	}

	return &FileAppender{rfw}, nil
}

func (a *FileAppender) Record(ctx context.Context, e *Event) {
	if err := WriteEventAsJSON(a.rfw, e); err != nil {
		log.Error("encoding event to file failed: %v", err)
	}
}

func (a *FileAppender) Close() error {
	return a.rfw.Close()
}

func (a *FileAppender) ReleaseReopen() error {
	return a.rfw.ReleaseReopen()
}

func WriteEventAsJSON(w io.Writer, e *Event) error {
	return json.NewEncoder(w).Encode(e)
}
