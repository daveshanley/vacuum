// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	ppconfig "github.com/pb33f/doctor/printingpress/config"
	ppmodel "github.com/pb33f/doctor/printingpress/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsCommandKeepsLiveProgressFlags(t *testing.T) {
	cmd := GetDocsCommand()

	assert.Nil(t, cmd.Flags().Lookup("theme"))
	assert.NotNil(t, cmd.Flags().Lookup("metrics"))
}

func TestApplyDocsConfigDescriptionDoesNotDependOnTitle(t *testing.T) {
	cmd := GetDocsCommand()
	require.NoError(t, cmd.Flags().Set("title", "CLI Title"))

	opts := &docsOptions{title: "CLI Title"}
	applyDocsConfigToOptions(cmd, opts, &ppconfig.File{
		Title:       "Config Title",
		Description: "Config description",
	})

	assert.Equal(t, "CLI Title", opts.title)
	assert.Equal(t, "Config description", opts.description)
}

func TestResolveDocsInputUsesConfigScanRoot(t *testing.T) {
	input, err := resolveDocsInput("", &ppconfig.File{
		Scan: ppconfig.ScanConfig{Root: "/tmp/apis"},
	})

	require.NoError(t, err)
	assert.Equal(t, "/tmp/apis", input)
}

func TestResolveDocsInputRequiresInput(t *testing.T) {
	_, err := resolveDocsInput("", nil)

	require.Error(t, err)
	assert.ErrorContains(t, err, "Supply an OpenAPI spec path, URL, or directory to generate the most fly")
	assert.ErrorContains(t, err, "vacuum docs ./openapi.yaml")
	assert.ErrorContains(t, err, "--docs-config printing-press.yaml")
}

func TestDetectDocsInputMode(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte("openapi: 3.1.0\n"), 0o644))

	mode, path, err := detectDocsInputMode("https://example.com/openapi.yaml")
	require.NoError(t, err)
	assert.Equal(t, docsInputSingle, mode)
	assert.Equal(t, "https://example.com/openapi.yaml", path)

	mode, path, err = detectDocsInputMode(dir)
	require.NoError(t, err)
	assert.Equal(t, docsInputAggregate, mode)
	assert.Equal(t, dir, path)

	mode, path, err = detectDocsInputMode(specPath)
	require.NoError(t, err)
	assert.Equal(t, docsInputSingle, mode)
	assert.Equal(t, specPath, path)
}

func TestDocsDiagnosticsFingerprintIncludesRuleContents(t *testing.T) {
	flags := &LintFlags{RemoteFlag: true, TimeoutFlag: 5, LookupTimeoutFlag: 500}
	first := docsDiagnosticsFingerprint(true, flags, docsFingerprintRuleSet("same id", "first message"))
	second := docsDiagnosticsFingerprint(true, flags, docsFingerprintRuleSet("same id", "second message"))

	assert.NotEqual(t, first, second)
}

func TestDocsCatalogLintJobsCollectsEntries(t *testing.T) {
	catalog := &ppmodel.CatalogSite{
		ScanRoot: "/repo/apis",
		Services: []*ppmodel.CatalogService{
			{
				Versions: []*ppmodel.CatalogVersion{
					{
						Entries: []*ppmodel.CatalogSpecEntry{
							{RelativePath: "users/openapi.yaml"},
							{RelativePath: ""},
							nil,
						},
					},
					nil,
				},
			},
			nil,
		},
	}

	jobs := docsCatalogLintJobs(catalog)

	require.Len(t, jobs, 1)
	assert.Equal(t, "users/openapi.yaml", jobs[0].relativePath)
	assert.Equal(t, filepath.Join("/repo/apis", "users", "openapi.yaml"), jobs[0].absPath)
}

func TestDocsDiagnosticsLintCatalogConvertsResults(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "apis"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "one.yaml"), []byte(docsDiagnosticsSpec("First API")), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "apis", "two.yaml"), []byte(docsDiagnosticsSpec("Second API")), 0o644))

	rulesetPath := filepath.Join(root, "ruleset.yaml")
	require.NoError(t, os.WriteFile(rulesetPath, []byte(docsDiagnosticsRuleset()), 0o644))

	flags := &LintFlags{
		RulesetFlag:       rulesetPath,
		RemoteFlag:        true,
		TimeoutFlag:       5,
		LookupTimeoutFlag: 500,
		SilentFlag:        true,
		NoStyleFlag:       true,
		PipelineOutput:    true,
	}
	httpClientConfig, err := GetHTTPClientConfig(flags)
	require.NoError(t, err)
	fetchConfig, err := GetFetchConfig(flags)
	require.NoError(t, err)
	diagnostics, err := newDocsDiagnosticsContext(flags, httpClientConfig, fetchConfig, true)
	require.NoError(t, err)

	var progressCalls atomic.Int32
	results, err := diagnostics.lintCatalog(&ppmodel.CatalogSite{
		ScanRoot: root,
		Services: []*ppmodel.CatalogService{
			{
				Versions: []*ppmodel.CatalogVersion{
					{
						Entries: []*ppmodel.CatalogSpecEntry{
							{RelativePath: "one.yaml"},
							{RelativePath: "apis/two.yaml"},
						},
					},
				},
			},
		},
	}, func(completed, total int, currentSpec string, elapsed time.Duration) {
		if completed > 0 {
			progressCalls.Add(1)
		}
	})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, progressCalls.Load(), int32(2))
	require.Len(t, results, 2)
	for _, path := range []string{"one.yaml", "apis/two.yaml"} {
		require.NotEmpty(t, results[path])
		assert.Equal(t, "check-title", results[path][0].RuleId)
		require.NotNil(t, results[path][0].Rule)
		assert.Equal(t, "check-title", results[path][0].Rule.Id)
	}
}

func TestRunDocsSingleGeneratesHTMLAndDiagnostics(t *testing.T) {
	root := t.TempDir()
	specPath := filepath.Join(root, "openapi.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(docsDiagnosticsSpec("First API")), 0o644))
	rulesetPath := filepath.Join(root, "ruleset.yaml")
	require.NoError(t, os.WriteFile(rulesetPath, []byte(docsDiagnosticsRuleset()), 0o644))
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)

	flags := &LintFlags{
		RulesetFlag:       rulesetPath,
		RemoteFlag:        true,
		TimeoutFlag:       5,
		LookupTimeoutFlag: 500,
		SilentFlag:        true,
		NoStyleFlag:       true,
		PipelineOutput:    true,
	}
	httpClientConfig, err := GetHTTPClientConfig(flags)
	require.NoError(t, err)
	fetchConfig, err := GetFetchConfig(flags)
	require.NoError(t, err)
	source := &docsSource{specBytes: specBytes, basePath: root, specPath: specPath}

	// diagnostics enabled -> a diagnostics page is written alongside the docs.
	withDiagnostics, err := newDocsDiagnosticsContext(flags, httpClientConfig, fetchConfig, true)
	require.NoError(t, err)
	outOn := filepath.Join(root, "out-on")
	term := newDocsTerminal(io.Discard, io.Discard, false)
	defer term.finish(nil)

	require.NoError(t, runDocsSingle(source, &docsOptions{outputDir: outOn, noLLM: true, noJSON: true, noLogo: true}, withDiagnostics, term))
	assert.FileExists(t, filepath.Join(outOn, "index.html"))
	assert.FileExists(t, filepath.Join(outOn, "static", "printing-press.css"))
	assert.FileExists(t, filepath.Join(outOn, "diagnostics.html"))

	// diagnostics disabled -> docs render but no diagnostics page.
	withoutDiagnostics, err := newDocsDiagnosticsContext(flags, httpClientConfig, fetchConfig, false)
	require.NoError(t, err)
	outOff := filepath.Join(root, "out-off")
	require.NoError(t, runDocsSingle(source, &docsOptions{outputDir: outOff, noLLM: true, noJSON: true, noLogo: true}, withoutDiagnostics, term))
	assert.FileExists(t, filepath.Join(outOff, "index.html"))
	assert.NoFileExists(t, filepath.Join(outOff, "diagnostics.html"))
}

func docsFingerprintRuleSet(id, message string) *rulesets.RuleSet {
	return &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			id: {
				Id:       id,
				Message:  message,
				Given:    "$.info",
				Severity: model.SeverityWarn,
				Then: map[string]any{
					"field":    "title",
					"function": "truthy",
				},
			},
		},
	}
}

func docsDiagnosticsSpec(title string) string {
	return `openapi: 3.1.0
info:
  title: ` + title + `
  version: 1.0.0
paths: {}
`
}

func docsDiagnosticsRuleset() string {
	return `extends: [[vacuum:oas, off]]
rules:
  check-title:
    description: Check the title value
    severity: warn
    message: title does not match
    given: $.info
    then:
      field: title
      function: pattern
      functionOptions:
        match: this specific thing
`
}
