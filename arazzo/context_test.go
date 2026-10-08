// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	validation "github.com/pb33f/libopenapi-validator/arazzo"
	source "github.com/pb33f/libopenapi/arazzo"
	"github.com/pb33f/testify/require"
)

func TestArazzoDetection(t *testing.T) {
	for _, tc := range []struct {
		spec, format string
		invalid      bool
	}{
		{"arazzo: 1.0.1", model.Arazzo10, false},
		{`{"arazzo":"1.1.0"}`, model.Arazzo11, false},
		{"arazzo: 1.2.0", model.Arazzo, true},
		{"arazzo: 1.0.-1", model.Arazzo, true},
		{"arazzo: 1.0.01", model.Arazzo10, false},
		{"arazzo: 1.1.0-preview", model.Arazzo11, false},
		{"arazzo: null", model.Arazzo, true},
		{"info: {description: arazzo}", "", false},
		{"openapi: 3.1.0\ninfo: {title: API}", "", false},
	} {
		format, err := DetectFormat([]byte(tc.spec))
		require.Equal(t, tc.format, format, tc.spec)
		require.Equal(t, tc.invalid, err != nil, tc.spec)
	}
}

func TestArazzoResolverHonorsLookupPolicyAndCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			started <- struct{}{}
			<-r.Context().Done()
			return
		}
		fmt.Fprint(w, "openapi: 3.1.0\ninfo: {title: API, version: '1'}\npaths: {}\n")
	}))
	defer server.Close()
	r := &Resolver{Client: server.Client()}
	_, err := r.Resolve(context.Background(), source.SourceRequest{URL: server.URL})
	require.ErrorIs(t, err, source.ErrUnresolvedSourceDesc)
	r.AllowRemote = true
	resolved, err := r.Resolve(context.Background(), source.SourceRequest{URL: server.URL})
	require.NoError(t, err)
	require.NotNil(t, resolved.RootNode)
	require.Equal(t, 1, r.Files)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := r.Resolve(ctx, source.SourceRequest{URL: server.URL + "/slow"}); done <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("request did not cancel")
	}
}

func TestArazzoResolverLocalFSAndLimits(t *testing.T) {
	r := &Resolver{AllowFiles: true, BasePath: "/specs", LocalFS: fstest.MapFS{"api.yaml": &fstest.MapFile{Data: []byte("openapi: 3.1.0\n")}}}
	_, err := r.Resolve(context.Background(), source.SourceRequest{URL: "file:///specs/api.yaml"})
	require.NoError(t, err)
	_, err = r.Resolve(context.Background(), source.SourceRequest{URL: "file:///private/api.yaml"})
	require.Error(t, err)
	r.LocalFS = nil
	_, err = r.Resolve(context.Background(), source.SourceRequest{URL: t.TempDir()})
	require.ErrorContains(t, err, "regular file")
	_, err = r.Resolve(context.Background(), source.SourceRequest{URL: "ftp://example.com/api.yaml"})
	require.ErrorContains(t, err, "scheme")
	path := filepath.Join(t.TempDir(), "large.yaml")
	require.NoError(t, os.WriteFile(path, []byte(strings.Repeat(" ", validation.DefaultLimits().MaxBytes+1)), 0600))
	_, err = r.Resolve(context.Background(), source.SourceRequest{URL: path})
	require.ErrorContains(t, err, "byte limit")
}

func TestArazzoLocationsPreserveEscapedKeysAndExternalSources(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("outputs:\n  a/b~c: value\n"), &root))
	c := &Context{RootNode: &root, results: make(map[string][]model.RuleFunctionResult)}
	c.add("arazzo-expression", "invalid", validation.Location{URI: "file:///external.yaml", Pointer: "/outputs/a~1b~0c"}, "file:///main.yaml", map[string]*SourceDocument{"file:///external.yaml": {Root: &root}})
	result := c.Results("arazzo-expression")[0]
	require.Equal(t, "$.outputs['a/b~c']", result.Path)
	require.Equal(t, 2, result.StartNode.Line)
	require.Equal(t, "/external.yaml", result.Origin.AbsoluteLocation)
}

func TestArazzoValidationRejectsAliasCycles(t *testing.T) {
	c, err := NewContext([]byte("arazzo: 1.0.1\ninfo: &cycle {title: *cycle, version: '1'}\n"), "cycle.yaml")
	require.NoError(t, err)
	err = c.Validate(context.Background(), "cycle.yaml", nil)
	require.NoError(t, err)
	require.False(t, c.Validation.Valid())
	require.NotEmpty(t, c.Results("arazzo-structure"))
}

func TestArazzoSizeLimitDoesNotRejectOtherDocumentFamilies(t *testing.T) {
	padding := "#" + strings.Repeat(" ", validation.DefaultLimits().MaxBytes) + "\n"
	format, err := DetectFormat([]byte("openapi: 3.1.0\ninfo: {title: Arazzo support, description: arazzo, version: '1'}\npaths: {}\n" + padding))
	require.NoError(t, err)
	require.Empty(t, format)
	_, err = NewContext([]byte("arazzo: 1.0.1\n"+padding), "workflow.yaml")
	require.ErrorContains(t, err, "byte limit")
}

func TestArazzoDocumentURIHonorsExplicitBase(t *testing.T) {
	for _, tc := range []struct{ filename, base, want string }{
		{"https://example.com/old/workflow.yaml", "", "https://example.com/old/workflow.yaml"},
		{"https://example.com/old/workflow.yaml", "https://other.example/specs", "https://other.example/specs/workflow.yaml"},
		{"https://example.com/old/workflow.yaml", "/specs", "file:///specs/workflow.yaml"},
		{"", "/specs", "file:///specs/stdin.yaml"},
		{"local/workflow.yaml", "https://example.com/specs", "https://example.com/specs/workflow.yaml"},
	} {
		got, err := DocumentURI(tc.filename, tc.base)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
}

func TestArazzoSourceContextIsBoundedAndAuthored(t *testing.T) {
	source := &SourceDocument{Content: "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine"}
	got := sourceContext(source, 5)
	require.Equal(t, 2, got.StartLine)
	require.Equal(t, []string{"two", "three", "four", "five", "six", "seven", "eight"}, got.Lines)
	for _, source := range []*SourceDocument{nil, {Content: "minified"}, {Content: "large\n" + strings.Repeat("x", 16<<10)}} {
		require.Empty(t, sourceContext(source, 1).Lines)
	}
}
