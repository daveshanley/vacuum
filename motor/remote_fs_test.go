// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"io"
	"io/fs"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/require"
)

type remoteTestFS struct {
	opens  atomic.Int32
	closes atomic.Int32
}

func (f *remoteTestFS) GetFiles() map[string]index.RolodexFile { return nil }

func (f *remoteTestFS) Open(name string) (fs.File, error) {
	f.opens.Add(1)
	file, err := (fstest.MapFS{"schema.yaml": &fstest.MapFile{Data: []byte("type: string\n")}}).Open(strings.TrimPrefix(name, "https://memory.invalid/"))
	if err != nil {
		return nil, err
	}
	return &remoteTestFile{File: file, closes: &f.closes}, nil
}

type remoteTestFile struct {
	fs.File
	closes *atomic.Int32
}

func (f *remoteTestFile) Close() error { f.closes.Add(1); return f.File.Close() }

func TestRemoteFSHandlerClosesAndCancels(t *testing.T) {
	files := &remoteTestFS{}
	ctx, cancel := context.WithCancel(context.Background())
	response, err := remoteFSHandler(ctx, files)("https://memory.invalid/schema.yaml")
	require.NoError(t, err)
	content, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "type: string\n", string(content))
	cancel()
	_, err = response.Body.Read(make([]byte, 1))
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, response.Body.Close())
	require.Equal(t, int32(1), files.closes.Load())
	_, err = remoteFSHandler(ctx, files)("https://memory.invalid/schema.yaml")
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, int32(1), files.opens.Load())
}

func TestRemoteReferencesUseSuppliedFilesystem(t *testing.T) {
	files := &remoteTestFS{}
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		Spec:         []byte("openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Value:\n      $ref: 'https://memory.invalid/schema.yaml'\n"),
		SpecFileName: "root.yaml", Base: t.TempDir(), AllowLookup: true, RolodexFS: files,
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"schema-description": {
			Id: "schema-description", Given: "$.components.schemas.Value", Resolved: true, Severity: model.SeverityError,
			Then: model.RuleAction{Field: "description", Function: "defined"},
		}}},
	})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.NotZero(t, files.opens.Load())
	require.Equal(t, files.opens.Load(), files.closes.Load())
	require.Len(t, result.Results, 1)
	require.Equal(t, "schema-description", result.Results[0].Rule.Id)
}

func TestRemoteFSHandlerPreservesOpenErrors(t *testing.T) {
	files := &remoteTestFS{}
	response, err := remoteFSHandler(context.Background(), files)("https://memory.invalid/missing.yaml")
	require.Nil(t, response)
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.Zero(t, files.closes.Load())
}

type cancelDuringOpenFS struct {
	*remoteTestFS
	cancel context.CancelFunc
	seen   context.Context
}

func (f *cancelDuringOpenFS) OpenWithContext(ctx context.Context, name string) (fs.File, error) {
	f.seen = ctx
	file, err := f.Open(name)
	f.cancel()
	return file, err
}

func TestRemoteFSHandlerClosesFileWhenOpenIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	files := &cancelDuringOpenFS{remoteTestFS: &remoteTestFS{}, cancel: cancel}
	response, err := remoteFSHandler(ctx, files)("https://memory.invalid/schema.yaml")
	require.Nil(t, response)
	require.ErrorIs(t, err, context.Canceled)
	require.Same(t, ctx, files.seen)
	require.Equal(t, int32(1), files.closes.Load())
}
