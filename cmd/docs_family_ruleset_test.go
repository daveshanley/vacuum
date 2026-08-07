// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	press "github.com/pb33f/doctor/printingpress"
	ppconfig "github.com/pb33f/doctor/printingpress/config"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestDocsFamilyRulesetFlags(t *testing.T) {
	cmd := GetDocsCommand()

	for _, name := range []string{"openapi-ruleset", "asyncapi-ruleset"} {
		flag := cmd.Flags().Lookup(name)
		require.NotNil(t, flag)
		assert.Contains(t, flag.Usage, "diagnostics")
	}
}

func TestDocsNoDiagnosticsDoesNotLoadConfiguredFamilyRulesets(t *testing.T) {
	for _, input := range []string{"single", "aggregate"} {
		t.Run(input, func(t *testing.T) {
			root := t.TempDir()
			specPath := filepath.Join(root, "openapi.yaml")
			writeTestFile(t, specPath, docsDiagnosticsSpec("Disabled Diagnostics"))
			docsInput := specPath
			if input == "aggregate" {
				docsInput = root
			}

			cmd := GetDocsCommand()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.Flags().String("ruleset", "", "")
			cmd.SetArgs([]string{
				docsInput,
				"--output", filepath.Join(t.TempDir(), "docs"),
				"--no-diagnostics",
				"--no-llm",
				"--no-json",
				"--no-logo",
				"--openapi-ruleset", filepath.Join(root, "missing-openapi.yaml"),
				"--asyncapi-ruleset", filepath.Join(root, "missing-asyncapi.yaml"),
				"--ruleset", filepath.Join(root, "missing-legacy.yaml"),
			})

			require.NoError(t, cmd.Execute())
		})
	}
}

func TestRunDocsSingleHonorsFamilyRulesets(t *testing.T) {
	tests := []struct {
		name          string
		specName      string
		spec          string
		configureOpts func(*docsOptions, string)
		writeRuleset  func(*testing.T, string, string, string) string
		ruleID        string
	}{
		{
			name:     "OpenAPI",
			specName: "openapi.yaml",
			spec:     docsDiagnosticsSpec("Single HTTP API"),
			configureOpts: func(opts *docsOptions, path string) {
				opts.openAPIRuleset = path
			},
			writeRuleset: writeDocsNamedRuleset,
			ruleID:       "single-openapi-family-rule",
		},
		{
			name:     "AsyncAPI",
			specName: "asyncapi.yaml",
			spec:     cmdAsyncAPI31Fixture,
			configureOpts: func(opts *docsOptions, path string) {
				opts.asyncAPIRuleset = path
			},
			writeRuleset: writeDocsNamedAsyncAPIRuleset,
			ruleID:       "single-asyncapi-family-rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			specPath := filepath.Join(root, tt.specName)
			writeTestFile(t, specPath, tt.spec)
			rulesetPath := tt.writeRuleset(t, root, "family-ruleset.yaml", tt.ruleID)
			opts := &docsOptions{
				outputDir: filepath.Join(t.TempDir(), "docs"),
				noLLM:     true,
				noJSON:    true,
				noLogo:    true,
			}
			tt.configureOpts(opts, rulesetPath)
			cmd := GetDocsCommand()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			require.NoError(t, runDocs(cmd, specPath, opts))
			assert.Contains(t, readTestFile(t, filepath.Join(opts.outputDir, "data", "pages", "diagnostics.js")), tt.ruleID)
		})
	}
}

func TestDocsDiagnosticsFamilyRulesetPrecedence(t *testing.T) {
	root := t.TempDir()
	legacyPath := writeDocsNamedRuleset(t, root, "legacy.yaml", "legacy-rule")
	openAPIPath := writeDocsNamedRuleset(t, root, "openapi.yaml", "openapi-rule")
	asyncAPIPath := writeDocsNamedRuleset(t, root, "asyncapi.yaml", "asyncapi-rule")
	flags := docsTestLintFlags(legacyPath)
	original := *flags

	ctx := newDocsTestDiagnosticsContext(t, flags, docsFamilyRulesetPaths{
		openAPI:  openAPIPath,
		asyncAPI: asyncAPIPath,
	})

	tests := []struct {
		name     string
		spec     string
		expected string
	}{
		{name: "OpenAPI family override", spec: docsDiagnosticsSpec("HTTP API"), expected: "openapi-rule"},
		{name: "AsyncAPI family override", spec: cmdAsyncAPI31Fixture, expected: "asyncapi-rule"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleSet, err := ctx.ruleSetForSpec([]byte(tt.spec))
			require.NoError(t, err)
			assert.Contains(t, ruleSet.Rules, tt.expected)
			assert.NotContains(t, ruleSet.Rules, "legacy-rule")
		})
	}

	assert.Equal(t, original, *flags, "loading docs rulesets must not mutate caller lint flags")
}

func TestDocsDiagnosticsPartialFamilyOverridesFallBack(t *testing.T) {
	tests := []struct {
		name             string
		legacy           bool
		configuredFamily string
		spec             string
		expectedRule     string
		unexpectedRule   string
	}{
		{name: "OpenAPI override and AsyncAPI legacy fallback", legacy: true, configuredFamily: "openapi", spec: cmdAsyncAPI31Fixture, expectedRule: "legacy-rule", unexpectedRule: "openapi-rule"},
		{name: "AsyncAPI override and OpenAPI legacy fallback", legacy: true, configuredFamily: "asyncapi", spec: docsDiagnosticsSpec("HTTP API"), expectedRule: "legacy-rule", unexpectedRule: "asyncapi-rule"},
		{name: "OpenAPI override and AsyncAPI built-in fallback", configuredFamily: "openapi", spec: cmdAsyncAPI31Fixture, expectedRule: rulesets.AsyncAPI3DocumentResolved, unexpectedRule: "openapi-rule"},
		{name: "AsyncAPI override and OpenAPI built-in fallback", configuredFamily: "asyncapi", spec: docsDiagnosticsSpec("HTTP API"), expectedRule: rulesets.OperationSuccessResponse, unexpectedRule: "asyncapi-rule"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			legacyPath := ""
			if tt.legacy {
				legacyPath = writeDocsNamedRuleset(t, root, "legacy.yaml", "legacy-rule")
			}
			paths := docsFamilyRulesetPaths{}
			switch tt.configuredFamily {
			case "openapi":
				paths.openAPI = writeDocsNamedRuleset(t, root, "openapi.yaml", "openapi-rule")
			case "asyncapi":
				paths.asyncAPI = writeDocsNamedRuleset(t, root, "asyncapi.yaml", "asyncapi-rule")
			}
			ctx := newDocsTestDiagnosticsContext(t, docsTestLintFlags(legacyPath), paths)

			ruleSet, err := ctx.ruleSetForSpec([]byte(tt.spec))
			require.NoError(t, err)
			assert.Contains(t, ruleSet.Rules, tt.expectedRule)
			assert.NotContains(t, ruleSet.Rules, tt.unexpectedRule)
		})
	}
}

func TestDocsDiagnosticsConfiguredRulesetsLoadOnce(t *testing.T) {
	root := t.TempDir()
	openAPIPath := writeDocsNamedRuleset(t, root, "openapi.yaml", "openapi-rule")
	asyncAPIPath := writeDocsNamedRuleset(t, root, "asyncapi.yaml", "asyncapi-rule")
	ctx := newDocsTestDiagnosticsContext(t, docsTestLintFlags(""), docsFamilyRulesetPaths{
		openAPI:  openAPIPath,
		asyncAPI: asyncAPIPath,
	})
	require.NoError(t, os.Remove(openAPIPath))
	require.NoError(t, os.Remove(asyncAPIPath))

	for i := 0; i < 2; i++ {
		openAPIRules, err := ctx.ruleSetForSpec([]byte(docsDiagnosticsSpec("HTTP API")))
		require.NoError(t, err)
		assert.Contains(t, openAPIRules.Rules, "openapi-rule")

		asyncAPIRules, err := ctx.ruleSetForSpec([]byte(cmdAsyncAPI31Fixture))
		require.NoError(t, err)
		assert.Contains(t, asyncAPIRules.Rules, "asyncapi-rule")
	}
}

func TestDocsDiagnosticsFamilyRulesetLoadErrorsIdentifyFamily(t *testing.T) {
	for _, family := range []string{"OpenAPI", "AsyncAPI"} {
		t.Run(family, func(t *testing.T) {
			paths := docsFamilyRulesetPaths{}
			if family == "OpenAPI" {
				paths.openAPI = filepath.Join(t.TempDir(), "missing.yaml")
			} else {
				paths.asyncAPI = filepath.Join(t.TempDir(), "missing.yaml")
			}

			_, err := newDocsDiagnosticsContext(docsTestLintFlags(""), utils.HTTPClientConfig{}, nil, true, paths)

			require.Error(t, err)
			assert.ErrorContains(t, err, family+" diagnostics ruleset")
		})
	}
}

func TestDocsDiagnosticsRejectsUnknownAndUnsupportedSpecFamilies(t *testing.T) {
	ctx := newDocsTestDiagnosticsContext(t, docsTestLintFlags(""), docsFamilyRulesetPaths{})
	for _, tt := range []struct {
		name string
		spec string
	}{
		{name: "unknown", spec: "info:\n  title: not a contract\n"},
		{name: "malformed", spec: `{"info": `},
		{name: "unsupported AsyncAPI", spec: "asyncapi: 2.6.0\ninfo:\n  title: legacy\n  version: 1.0.0\nchannels: {}\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ctx.ruleSetForSpec([]byte(tt.spec))
			require.Error(t, err)
			assert.ErrorContains(t, err, "detect diagnostics spec family")
		})
	}
}

func TestDocsDiagnosticsFingerprintIncludesAllRulesetIdentities(t *testing.T) {
	root := t.TempDir()
	legacyPath := writeDocsNamedRuleset(t, root, "legacy.yaml", "legacy-rule")
	openAPIPath := writeDocsNamedRuleset(t, root, "openapi.yaml", "openapi-rule")
	asyncAPIPath := writeDocsNamedRuleset(t, root, "asyncapi.yaml", "asyncapi-rule")
	flags := docsTestLintFlags(legacyPath)
	ctx := newDocsTestDiagnosticsContext(t, flags, docsFamilyRulesetPaths{openAPI: openAPIPath, asyncAPI: asyncAPIPath})
	baseline := ctx.fingerprint

	for i := 0; i < 20; i++ {
		assert.Equal(t, baseline, docsDiagnosticsFingerprint(true, flags, ctx.legacyRuleset, docsFamilyRulesets{
			openAPIPath: openAPIPath, openAPI: ctx.openAPIRuleset,
			asyncAPIPath: asyncAPIPath, asyncAPI: ctx.asyncAPIRuleset,
		}))
	}

	for _, tc := range []struct {
		name string
		path *string
		rule string
	}{
		{name: "legacy content", path: &legacyPath, rule: "legacy-changed"},
		{name: "OpenAPI content", path: &openAPIPath, rule: "openapi-changed"},
		{name: "AsyncAPI content", path: &asyncAPIPath, rule: "asyncapi-changed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeTestFile(t, legacyPath, docsNamedRuleset("legacy-rule"))
			writeTestFile(t, openAPIPath, docsNamedRuleset("openapi-rule"))
			writeTestFile(t, asyncAPIPath, docsNamedRuleset("asyncapi-rule"))
			writeTestFile(t, *tc.path, docsNamedRuleset(tc.rule))
			changed := newDocsTestDiagnosticsContext(t, docsTestLintFlags(legacyPath), docsFamilyRulesetPaths{openAPI: openAPIPath, asyncAPI: asyncAPIPath})
			assert.NotEqual(t, baseline, changed.fingerprint)
		})
	}

	writeTestFile(t, legacyPath, docsNamedRuleset("legacy-rule"))
	writeTestFile(t, openAPIPath, docsNamedRuleset("openapi-rule"))
	writeTestFile(t, asyncAPIPath, docsNamedRuleset("asyncapi-rule"))
	for _, family := range []string{"legacy", "OpenAPI", "AsyncAPI"} {
		t.Run(family+" path", func(t *testing.T) {
			otherRoot := t.TempDir()
			otherLegacyPath := legacyPath
			otherOpenAPIPath := openAPIPath
			otherAsyncAPIPath := asyncAPIPath
			switch family {
			case "legacy":
				otherLegacyPath = writeDocsNamedRuleset(t, otherRoot, "legacy.yaml", "legacy-rule")
			case "OpenAPI":
				otherOpenAPIPath = writeDocsNamedRuleset(t, otherRoot, "openapi.yaml", "openapi-rule")
			case "AsyncAPI":
				otherAsyncAPIPath = writeDocsNamedRuleset(t, otherRoot, "asyncapi.yaml", "asyncapi-rule")
			}
			pathChanged := newDocsTestDiagnosticsContext(t, docsTestLintFlags(otherLegacyPath), docsFamilyRulesetPaths{openAPI: otherOpenAPIPath, asyncAPI: otherAsyncAPIPath})
			assert.NotEqual(t, baseline, pathChanged.fingerprint)
		})
	}
}

func TestDocsDiagnosticsFingerprintIsStableAcrossRuleMapOrder(t *testing.T) {
	first := docsFingerprintRuleSet("alpha", "first")
	first.Rules["beta"] = docsFingerprintRuleSet("beta", "second").Rules["beta"]
	second := docsFingerprintRuleSet("beta", "second")
	second.Rules["alpha"] = docsFingerprintRuleSet("alpha", "first").Rules["alpha"]
	flags := docsTestLintFlags("")

	assert.Equal(t,
		docsDiagnosticsFingerprint(true, flags, nil, docsFamilyRulesets{openAPI: first}),
		docsDiagnosticsFingerprint(true, flags, nil, docsFamilyRulesets{openAPI: second}),
	)
}

func TestBuildDocsAggregateConfigPassesThroughGroupingWithoutAliasing(t *testing.T) {
	fileConfig := &ppconfig.File{Grouping: ppconfig.GroupingConfig{
		ServiceIdentity: ppconfig.ServiceIdentityConfig{
			MetadataPointers:           []string{"/info/x-owner/service"},
			StripPrefixes:              []string{"platform-"},
			StripSuffixes:              []string{"-api"},
			PreferOpenAPISlug:          true,
			MetadataOptionalForOpenAPI: true,
		},
		ContractRoles: []ppconfig.ContractRoleRule{
			{Pattern: "**/openapi.yaml", Role: "http-api", ContractID: "http", Default: true},
			{Pattern: "**/asyncapi.yaml", Role: "published-event", ContractID: "events", Default: false},
		},
	}}

	cfg := buildDocsAggregateConfig("/apis", "/output", "hosted", &docsOptions{}, fileConfig)

	assert.Equal(t, press.AggregateServiceIdentityConfig{
		MetadataPointers:           []string{"/info/x-owner/service"},
		StripPrefixes:              []string{"platform-"},
		StripSuffixes:              []string{"-api"},
		PreferOpenAPISlug:          true,
		MetadataOptionalForOpenAPI: true,
	}, cfg.ServiceIdentity)
	assert.Equal(t, []press.AggregateContractRoleRule{
		{Pattern: "**/openapi.yaml", Role: "http-api", ContractID: "http", Default: true},
		{Pattern: "**/asyncapi.yaml", Role: "published-event", ContractID: "events", Default: false},
	}, cfg.ContractRoles)

	cfg.ServiceIdentity.MetadataPointers[0] = "/changed"
	cfg.ServiceIdentity.StripPrefixes[0] = "changed-"
	cfg.ServiceIdentity.StripSuffixes[0] = "-changed"
	cfg.ServiceIdentity.MetadataOptionalForOpenAPI = false
	cfg.ContractRoles[0].Pattern = "changed"
	assert.Equal(t, "/info/x-owner/service", fileConfig.Grouping.ServiceIdentity.MetadataPointers[0])
	assert.Equal(t, "platform-", fileConfig.Grouping.ServiceIdentity.StripPrefixes[0])
	assert.Equal(t, "-api", fileConfig.Grouping.ServiceIdentity.StripSuffixes[0])
	assert.True(t, fileConfig.Grouping.ServiceIdentity.MetadataOptionalForOpenAPI)
	assert.Equal(t, "**/openapi.yaml", fileConfig.Grouping.ContractRoles[0].Pattern)
}

func newDocsTestDiagnosticsContext(t *testing.T, flags *LintFlags, paths docsFamilyRulesetPaths) *docsDiagnosticsContext {
	t.Helper()
	ctx, err := newDocsDiagnosticsContext(flags, utils.HTTPClientConfig{}, nil, true, paths)
	require.NoError(t, err)
	return ctx
}

func docsTestLintFlags(legacyPath string) *LintFlags {
	return &LintFlags{
		RulesetFlag:       legacyPath,
		RemoteFlag:        true,
		TimeoutFlag:       5,
		LookupTimeoutFlag: 500,
		SilentFlag:        true,
		NoStyleFlag:       true,
		PipelineOutput:    true,
	}
}

func writeDocsNamedRuleset(t *testing.T, root, name, ruleID string) string {
	t.Helper()
	path := filepath.Join(root, name)
	writeTestFile(t, path, docsNamedRuleset(ruleID))
	return path
}

func writeDocsNamedAsyncAPIRuleset(t *testing.T, root, name, ruleID string) string {
	t.Helper()
	path := filepath.Join(root, name)
	writeTestFile(t, path, `extends: [[vacuum:asyncapi, off]]
rules:
  `+ruleID+`:
    description: Family marker
    severity: warn
    formats: [asyncapi3]
    given: $.info
    then:
      field: title
      function: pattern
      functionOptions:
        match: this-title-never-matches
`)
	return path
}

func docsNamedRuleset(ruleID string) string {
	return `extends: [[vacuum:oas, off]]
rules:
  ` + ruleID + `:
    description: Family marker
    severity: warn
    given: $.info
    then:
      field: title
      function: pattern
      functionOptions:
        match: this-title-never-matches
`
}
