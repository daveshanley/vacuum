// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	ppmodel "github.com/pb33f/doctor/printingpress/model"
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
	cmd.SetArgs([]string{"--stdout", "--no-style", specPath, filepath.Join(dir, "overlay.yaml")})

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

func TestRunDocsAggregateGroupsMixedContractsWithFamilyDiagnostics(t *testing.T) {
	root := t.TempDir()
	openAPIPath := filepath.Join(root, "services", "generic", "http", "v1", "openapi.yaml")
	asyncAPIPath := filepath.Join(root, "services", "generic", "events", "v1", "asyncapi.yaml")
	writeTestFile(t, openAPIPath, `openapi: 3.1.0
info:
  title: Generic Catalog HTTP API
  version: 1.0.0
  description: HTTP contract.
  x-owner:
    service: Generic Catalog
paths: {}
`)
	writeTestFile(t, asyncAPIPath, `asyncapi: 3.1.0
info:
  title: Generic Catalog Published Events
  version: 1.0.0
  description: Published event contract.
  x-owner:
    service: Generic Catalog
tags:
  - name: events
    description: Published events.
servers:
  production:
    host: broker.example.net
    protocol: mqtt
channels: {}
operations: {}
`)

	configPath := filepath.Join(root, "printing-press.yaml")
	writeTestFile(t, configPath, `grouping:
  serviceIdentity:
    metadataPointers:
      - /info/x-owner/service
  contractRoles:
    - pattern: "**/http/**"
      role: http-api
      contractID: http-api
      default: true
    - pattern: "**/events/**"
      role: published-events
      contractID: published-events
`)
	openAPIRuleset := writeDocsNamedRuleset(t, root, "openapi-ruleset.yaml", "custom-openapi-family-rule")
	outputDir := filepath.Join(t.TempDir(), "docs")
	cmd := GetDocsCommand()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := runDocs(cmd, root, &docsOptions{
		outputDir:      outputDir,
		docsConfigPath: configPath,
		openAPIRuleset: openAPIRuleset,
		noLLM:          true,
		noJSON:         true,
		noLogo:         true,
		includeSpec:    true,
	})

	require.NoError(t, err)
	catalogHTML, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	require.NoError(t, err)
	defaultHTTPHref := "services/generic-catalog/versions/1-0-0/specs/generic-catalog-http-api/index.html"
	assert.Contains(t, string(catalogHTML), `href="`+defaultHTTPHref+`"`)

	serviceSpecsRoot := filepath.Join(outputDir, "services", "generic-catalog", "versions", "1-0-0", "specs")
	httpRoot := filepath.Join(serviceSpecsRoot, "generic-catalog-http-api")
	eventRoot := filepath.Join(serviceSpecsRoot, "generic-catalog-published-events")
	httpOverview := readDocsContractPage(t, filepath.Join(httpRoot, "index.html"))
	assert.Equal(t, "API OVERVIEW", httpOverview.overviewLabel)
	assert.Equal(t, "Generic Catalog HTTP API", httpOverview.serviceName)
	assertDocsContractTree(t, httpOverview.groups, "http-api")

	eventOverview := readDocsContractPage(t, filepath.Join(eventRoot, "index.html"))
	assert.Equal(t, "EVENT OVERVIEW", eventOverview.overviewLabel)
	assert.Equal(t, "Generic Catalog HTTP API", eventOverview.serviceName)
	assertDocsContractTree(t, eventOverview.groups, "published-events")

	openAPIDiagnostics := readTestFile(t, filepath.Join(httpRoot, "data", "pages", "diagnostics.js"))
	assert.Contains(t, openAPIDiagnostics, "custom-openapi-family-rule")
	assert.NotContains(t, openAPIDiagnostics, rulesets.AsyncAPIInfoContact)
	asyncAPIDiagnostics := readTestFile(t, filepath.Join(eventRoot, "data", "pages", "diagnostics.js"))
	assert.Contains(t, asyncAPIDiagnostics, rulesets.AsyncAPIInfoContact)
	assert.NotContains(t, asyncAPIDiagnostics, "custom-openapi-family-rule")

	legacyRuleset := writeDocsNamedRuleset(t, root, "legacy-ruleset.yaml", "legacy-docs-rule")
	legacyOutput := filepath.Join(t.TempDir(), "legacy-docs")
	legacyCmd := GetDocsCommand()
	legacyCmd.SetOut(io.Discard)
	legacyCmd.SetErr(io.Discard)
	legacyCmd.Flags().String("ruleset", "", "")
	require.NoError(t, legacyCmd.Flags().Set("ruleset", legacyRuleset))
	legacyFlags := docsLintFlags(legacyCmd)
	assert.Equal(t, legacyRuleset, legacyFlags.RulesetFlag)
	legacyDiagnostics, err := newDocsDiagnosticsContext(legacyFlags, utils.HTTPClientConfig{}, nil, true)
	require.NoError(t, err)
	legacyResults, err := legacyDiagnostics.lintSpec([]byte(docsDiagnosticsSpec("Legacy HTTP API")), openAPIPath)
	require.NoError(t, err)
	require.NotEmpty(t, legacyResults)
	assert.Equal(t, "legacy-docs-rule", legacyResults[0].RuleId)
	require.NoError(t, runDocs(legacyCmd, root, &docsOptions{
		outputDir:      legacyOutput,
		docsConfigPath: configPath,
		noLLM:          true,
		noJSON:         true,
		noLogo:         true,
	}))
	legacyDiagnosticsPath := filepath.Join(legacyOutput, "services", "generic-catalog", "versions", "1-0-0", "specs", "generic-catalog-http-api", "data", "pages", "diagnostics.js")
	assert.Contains(t, readTestFile(t, legacyDiagnosticsPath), "legacy-docs-rule")
}

type docsContractPage struct {
	overviewLabel string
	serviceName   string
	groups        []*ppmodel.SiteContractGroup
}

func readDocsContractPage(t *testing.T, pagePath string) docsContractPage {
	t.Helper()
	rendered := readTestFile(t, pagePath)
	contractPayload := docsHTMLAttribute(t, rendered, "data-pp-contracts")
	var groups []*ppmodel.SiteContractGroup
	require.NoError(t, json.Unmarshal([]byte(contractPayload), &groups))
	return docsContractPage{
		overviewLabel: docsHTMLAttribute(t, rendered, "data-pp-overview-label"),
		serviceName:   docsHTMLAttribute(t, rendered, "data-pp-service-name"),
		groups:        groups,
	}
}

func docsHTMLAttribute(t *testing.T, rendered, name string) string {
	t.Helper()
	marker := name + `="`
	start := strings.Index(rendered, marker)
	require.GreaterOrEqual(t, start, 0, "missing %s", name)
	valueStart := start + len(marker)
	valueEnd := strings.Index(rendered[valueStart:], `"`)
	require.GreaterOrEqual(t, valueEnd, 0, "unterminated %s", name)
	return html.UnescapeString(rendered[valueStart : valueStart+valueEnd])
}

func assertDocsContractTree(t *testing.T, groups []*ppmodel.SiteContractGroup, activeRole string) {
	t.Helper()
	require.Len(t, groups, 2)
	assert.Equal(t, ppmodel.ContractRoleHTTPAPI, groups[0].Role)
	assert.Equal(t, "HTTP API", groups[0].Label)
	require.Len(t, groups[0].Contracts, 1)
	assert.Equal(t, "http-api", groups[0].Contracts[0].ID)
	assert.Equal(t, "Generic Catalog HTTP API", groups[0].Contracts[0].Label)
	assert.Equal(t, activeRole == "http-api", groups[0].Contracts[0].Active)
	assert.Equal(t, ppmodel.ContractRolePublishedEvents, groups[1].Role)
	assert.Equal(t, "Published Events", groups[1].Label)
	require.Len(t, groups[1].Contracts, 1)
	assert.Equal(t, "published-events", groups[1].Contracts[0].ID)
	assert.Equal(t, "Generic Catalog Published Events", groups[1].Contracts[0].Label)
	assert.Equal(t, activeRole == "published-events", groups[1].Contracts[0].Active)
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
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
