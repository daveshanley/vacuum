// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"io/fs"
	"net/http"

	"github.com/pb33f/libopenapi/index"
)

// remoteFSHandler preserves the caller's filesystem for model builders that
// load remote references through an HTTP handler instead of RemoteFS.
func remoteFSHandler(ctx context.Context, files fs.FS) func(string) (*http.Response, error) {
	return func(location string) (*http.Response, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var file fs.File
		var err error
		if contextual, ok := files.(index.RolodexFSWithContext); ok {
			file, err = contextual.OpenWithContext(ctx, location)
		} else {
			file, err = files.Open(location)
		}
		if err != nil {
			return nil, err
		}
		if file == nil {
			return nil, &fs.PathError{Op: "open", Path: location, Err: fs.ErrNotExist}
		}
		if err := ctx.Err(); err != nil {
			file.Close()
			return nil, err
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			ContentLength: -1,
			Body:          &remoteFSBody{File: file, ctx: ctx},
		}, nil
	}
}

type remoteFSBody struct {
	fs.File
	ctx context.Context
}

func (b *remoteFSBody) Read(p []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	return b.File.Read(p)
}
