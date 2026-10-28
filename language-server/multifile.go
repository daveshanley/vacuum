// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package languageserver

import (
	"context"
	"fmt"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/language-server/internal/lsp"
	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/utils"
)

func remoteDocument(name string) bool {
	return strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://")
}

func documentLocation(name string) string {
	if remoteDocument(name) {
		return name
	}
	name = fileURIToPath(name)
	if absolute, err := filepath.Abs(name); err == nil {
		return absolute
	}
	return filepath.Clean(name)
}

func documentURI(name string) string {
	if remoteDocument(name) {
		return name
	}
	path := filepath.ToSlash(documentLocation(name))
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func documentBase(name string) string {
	if remoteDocument(name) {
		parsed, err := url.Parse(name)
		if err != nil {
			return name
		}
		return parsed.ResolveReference(&url.URL{Path: "."}).String()
	}
	return filepath.Dir(name)
}

// documentOverlay keeps one immutable view of open buffers for a lint invocation.
// Unopened references retain the normal disk and HTTP loading behavior.
type documentOverlay struct {
	contents map[string]string
	base     string
	ctx      context.Context
	client   *http.Client
	rolodex  *index.Rolodex
	files    sync.Map
}

// GetFiles avoids a directory scan: the resolver loads only referenced files.
func (o *documentOverlay) GetFiles() map[string]index.RolodexFile { return nil }
func (o *documentOverlay) SetRolodex(r *index.Rolodex)            { o.rolodex = r; o.files.Clear() }
func (o *documentOverlay) SetLogger(*slog.Logger)                 {}

// Stat must not construct an index: reference extraction probes file existence
// while constructing the parent index, including for circular references.
func (o *documentOverlay) Stat(name string) (fs.FileInfo, error) {
	file, err := o.openContent(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.Stat()
}

func (o *documentOverlay) Open(name string) (fs.File, error) {
	if remoteDocument(name) {
		return o.openContent(name)
	}
	name = o.location(name)
	if cached, ok := o.files.Load(name); ok {
		return cached.(*indexedBuffer), nil
	}
	file, err := o.openContent(name)
	if err != nil || o.rolodex == nil {
		return file, err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, err
	}
	if !remoteDocument(name) && !filepath.IsAbs(name) {
		name = filepath.Join(o.base, name)
	}
	name = documentLocation(name)
	cfg := *o.rolodex.GetConfig()
	cfg.SpecAbsolutePath = name
	cfg.AvoidBuildIndex = true
	cfg.SpecInfo = nil
	// Register a complete index before following external edges. Eager lookup
	// here recurses through circular files before either index is cached.
	cfg.SkipExternalRefResolution = true
	cfg.Rolodex = nil
	idx := index.NewSpecIndexWithConfig(&node, &cfg)
	index.NewResolver(idx)
	buffer := &indexedBuffer{bufferFile: newBufferFile(name, string(content)), node: &node, idx: idx, content: string(content)}
	idx.BuildIndex()
	cfg.Rolodex = o.rolodex
	idx.SetRolodex(o.rolodex)
	cfg.SkipExternalRefResolution = false
	cached, loaded := o.files.LoadOrStore(name, buffer)
	if loaded {
		return cached.(*indexedBuffer), nil
	}
	o.rolodex.AddIndex(idx)
	return buffer, nil
}

type indexedBuffer struct {
	*bufferFile
	node    *yaml.Node
	idx     *index.SpecIndex
	content string
}

func (f *indexedBuffer) GetContent() string                        { return f.content }
func (f *indexedBuffer) GetFileExtension() index.FileExtension     { return index.ExtractFileType(f.name) }
func (f *indexedBuffer) GetFullPath() string                       { return f.name }
func (f *indexedBuffer) GetErrors() []error                        { return nil }
func (f *indexedBuffer) GetContentAsYAMLNode() (*yaml.Node, error) { return f.node, nil }
func (f *indexedBuffer) GetIndex() *index.SpecIndex                { return f.idx }
func (f *indexedBuffer) WaitForIndexing()                          {}
func (f *indexedBuffer) Name() string                              { return filepath.Base(f.name) }
func (f *indexedBuffer) ModTime() time.Time                        { return time.Time{} }
func (f *indexedBuffer) IsDir() bool                               { return false }
func (f *indexedBuffer) Sys() any                                  { return nil }
func (f *indexedBuffer) Mode() fs.FileMode                         { return 0444 }

func (o *documentOverlay) location(name string) string {
	if !remoteDocument(name) && !filepath.IsAbs(name) {
		if remoteDocument(o.base) {
			base, err := url.Parse(o.base)
			ref, refErr := url.Parse(name)
			if err == nil && refErr == nil {
				return base.ResolveReference(ref).String()
			}
		}
		name = filepath.Join(o.base, name)
	}
	return documentLocation(name)
}

func (o *documentOverlay) openContent(name string) (fs.File, error) {
	name = o.location(name)
	if !remoteDocument(name) && !filepath.IsAbs(name) {
		name = filepath.Join(o.base, name)
	}
	name = documentLocation(name)
	if content, ok := o.contents[name]; ok {
		return newBufferFile(name, content), nil
	}
	if !remoteDocument(name) {
		return os.Open(name)
	}
	response, err := utils.CreateRemoteURLHandlerWithContext(o.ctx, o.client)(name)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("load %s: HTTP %s", name, response.Status)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return newBufferFile(name, string(content)), nil
}

type bufferFile struct {
	*strings.Reader
	name string
}

func newBufferFile(name, content string) *bufferFile {
	return &bufferFile{Reader: strings.NewReader(content), name: name}
}
func (f *bufferFile) Close() error               { return nil }
func (f *bufferFile) Stat() (fs.FileInfo, error) { return bufferInfo{f.name, f.Size()}, nil }

type bufferInfo struct {
	name string
	size int64
}

func (i bufferInfo) Name() string       { return filepath.Base(i.name) }
func (i bufferInfo) Size() int64        { return i.size }
func (i bufferInfo) Mode() fs.FileMode  { return 0444 }
func (i bufferInfo) ModTime() time.Time { return time.Time{} }
func (i bufferInfo) IsDir() bool        { return false }
func (i bufferInfo) Sys() any           { return nil }

// Keep at most one root lint active. Changes cancel it and coalesce into the next
// snapshot, so typing cannot queue one complete graph per open document.
func (s *ServerState) scheduleRootDiagnostic(notify lsp.NotifyFunc) {
	s.rootMu.Lock()
	defer s.rootMu.Unlock()
	s.rootGeneration++
	if s.rootCancel != nil {
		s.rootCancel()
	}
	if s.rootRunning {
		return
	}
	s.rootRunning = true
	go s.runRootDiagnostics(notify)
}

func (s *ServerState) runRootDiagnostics(notify lsp.NotifyFunc) {
	for {
		s.rootMu.Lock()
		generation := s.rootGeneration
		parent := context.Background()
		if s.executionOptions != nil && s.executionOptions.Context != nil {
			parent = s.executionOptions.Context
		}
		ctx, cancel := context.WithCancel(parent)
		s.rootCancel = cancel
		s.rootMu.Unlock()

		publications := s.rootDiagnostics(ctx)
		cancel()
		s.rootMu.Lock()
		if generation != s.rootGeneration {
			s.rootMu.Unlock()
			continue
		}
		for uri, diagnostics := range publications {
			notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{URI: uri, Diagnostics: diagnostics})
		}
		s.rootCancel = nil
		s.rootRunning = false
		s.rootMu.Unlock()
		return
	}
}

func (s *ServerState) rootDiagnostics(ctx context.Context) map[string][]protocol.Diagnostic {
	root := documentLocation(s.lintRequest.MainSpecPath)
	rootURI := documentURI(root)
	contents := make(map[string]string)
	uris := map[string]string{root: rootURI}
	publications := map[string][]protocol.Diagnostic{rootURI: {}}
	s.documentStore.mu.RLock()
	for uri, doc := range s.documentStore.documents {
		doc.mu.RLock()
		contents[documentLocation(uri)] = doc.Content
		doc.mu.RUnlock()
		uris[documentLocation(uri)] = uri
		publications[uri] = []protocol.Diagnostic{}
	}
	s.documentStore.mu.RUnlock()
	rootURI = uris[root]
	fail := func(err error) map[string][]protocol.Diagnostic {
		publications[rootURI] = []protocol.Diagnostic{ConvertErrorIntoDiagnostic(err)}
		return publications
	}
	config, err := s.runtimeConfigForDocument(rootURI)
	if err != nil {
		return fail(err)
	}
	client, err := utils.CreateHTTPClientIfNeeded(config.httpClientConfig)
	if err != nil {
		return fail(err)
	}
	if client != nil {
		defer client.CloseIdleConnections()
		if client.Timeout == 0 {
			client.Timeout = 120 * time.Second
		}
	}
	base := config.config.Base
	if base == "" {
		base = documentBase(root)
	}
	overlay := &documentOverlay{contents: contents, base: base, ctx: ctx, client: client}
	file, err := overlay.Open(root)
	if err != nil {
		return fail(fmt.Errorf("load root specification: %w", err))
	}
	content, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		return fail(fmt.Errorf("read root specification: %w", err))
	}
	options := motor.ExecutionOptions{}
	if s.executionOptions != nil {
		options = *s.executionOptions
	}
	options.Context = ctx
	result, _, err := s.lintDocument(string(content), rootURI, root, overlay, &options)
	if err != nil {
		return fail(err)
	}
	defer result.ReleaseOwnedResources()
	for i := range result.Results {
		item := &result.Results[i]
		source := root
		if item.Origin != nil && item.Origin.AbsoluteLocation != "" {
			source = documentLocation(item.Origin.AbsoluteLocation)
		}
		if uri, ok := uris[source]; ok {
			diagnostic := ConvertResultIntoDiagnostic(item)
			anchorArazzoSourceDiagnostic(result, item, &diagnostic)
			publications[uri] = append(publications[uri], diagnostic)
		}
	}
	for _, err := range result.Errors {
		if err != nil {
			publications[rootURI] = append(publications[rootURI], ConvertErrorIntoDiagnostic(err))
		}
	}
	return publications
}
