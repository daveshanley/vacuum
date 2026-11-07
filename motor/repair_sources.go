// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

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

// repairSources holds only bytes already read during the baseline lint. Missing
// files fail closed: verification cannot fetch a new or changed dependency.
type repairSources struct {
	files        map[string][]byte
	base         string
	retrievals   map[string]string
	captureError error
}

func (s *repairSources) Open(name string) (fs.File, error) {
	data, ok := s.files[name]
	if !ok && !strings.Contains(name, "://") {
		data, ok = s.files[filepath.Join(s.base, name)]
	}
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &repairSourceFile{Reader: bytes.NewReader(data), name: name, size: int64(len(data))}, nil
}

type repairSourceFile struct {
	*bytes.Reader
	name string
	size int64
}

func (f *repairSourceFile) Close() error               { return nil }
func (f *repairSourceFile) Stat() (fs.FileInfo, error) { return f, nil }
func (f *repairSourceFile) Name() string               { return f.name }
func (f *repairSourceFile) Size() int64                { return f.size }
func (f *repairSourceFile) Mode() fs.FileMode          { return 0444 }
func (f *repairSourceFile) ModTime() time.Time         { return time.Time{} }
func (f *repairSourceFile) IsDir() bool                { return false }
func (f *repairSourceFile) Sys() any                   { return nil }

func freezeRepairSources(result *RuleSetExecutionResult) (*repairSources, map[string]string, error) {
	e := result.RuleSetExecution
	source := &repairSources{files: make(map[string][]byte)}
	absolute, err := filepath.Abs(e.SpecFileName)
	if err != nil {
		return nil, nil, err
	}
	source.base = filepath.Dir(absolute)
	hashes := make(map[string]string)
	if !externalRepairInputs(e.CanonicalDocument) {
		return source, hashes, nil
	}
	if e.Arazzo != nil {
		captured := result.repairSources
		if captured == nil || len(captured.files) == 0 {
			return nil, nil, fmt.Errorf("external Arazzo source bytes were not captured")
		}
		if captured.captureError != nil {
			return nil, nil, captured.captureError
		}
		for name, data := range captured.files {
			hashes[name] = repair.Hash(data)
		}
		return captured, hashes, nil
	}
	if e.AsyncAPI != nil || result.Index == nil || result.Index.GetRolodex() == nil {
		return nil, nil, fmt.Errorf("external inputs for this format require a frozen source collection")
	}
	rolodexes := []*index.Rolodex{result.Index.GetRolodex()}
	if e.IndexResolved != nil {
		rolodexes = append(rolodexes, e.IndexResolved.GetRolodex())
	}
	if e.IndexUnresolved != nil {
		rolodexes = append(rolodexes, e.IndexUnresolved.GetRolodex())
	}
	if e.Document != nil {
		rolodexes = append(rolodexes, e.Document.GetRolodex())
	}
	seen := make(map[*index.Rolodex]bool)
	for _, rolo := range rolodexes {
		if rolo == nil || seen[rolo] {
			continue
		}
		seen[rolo] = true
		for _, idx := range rolo.GetIndexes() {
			name := idx.GetSpecAbsolutePath()
			if name == "" || name == absolute {
				continue
			}
			file, err := rolo.Open(name)
			if err != nil {
				return nil, nil, fmt.Errorf("snapshot repair dependency: %w", err)
			}
			data := []byte(file.GetContent())
			hash := repair.Hash(data)
			if previous, ok := hashes[name]; ok && previous != hash {
				return nil, nil, fmt.Errorf("dependency changed during baseline lint: %s", name)
			}
			source.files[name] = data
			hashes[name] = hash
		}
	}
	if len(hashes) == 0 {
		return nil, nil, fmt.Errorf("external repair dependencies were not captured")
	}
	return source, hashes, nil
}

func (s *repairSources) GetFiles() map[string]index.RolodexFile { return nil }

// repairSourceResolver captures the existing Arazzo resolver's completed reads,
// or replays those exact bytes during verification. A nil inner never performs I/O.
type repairSourceResolver struct {
	inner    arazzosource.SourceDocumentResolver
	snapshot *repairSources
	reports  *arazzocontext.Resolver
}

func (r *repairSourceResolver) Resolve(ctx context.Context, request arazzosource.SourceRequest) (*arazzosource.ResolvedSource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.inner != nil {
		loaded, err := r.inner.Resolve(ctx, request)
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
			if old, ok := r.snapshot.files[alias]; ok && (!bytes.Equal(old, data) || r.snapshot.retrievals[alias] != retrieval) {
				r.snapshot.captureError = fmt.Errorf("Arazzo source changed during baseline validation: %s", alias)
				return nil, r.snapshot.captureError
			}
			r.snapshot.files[alias] = data
			r.snapshot.retrievals[alias] = retrieval
		}
		return loaded, nil
	}
	data, ok := r.snapshot.files[request.URL]
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
	retrieval := r.snapshot.retrievals[request.URL]
	if retrieval == "" {
		retrieval = request.URL
	}
	retained := &arazzocontext.SourceDocument{Root: root, Content: string(data)}
	r.reports.Sources[request.URL] = retained
	r.reports.Sources[retrieval] = retained
	r.reports.Files++
	r.reports.Bytes += int64(len(data))
	return &arazzosource.ResolvedSource{Name: request.Name, RetrievalURI: retrieval, RootNode: root, SourceBytes: append([]byte(nil), data...)}, nil
}
