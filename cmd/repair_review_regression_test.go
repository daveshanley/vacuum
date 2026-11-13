// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestRepairDryRunPreservesBaselineExitAndReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.json")
	original := []byte(`{"openapi":"3.1.0","info":{"title":"Pets","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["cat","cat"]}}}}`)
	require.NoError(t, os.WriteFile(path, original, 0600))
	rules := focusedRepairRules(t, dir, "duplicated-entry-in-enum")
	for _, fixArgs := range [][]string{nil, {"--fix"}, {"--fix", "--dry-run"}} {
		args := append([]string{"lint", path, "--ruleset", rules, "--format", "json"}, fixArgs...)
		report, review, err := exportCommand(t, args...)
		require.Error(t, err)
		var exit *ExitError
		require.ErrorAs(t, err, &exit)
		require.Equal(t, ExitCodeViolations, exit.Code)
		require.Contains(t, report, "duplicated-entry-in-enum")
		if len(fixArgs) == 2 {
			require.Equal(t, 1, strings.Count(review, "--- original"))
			require.NotContains(t, report, "--- original")
		}
		if len(fixArgs) == 1 {
			require.Contains(t, review, "visible terminal review")
			require.NotContains(t, review, "--- original")
		}
		actual, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, original, actual)
	}
}

func TestRepairDeclinesUnsupportedSourceBeforePrompt(t *testing.T) {
	source := []byte(`{"openapi":"3.1.0","info":{"title":"Pets","version":"1"},"paths":{}}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "api.json")
	alias := filepath.Join(dir, "link.json")
	require.NoError(t, os.WriteFile(path, source, 0600))
	require.NoError(t, os.Symlink(path, alias))
	for _, name := range []string{alias, "https://example.test/api.json"} {
		e := &motor.RuleSetExecution{Spec: source, SpecFileName: name, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}}
		var output bytes.Buffer
		result := runReviewedCLIRepairWithIO(e, nil, &LintFlags{FixFlag: true}, nil, errorReader{}, &output, true)
		require.Empty(t, result.Errors)
		require.NotEmpty(t, result.RepairWarnings)
		require.Len(t, result.Results, 1)
		require.Empty(t, output.String())
		result.ReleaseOwnedResources()
	}
}

func TestRepairWriteLeavesNoBackupOrStagedFile(t *testing.T) {
	dir := t.TempDir()
	backupDir := t.TempDir()
	t.Setenv("TMPDIR", backupDir)
	source := filepath.Join(dir, "api.yaml")
	require.NoError(t, os.WriteFile(source, []byte("old"), 0600))
	require.NoError(t, writeRepairFile(source, "", []byte("old"), []byte("new")))
	files, err := os.ReadDir(backupDir)
	require.NoError(t, err)
	require.Empty(t, files)
	files, err = os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestRepairPlanningFailureKeepsTolerantBaseline(t *testing.T) {
	source := []byte(`{"openapi":"3.1.0","info":{"title":"Pets","version":"1"},"paths":{}}`)
	rules := rulesets.GetAllBuiltInRules()
	invalid := &model.Rule{Id: "invalid-schema-dialect", Given: "$.info", Severity: "warn", Then: &model.RuleAction{Function: "schema", FunctionOptions: map[string]any{"dialect": "draft99", "schema": map[string]any{"type": "object"}}}}
	e := &motor.RuleSetExecution{Spec: source, SpecFileName: "api.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rules["info-description"], invalid.Id: invalid}}}
	baseline := motor.ApplyRulesToRuleSet(e.CopyConfig())
	defer baseline.ReleaseOwnedResources()
	var output bytes.Buffer
	result := runReviewedCLIRepairWithIO(e, nil, &LintFlags{FixFlag: true, DryRun: true}, nil, errorReader{}, &output, false)
	defer result.ReleaseOwnedResources()
	require.Len(t, result.Errors, len(baseline.Errors))
	require.Len(t, result.Results, len(baseline.Results))
	require.NotEmpty(t, result.RepairWarnings)
	require.Empty(t, result.ModifiedSpec)
}

func TestRepairRejectsInvalidDestinationBeforePrompt(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "api.yaml")
	require.NoError(t, os.WriteFile(source, []byte("openapi: 3.1.0"), 0600))
	for _, target := range []string{dir, filepath.Join(dir, "missing", "out.yaml")} {
		var output bytes.Buffer
		result := runReviewedCLIRepairWithIO(&motor.RuleSetExecution{SpecFileName: source}, nil, &LintFlags{FixFileFlag: target}, nil, errorReader{}, &output, true)
		require.Len(t, result.Errors, 1)
		require.ErrorContains(t, result.Errors[0], "repair output")
		require.Empty(t, output.String())
	}
}
