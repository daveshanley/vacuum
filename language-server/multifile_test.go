// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package languageserver

import (
	"context"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/require"
)

const multifileRoot = `openapi: 3.0.3
info:
  title: Synthetic API
  version: '1'
paths: {}
components:
  schemas:
    Widget:
      $ref: './widget.yaml'
`
const multifileSchema = "type: object\nproperties:\n  name:\n    type: string\n"

func newMultifileTestServer(root string, httpConfig utils.HTTPClientConfig) *ServerState {
	state := NewServer("test", &utils.LintFileRequest{
		MainSpecPath: root, Remote: true, TimeoutFlag: 5, LookupTimeoutFlag: 500,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), HTTPClientConfig: httpConfig,
	})
	state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
		return &rulesets.RuleSet{Rules: map[string]*model.Rule{
			"root-description":   {Id: "root-description", Given: "$.info", Resolved: true, Severity: model.SeverityError, Then: model.RuleAction{Field: "description", Function: "truthy"}},
			"schema-description": {Id: "schema-description", Given: "$.components.schemas.*.properties.name", Resolved: true, Severity: model.SeverityWarn, Then: model.RuleAction{Field: "description", Function: "truthy"}},
		}}
	}
	return state
}

func TestMultifileDiagnosticsUsesUnsavedBuffersAndRoutesSources(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "API spec.yaml")
	ref := filepath.Join(dir, "widget.yaml")
	require.NoError(t, os.WriteFile(root, []byte(strings.Replace(multifileRoot, "  title:", "  description: saved\n  title:", 1)), 0600))
	require.NoError(t, os.WriteFile(ref, []byte(strings.Replace(multifileSchema, "    type:", "    description: saved\n    type:", 1)), 0600))
	state := newMultifileTestServer(root, utils.HTTPClientConfig{})
	rootURI, refURI := fileURI(root), fileURI(ref)
	state.documentStore.Add(rootURI, multifileRoot)
	state.documentStore.Add(refURI, multifileSchema)
	unrelatedURI := fileURI(filepath.Join(dir, "unrelated.yaml"))
	state.documentStore.Add(unrelatedURI, "not an API")
	publications := state.rootDiagnostics(context.Background())
	require.Len(t, publications[rootURI], 1, "%+v", publications)
	require.Equal(t, "root-description", publications[rootURI][0].Code.Value)
	require.Len(t, publications[refURI], 1, "%+v", publications)
	require.Equal(t, "schema-description", publications[refURI][0].Code.Value)
	require.Equal(t, uint32(3), publications[refURI][0].Range.Start.Line)
	require.Empty(t, publications[unrelatedURI])

	state.documentStore.Add(rootURI, strings.Replace(multifileRoot, "  title:", "  description: unsaved\n  title:", 1))
	state.documentStore.Add(refURI, strings.Replace(multifileSchema, "    type:", "    description: unsaved\n    type:", 1))
	publications = state.rootDiagnostics(context.Background())
	require.Empty(t, publications[rootURI])
	require.Empty(t, publications[refURI])
}

func TestMultifileDiagnosticsRefreshesAllDocumentsAndClearsRemovedReference(t *testing.T) {
	dir := t.TempDir()
	root, ref := filepath.Join(dir, "root.yaml"), filepath.Join(dir, "widget.yaml")
	state := newMultifileTestServer(root, utils.HTTPClientConfig{})
	rootDoc := state.documentStore.Add(fileURI(root), multifileRoot)
	refDoc := state.documentStore.Add(fileURI(ref), multifileSchema)
	published := make(chan protocol.PublishDiagnosticsParams, 8)
	notify := func(_ string, params any) { published <- params.(protocol.PublishDiagnosticsParams) }
	collect := func() map[string][]protocol.Diagnostic {
		t.Helper()
		results := make(map[string][]protocol.Diagnostic)
		deadline := time.NewTimer(5 * time.Second)
		defer deadline.Stop()
		for len(results) < 2 {
			select {
			case p := <-published:
				results[p.URI] = p.Diagnostics
			case <-deadline.C:
				t.Fatal("timed out waiting for root and reference diagnostics")
			}
		}
		return results
	}
	state.runDiagnostic(refDoc, notify)
	initial := collect()
	require.Len(t, initial[rootDoc.URI], 1)
	require.Len(t, initial[refDoc.URI], 1)
	require.NoError(t, applyDocumentChanges(rootDoc, []any{protocol.TextDocumentContentChangeEventWhole{Text: "openapi: 3.0.3\ninfo: {title: Synthetic, version: '1', description: Present}\npaths: {}\n"}}))
	state.runDiagnostic(rootDoc, notify)
	refreshed := collect()
	require.Empty(t, refreshed[rootDoc.URI])
	require.Empty(t, refreshed[refDoc.URI])
}

func TestMultifileDiagnosticsMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing.yaml")
	state := newMultifileTestServer(root, utils.HTTPClientConfig{})
	refURI := fileURI(filepath.Join(filepath.Dir(root), "widget.yaml"))
	state.documentStore.Add(refURI, multifileSchema)
	publications := state.rootDiagnostics(context.Background())
	require.Len(t, publications[fileURI(root)], 1)
	require.Contains(t, publications[fileURI(root)][0].Message, "load root specification")
	require.Empty(t, publications[refURI])
}

func TestMultifileDiagnosticsRemoteRootAndTLS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/root.yaml":
			if got := r.Header.Get("User-Agent"); got != "vacuum-linter/1.0" {
				t.Errorf("unexpected User-Agent: %q", got)
			}
			_, _ = io.WriteString(w, multifileRoot)
		case "/api/widget.yaml":
			_, _ = io.WriteString(w, multifileSchema)
		default:
			http.NotFound(w, r)
		}
	})
	server := httptest.NewTLSServer(handler)
	defer server.Close()
	root, ref := server.URL+"/api/root.yaml", server.URL+"/api/widget.yaml"
	state := newMultifileTestServer(root, utils.HTTPClientConfig{})
	publications := state.rootDiagnostics(context.Background())
	require.Len(t, publications[root], 1)
	require.Contains(t, publications[root][0].Message, "certificate")

	caPath := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600))
	state = newMultifileTestServer(root, utils.HTTPClientConfig{CAFile: caPath})
	state.documentStore.Add(ref, strings.Replace(multifileSchema, "    type:", "    description: unsaved\n    type:", 1))
	publications = state.rootDiagnostics(context.Background())
	require.Len(t, publications[root], 1, "%+v", publications)
	require.Equal(t, "root-description", publications[root][0].Code.Value)
	require.Empty(t, publications[ref])

	state = newMultifileTestServer(server.URL+"/missing.yaml", utils.HTTPClientConfig{CAFile: caPath})
	publications = state.rootDiagnostics(context.Background())
	require.Contains(t, publications[server.URL+"/missing.yaml"][0].Message, "404")
}

func TestMultifileDiagnosticsNestedRelativeReference(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.yaml")
	responsePath := filepath.Join(dir, "parts", "response.yaml")
	schemaPath := filepath.Join(dir, "parts", "schema.yaml")
	require.NoError(t, os.Mkdir(filepath.Dir(responsePath), 0700))
	root := `openapi: 3.0.3
info: {title: Synthetic, version: '1'}
paths:
  /widgets:
    get:
      responses:
        '200':
          $ref: './parts/response.yaml'
`
	response := "description: Response\ncontent:\n  application/json:\n    schema:\n      $ref: './schema.yaml'\n"
	require.NoError(t, os.WriteFile(rootPath, []byte(root), 0600))
	require.NoError(t, os.WriteFile(responsePath, []byte(response), 0600))
	require.NoError(t, os.WriteFile(schemaPath, []byte(multifileSchema), 0600))
	state := newMultifileTestServer(rootPath, utils.HTTPClientConfig{})
	state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
		return &rulesets.RuleSet{Rules: map[string]*model.Rule{
			rulesets.OasSchemaCheck: rulesets.GetSchemaTypeCheckRule(),
		}}
	}
	state.documentStore.Add(fileURI(schemaPath), strings.Replace(multifileSchema, "type: string", "type: invalid", 1))
	publications := state.rootDiagnostics(context.Background())
	require.Empty(t, publications[fileURI(rootPath)], "%+v", publications)
	require.Len(t, publications[fileURI(schemaPath)], 1, "%+v", publications)
	require.Equal(t, rulesets.OasSchemaCheck, publications[fileURI(schemaPath)][0].Code.Value)
	require.Equal(t, uint32(3), publications[fileURI(schemaPath)][0].Range.Start.Line)

	state.documentStore.Add(fileURI(schemaPath), multifileSchema)
	publications = state.rootDiagnostics(context.Background())
	require.Empty(t, publications[fileURI(rootPath)])
	require.Empty(t, publications[fileURI(schemaPath)])
}

func TestMultifileDiagnosticsCancelsOldFetchAndPublishesLatestBuffer(t *testing.T) {
	entered := make(chan struct{})
	canceled := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		select {
		case <-r.Context().Done():
			close(canceled)
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	root := server.URL + "/root.yaml"
	state := newMultifileTestServer(root, utils.HTTPClientConfig{})
	published := make(chan protocol.PublishDiagnosticsParams, 4)
	notify := func(_ string, params any) { published <- params.(protocol.PublishDiagnosticsParams) }
	state.runDiagnostic(&Document{URI: root}, notify)
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("root fetch did not start")
	}
	doc := state.documentStore.Add(root, "openapi: 3.0.3\ninfo: {title: Synthetic, version: '1', description: Present}\npaths: {}\n")
	state.runDiagnostic(doc, notify)
	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("obsolete root fetch was not canceled")
	}
	select {
	case p := <-published:
		require.Equal(t, root, p.URI)
		require.Empty(t, p.Diagnostics)
	case <-time.After(5 * time.Second):
		t.Fatal("latest buffer diagnostics were not published")
	}
	// Publication and the running flag use the same lock, so this also waits for
	// the completed generation to leave its publish section.
	state.rootMu.Lock()
	running := state.rootRunning
	state.rootMu.Unlock()
	require.False(t, running)
	require.Empty(t, published, "obsolete generation must not publish diagnostics")
}

func TestMultifileDiagnosticsCircularEditorReferences(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root.yaml")
	state := newMultifileTestServer(root, utils.HTTPClientConfig{})
	state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
		return &rulesets.RuleSet{Rules: map[string]*model.Rule{}}
	}
	state.documentStore.Add(fileURI(root), `openapi: 3.0.3
info: {title: Synthetic, version: '1'}
paths: {}
components:
  schemas:
    A:
      $ref: './a.yaml#/A'
`)
	state.documentStore.Add(fileURI(filepath.Join(dir, "a.yaml")), `A:
  type: object
  required: [b]
  properties:
    b:
      $ref: './b.yaml#/B'
`)
	state.documentStore.Add(fileURI(filepath.Join(dir, "b.yaml")), `B:
  type: object
  required: [a]
  properties:
    a:
      $ref: './a.yaml#/A'
`)
	done := make(chan map[string][]protocol.Diagnostic, 1)
	go func() { done <- state.rootDiagnostics(context.Background()) }()
	select {
	case publications := <-done:
		var circularMessages []string
		for _, diagnostics := range publications {
			for _, diagnostic := range diagnostics {
				require.NotContains(t, diagnostic.Message, "cannot resolve reference")
				if strings.Contains(diagnostic.Message, "circular reference") {
					circularMessages = append(circularMessages, diagnostic.Message)
				}
			}
		}
		require.NotEmpty(t, circularMessages, "expected circular diagnostics from editor-only references")
	case <-time.After(5 * time.Second):
		t.Fatal("circular editor references did not finish")
	}
}

func TestMultifileDiagnosticsClosedReferenceReturnsToDisk(t *testing.T) {
	dir := t.TempDir()
	root, ref := filepath.Join(dir, "root.yaml"), filepath.Join(dir, "widget.yaml")
	require.NoError(t, os.WriteFile(ref, []byte(multifileSchema), 0600))
	state := newMultifileTestServer(root, utils.HTTPClientConfig{})
	state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
		return &rulesets.RuleSet{Rules: map[string]*model.Rule{
			"schema-description": {Id: "schema-description", Given: "$.components.schemas.*", Resolved: true, Severity: model.SeverityWarn, Then: model.RuleAction{Field: "description", Function: "truthy"}},
		}}
	}
	rootURI, refURI := fileURI(root), fileURI(ref)
	state.documentStore.Add(rootURI, multifileRoot)
	state.documentStore.Add(refURI, "description: Unsaved\n"+multifileSchema)
	publications := state.rootDiagnostics(context.Background())
	require.Empty(t, publications[rootURI])
	require.Empty(t, publications[refURI])

	state.documentStore.Remove(refURI)
	publications = state.rootDiagnostics(context.Background())
	// A missing field on the referenced schema itself anchors to the root's
	// reference node. Its return proves the closed buffer no longer masks disk.
	require.Len(t, publications[rootURI], 1)
	require.Equal(t, "schema-description", publications[rootURI][0].Code.Value)
	require.NotContains(t, publications, refURI)
}

func TestMultifileDiagnosticsCrossFilePathItems(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root.yaml")
	state := newMultifileTestServer(root, utils.HTTPClientConfig{})
	state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
		rule := rulesets.GetOperationIdRule()
		return &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}
	}
	state.documentStore.Add(fileURI(root), `openapi: 3.0.3
info: {title: Synthetic, version: '1'}
paths:
  /one:
    $ref: './a.yaml#/paths/~1one'
  /two:
    $ref: './b.yaml#/paths/~1two'
`)
	state.documentStore.Add(fileURI(filepath.Join(dir, "a.yaml")), `paths:
  /one:
    get:
      responses:
        '200': {description: OK}
  /two:
    $ref: './b.yaml#/paths/~1two'
`)
	state.documentStore.Add(fileURI(filepath.Join(dir, "b.yaml")), `paths:
  /two:
    get:
      responses:
        '200': {description: OK}
  /one:
    $ref: './a.yaml#/paths/~1one'
`)
	done := make(chan map[string][]protocol.Diagnostic, 1)
	go func() { done <- state.rootDiagnostics(context.Background()) }()
	select {
	case publications := <-done:
		var found int
		for _, diagnostics := range publications {
			for _, diagnostic := range diagnostics {
				require.NotNil(t, diagnostic.Code, "%+v", diagnostic)
				require.Equal(t, rulesets.GetOperationIdRule().Id, diagnostic.Code.Value, "%+v", diagnostic)
				found++
			}
		}
		require.Equal(t, 2, found, "both operations must retain their missing operationId diagnostic")
	case <-time.After(5 * time.Second):
		t.Fatal("cross-file path items did not finish")
	}
}

func TestMultifileDiagnosticsSharedSchemaDiamond(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root.yaml")
	state := newMultifileTestServer(root, utils.HTTPClientConfig{})
	state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
		return &rulesets.RuleSet{Rules: map[string]*model.Rule{rulesets.OasSchemaCheck: rulesets.GetSchemaTypeCheckRule()}}
	}
	state.documentStore.Add(fileURI(root), `openapi: 3.0.3
info: {title: Synthetic, version: '1'}
paths: {}
components:
  schemas:
    A: {$ref: './a.yaml'}
    B: {$ref: './b.yaml'}
`)
	for _, name := range []string{"a.yaml", "b.yaml"} {
		state.documentStore.Add(fileURI(filepath.Join(dir, name)), "type: object\nproperties:\n  shared:\n    $ref: './shared.yaml'\n")
	}
	sharedURI := fileURI(filepath.Join(dir, "shared.yaml"))
	state.documentStore.Add(sharedURI, strings.Replace(multifileSchema, "type: string", "type: invalid", 1))
	publications := state.rootDiagnostics(context.Background())
	for uri, diagnostics := range publications {
		if uri != sharedURI {
			require.Empty(t, diagnostics, "unexpected diagnostic source %s", uri)
		}
	}
	require.NotEmpty(t, publications[sharedURI])
	for _, diagnostic := range publications[sharedURI] {
		require.Equal(t, rulesets.OasSchemaCheck, diagnostic.Code.Value)
		require.Equal(t, uint32(3), diagnostic.Range.Start.Line)
	}
}
