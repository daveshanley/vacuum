// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestSelectRuleSetForBuildResults_AsyncAPIDefault(t *testing.T) {
	ruleSet, specFormat, err := selectRuleSetForBuildResults(true, false, "", []byte(cmdAsyncAPI31Fixture), false, utils.HTTPClientConfig{}, nil)

	require.NoError(t, err)
	require.NotNil(t, ruleSet)
	assert.Equal(t, model.AsyncAPI31, specFormat)
	assert.Contains(t, ruleSet.Rules, rulesets.AsyncAPI3DocumentResolved)
	assert.NotContains(t, ruleSet.Rules, rulesets.OperationSuccessResponse)
}

func TestDocsDiagnosticsSelectsRuleSetForEachSpec(t *testing.T) {
	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	ctx := &docsDiagnosticsContext{
		flags: &LintFlags{
			RemoteFlag:     true,
			SilentFlag:     true,
			PipelineOutput: true,
		},
		selectedRuleset: defaultRuleSets.GenerateOpenAPIRecommendedRuleSet(),
	}

	asyncRules, err := ctx.ruleSetForSpec([]byte(cmdAsyncAPI31Fixture))
	require.NoError(t, err)
	assert.Contains(t, asyncRules.Rules, rulesets.AsyncAPI3DocumentResolved)
	assert.NotContains(t, asyncRules.Rules, rulesets.OperationSuccessResponse)

	openAPIRules, err := ctx.ruleSetForSpec([]byte(`openapi: 3.1.0
info:
  title: HTTP API
  version: 1.0.0
paths: {}`))
	require.NoError(t, err)
	assert.Contains(t, openAPIRules.Rules, rulesets.OperationSuccessResponse)
	assert.NotContains(t, openAPIRules.Rules, rulesets.AsyncAPI3DocumentResolved)
}

func TestRejectAsyncAPIForOpenAPICommand(t *testing.T) {
	err := rejectAsyncAPIForOpenAPICommand("bundle", []byte(cmdAsyncAPI31Fixture))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "only supports OpenAPI")
}

func TestRejectAsyncAPIForOpenAPICommandRejectsMalformedAsyncAPI(t *testing.T) {
	err := rejectAsyncAPIForOpenAPICommand("bundle", []byte(cmdMalformedAsyncAPI31Fixture))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "only supports OpenAPI")
}

func TestLintMultipleFilesReturnsInputErrorForAsyncAPI2WithFailSeverityNone(t *testing.T) {
	dir := t.TempDir()
	openAPIPath := filepath.Join(dir, "openapi.yaml")
	asyncAPIPath := filepath.Join(dir, "asyncapi.yaml")
	writeTestFile(t, openAPIPath, `
openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`)
	writeTestFile(t, asyncAPIPath, `
asyncapi: 2.6.0
info:
  title: Legacy Events
  version: 1.0.0
channels: {}
`)

	cmd := GetLintCommand()
	output := bytes.NewBufferString("")
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"--fail-severity", "none", "--no-style", "--silent", openAPIPath, asyncAPIPath})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "input/tool errors")
	var exitErr *ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.Code)
}

func TestBundleAsyncAPIStdoutWritesRejectionToStderr(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "asyncapi.yaml")
	writeTestFile(t, specPath, cmdAsyncAPI31Fixture)

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{specPath, "--stdout"})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "only supports OpenAPI")
}

func TestBundleMalformedAsyncAPIStdoutWritesRejectionToStderr(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "asyncapi.yaml")
	writeTestFile(t, specPath, cmdMalformedAsyncAPI31Fixture)

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{specPath, "--stdout"})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "only supports OpenAPI")
}

func TestApplyOverlayAsyncAPIStdoutWritesRejectionToStderr(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "asyncapi.yaml")
	writeTestFile(t, specPath, cmdAsyncAPI31Fixture)

	cmd := GetApplyOverlayCommand()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--stdout", "--no-style", "--include-tag", "public", "--tag-match", "all", "--prune-unused", specPath, filepath.Join(dir, "overlay.yaml")})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "only supports OpenAPI")
}

func TestRunDocsSupportsAsyncAPIWithDiagnosticsAndIncludedContract(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "asyncapi.yaml")
	writeTestFile(t, specPath, cmdAsyncAPI31Fixture)
	outputDir := filepath.Join(dir, "docs")

	cmd := GetDocsCommand()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := runDocs(cmd, specPath, &docsOptions{
		outputDir:   outputDir,
		noLLM:       true,
		noJSON:      true,
		noLogo:      true,
		includeSpec: true,
	})

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(outputDir, "index.html"))
	assert.FileExists(t, filepath.Join(outputDir, "spec", "asyncapi.yaml"))
	indexHTML, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(indexHTML), "<h1>Events</h1>")
	assert.Contains(t, string(indexHTML), "mqtt://api.example.com")
	assert.Contains(t, string(indexHTML), `href="spec/asyncapi.yaml"`)
}

func TestRunDocsReportsMalformedAsyncAPI(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "asyncapi.yaml")
	writeTestFile(t, specPath, cmdMalformedAsyncAPI31Fixture)

	cmd := GetDocsCommand()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := runDocs(cmd, specPath, &docsOptions{
		outputDir: filepath.Join(dir, "docs"),
		noLLM:     true,
		noJSON:    true,
		noLogo:    true,
	})

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "only supports OpenAPI")
}

func TestRunDocsAggregateSupportsMixedOpenAPIAndAsyncAPI(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "asyncapi.yaml"), cmdAsyncAPI31Fixture)
	writeTestFile(t, filepath.Join(dir, "openapi.yaml"), `openapi: 3.1.0
info:
  title: HTTP API
  version: 1.0.0
  description: HTTP contract.
paths: {}`)
	outputDir := filepath.Join(t.TempDir(), "docs")
	cmd := GetDocsCommand()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := runDocs(cmd, dir, &docsOptions{
		outputDir:   outputDir,
		noLLM:       true,
		noJSON:      true,
		noLogo:      true,
		includeSpec: true,
	})

	require.NoError(t, err)
	indexHTML, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(indexHTML), "services/openapi/")
	assert.Contains(t, string(indexHTML), "services/asyncapi/")
	assert.True(t, docsOutputContainsFile(t, outputDir, "asyncapi.yaml"))
}

func docsOutputContainsFile(t *testing.T, root, name string) bool {
	t.Helper()
	found := false
	require.NoError(t, filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == name && filepath.Base(filepath.Dir(path)) == "spec" {
			found = true
		}
		return nil
	}))
	return found
}

func captureOSStreams(t *testing.T, fn func()) (string, string) {
	t.Helper()

	originalStdout := os.Stdout
	originalStderr := os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err)
	stderrReader, stderrWriter, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()

	fn()

	require.NoError(t, stdoutWriter.Close())
	require.NoError(t, stderrWriter.Close())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	_, _ = io.Copy(&stdout, stdoutReader)
	_, _ = io.Copy(&stderr, stderrReader)
	return stdout.String(), stderr.String()
}

const cmdAsyncAPI31Fixture = `asyncapi: 3.1.0
info:
  title: Events
  version: 1.0.0
  description: Event contract.
  contact:
    name: API Team
    url: https://example.com
    email: api@example.com
  license:
    name: MIT
tags:
  - name: events
    description: Event APIs.
servers:
  production:
    host: api.example.com
    protocol: mqtt
channels: {}
operations: {}
`

const cmdMalformedAsyncAPI31Fixture = `asyncapi: 3.1.0
info:
  title: [
`
