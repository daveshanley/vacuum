// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	arazzocontext "github.com/daveshanley/vacuum/arazzo"
	"github.com/daveshanley/vacuum/repair"
	arazzosource "github.com/pb33f/libopenapi/arazzo"
	"github.com/pb33f/libopenapi/index"
)

// Sources holds only bytes already read during the baseline lint. Missing
// files fail closed: verification cannot fetch a new or changed dependency.
type Sources struct {
	Files        map[string][]byte
	Base         string
	Retrievals   map[string]string
	CaptureError error
}

// Open reads a captured source without accessing the filesystem or network.
func (s *Sources) Open(name string) (fs.File, error) {
	data, ok := s.Files[name]
	if !ok && !strings.Contains(name, "://") {
		data, ok = s.Files[filepath.Join(s.Base, name)]
	}
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &SourceFile{Reader: bytes.NewReader(data), Filename: name, Length: int64(len(data))}, nil
}

// SourceFile exposes captured bytes as a read-only filesystem file.
type SourceFile struct {
	*bytes.Reader
	Filename string
	Length   int64
}

// Close leaves the in-memory bytes available.
func (f *SourceFile) Close() error { return nil }

// Stat returns immutable metadata for the captured bytes.
func (f *SourceFile) Stat() (fs.FileInfo, error) { return f, nil }

// Name returns the captured source location.
func (f *SourceFile) Name() string { return f.Filename }

// Size returns the captured byte count.
func (f *SourceFile) Size() int64 { return f.Length }

// Mode reports a read-only regular file.
func (f *SourceFile) Mode() fs.FileMode { return 0444 }

// ModTime is unknown for a source snapshot.
func (f *SourceFile) ModTime() time.Time { return time.Time{} }

// IsDir reports false for captured sources.
func (f *SourceFile) IsDir() bool { return false }

// Sys returns no platform-specific metadata.
func (f *SourceFile) Sys() any { return nil }

// GetFiles leaves index ownership to the caller; Sources retains only bytes.
func (s *Sources) GetFiles() map[string]index.RolodexFile { return nil }

// SourceResolver captures the existing Arazzo resolver's completed reads,
// or replays those exact bytes during verification. A nil Inner never performs I/O.
type SourceResolver struct {
	Inner    arazzosource.SourceDocumentResolver
	Snapshot *Sources
	Reports  *arazzocontext.Resolver
}

// Resolve captures a baseline read or replays its bytes and retrieval URI.
func (r *SourceResolver) Resolve(ctx context.Context, request arazzosource.SourceRequest) (*arazzosource.ResolvedSource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.Inner != nil {
		loaded, err := r.Inner.Resolve(ctx, request)
		if err != nil {
			return nil, err
		}
		if loaded == nil || len(loaded.SourceBytes) == 0 {
			return nil, fmt.Errorf("Arazzo source has no captured bytes: %s", request.URL)
		}
		retrieval := loaded.RetrievalURI
		if retrieval == "" {
			retrieval = request.URL
		}
		data := append([]byte(nil), loaded.SourceBytes...)
		for _, alias := range []string{request.URL, retrieval} {
			if old, ok := r.Snapshot.Files[alias]; ok && (!bytes.Equal(old, data) || r.Snapshot.Retrievals[alias] != retrieval) {
				r.Snapshot.CaptureError = fmt.Errorf("Arazzo source changed during baseline validation: %s", alias)
				return nil, r.Snapshot.CaptureError
			}
			r.Snapshot.Files[alias] = data
			r.Snapshot.Retrievals[alias] = retrieval
		}
		return loaded, nil
	}
	data, ok := r.Snapshot.Files[request.URL]
	if !ok {
		return nil, fmt.Errorf("Arazzo source is outside the captured collection: %s", request.URL)
	}
	root, err := repair.ParseDocument(data)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	retrieval := r.Snapshot.Retrievals[request.URL]
	if retrieval == "" {
		retrieval = request.URL
	}
	retained := &arazzocontext.SourceDocument{Root: root, Content: string(data)}
	r.Reports.Sources[request.URL] = retained
	r.Reports.Sources[retrieval] = retained
	r.Reports.Files++
	r.Reports.Bytes += int64(len(data))
	return &arazzosource.ResolvedSource{Name: request.Name, RetrievalURI: retrieval, RootNode: root, SourceBytes: append([]byte(nil), data...)}, nil
}
