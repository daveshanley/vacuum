// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/testify/require"
)

func arazzoCommandFixture(t *testing.T) (string, string) {
	t.Helper()
	path, err := filepath.Abs("../model/test_files/arazzo/workflow.yaml")
	require.NoError(t, err)
	spec, err := os.ReadFile(path)
	require.NoError(t, err)
	return path, string(spec)
}

func TestArazzoLintExitContract(t *testing.T) {
	path, valid := arazzoCommandFixture(t)
	for _, tc := range []struct {
		name, spec string
		code       int
	}{
		{"valid", valid, 0},
		{"invalid target", strings.ReplaceAll(valid, "$sourceDescriptions.api.read", "$sourceDescriptions.api.missing"), 1},
		{"unsupported", strings.Replace(valid, "1.0.1", "2.0.0", 1), 2},
		{"malformed", "arazzo: [\n", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := filepath.Join(t.TempDir(), "workflow.yaml")
			require.NoError(t, os.WriteFile(input, []byte(tc.spec), 0600))
			output, code := runVacuum(t, "lint", input, "--base", filepath.Dir(path), "--no-update-check", "--no-banner", "--no-style")
			require.Equal(t, tc.code, code, "%s", output)
			require.NotContains(t, string(output), "oas3-")
		})
	}
	input := filepath.Join(t.TempDir(), "future.yaml")
	require.NoError(t, os.WriteFile(input, []byte("arazzo: 2.0.0\n"), 0600))
	output, code := runVacuum(t, "lint", input, "--fail-severity", "none", "--no-update-check", "--silent")
	require.Equal(t, 2, code, "%s", output)
}

func TestArazzoMachineReportsAndRulesetOverride(t *testing.T) {
	path, spec := arazzoCommandFixture(t)
	input := filepath.Join(t.TempDir(), "workflow.yaml")
	require.NoError(t, os.WriteFile(input, []byte(strings.ReplaceAll(spec, "$sourceDescriptions.api.read", "$sourceDescriptions.api.missing")), 0600))
	workDir := t.TempDir()
	output, code := runVacuumInDir(t, workDir, "spectral-report", input, "--base", filepath.Dir(path), "--stdout", "--no-update-check")
	require.Equal(t, 0, code, "%s", output)
	var spectral []struct {
		Code   string `json:"code"`
		Path   []any  `json:"path"`
		Source string `json:"source"`
	}
	require.NoError(t, json.Unmarshal(output, &spectral), "%s", output)
	require.Len(t, spectral, 1)
	require.Equal(t, "arazzo-reference", spectral[0].Code)
	require.Equal(t, []any{"workflows", "0", "steps", "0", "operationId"}, spectral[0].Path)
	resolvedSource := spectral[0].Source
	if !filepath.IsAbs(resolvedSource) {
		resolvedSource = filepath.Join(workDir, resolvedSource)
	}
	require.Equal(t, input, resolvedSource)
	output, code = runVacuum(t, "report", input, "--base", filepath.Dir(path), "--stdout", "--no-update-check")
	require.Equal(t, 0, code, "%s", output)
	var report vacuum_report.VacuumReport
	require.NoError(t, json.Unmarshal(output, &report), "%s", output)
	require.Equal(t, "arazzo", report.SpecInfo.SpecType)
	require.Len(t, report.ResultSet.Results, 1)
	require.Equal(t, "$.workflows[0].steps[0].operationId", report.ResultSet.Results[0].Path)
	ruleFile := filepath.Join(t.TempDir(), "rules.yaml")
	require.NoError(t, os.WriteFile(ruleFile, []byte("extends: [spectral:arazzo]\nrules:\n  arazzo-reference: off\n"), 0600))
	output, code = runVacuum(t, "lint", input, "--base", filepath.Dir(path), "--ruleset", ruleFile, "--no-update-check", "--silent")
	require.Equal(t, 0, code, "%s", output)
	require.NoError(t, os.WriteFile(ruleFile, []byte("extends: [[spectral:arazzo, off]]\nrules:\n  arazzo-reference: true\n"), 0600))
	output, code = runVacuum(t, "lint", input, "--base", filepath.Dir(path), "--ruleset", ruleFile, "--no-update-check", "--silent")
	require.Equal(t, 1, code, "%s", output)
}

func TestArazzoStdinUsesBase(t *testing.T) {
	path, spec := arazzoCommandFixture(t)
	dir := t.TempDir()
	process := exec.Command(os.Args[0], "-test.run=^TestVacuumSubprocess$", "--", "spectral-report", "--stdin", "--stdout", "--base", filepath.Dir(path), "--no-update-check")
	process.Dir = dir
	process.Env = append(os.Environ(), "VACUUM_TEST_SUBPROCESS=1", "XDG_CONFIG_HOME="+dir)
	process.Stdin = strings.NewReader(spec)
	output, err := process.CombinedOutput()
	require.NoError(t, err, "%s", output)
	var results []any
	require.NoError(t, json.Unmarshal(output, &results))
	require.Empty(t, results)
}

func TestArazzoDefaultsAndGeneratedRuleset(t *testing.T) {
	_, spec := arazzoCommandFixture(t)
	for _, hard := range []bool{false, true} {
		rs, format, _ := prepareDefaultRuleSetForSpec(rulesets.BuildDefaultRuleSets(), []byte(spec), hard, false)
		require.Equal(t, model.Arazzo10, format)
		require.Contains(t, rs.Rules, "arazzo-structure")
		require.NotContains(t, rs.Rules, rulesets.OwaspArrayLimit)
		_, advisory := rs.Rules["arazzo-advisory"]
		require.Equal(t, hard, advisory)
	}
	for _, name := range []string{"arazzo-recommended", "arazzo-all"} {
		prefix := filepath.Join(t.TempDir(), "rules")
		output, code := runVacuum(t, "generate-ruleset", name, prefix, "--no-update-check")
		require.Equal(t, 0, code, "%s", output)
		data, err := os.ReadFile(prefix + "-" + name + ".yaml")
		require.NoError(t, err)
		rs, err := rulesets.CreateRuleSetFromData(data)
		require.NoError(t, err)
		loaded := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(rs)
		require.NoError(t, loaded.LoadError())
		require.Contains(t, loaded.Rules, "arazzo-structure")
	}
}

func TestArazzoOriginalKeepsNewSourceErrors(t *testing.T) {
	path, spec := arazzoCommandFixture(t)
	api, err := os.ReadFile(filepath.Join(filepath.Dir(path), "api.yaml"))
	require.NoError(t, err)
	oldDir, newDir := t.TempDir(), t.TempDir()
	for _, dir := range []string{oldDir, newDir} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(spec), 0600))
		source := string(api)
		if dir == newDir {
			source = strings.Replace(source, "operationId: read", "operationId: gone", 1)
		}
		require.NoError(t, os.WriteFile(filepath.Join(dir, "api.yaml"), []byte(source), 0600))
	}
	output, code := runVacuum(t, "lint", filepath.Join(newDir, "workflow.yaml"), "--original", filepath.Join(oldDir, "workflow.yaml"), "--details", "--no-update-check", "--no-banner", "--no-style")
	require.Equal(t, 1, code, "%s", output)
	require.Contains(t, string(output), "arazzo-reference")
	require.NotContains(t, string(output), "Failed to load changes")
	require.NotContains(t, string(output), "Failed to generate change report")
}

func TestArazzoAnnotationDoesNotOverrideSchemaCommand(t *testing.T) {
	for _, marker := range []string{"", "arazzo: 1.0.1\n"} {
		input := filepath.Join(t.TempDir(), "schema.yaml")
		require.NoError(t, os.WriteFile(input, []byte("$schema: https://json-schema.org/draft/2020-12/schema\ntype: 42\n"+marker), 0600))
		output, code := runVacuum(t, "schema", input, "--details", "--no-update-check", "--no-banner", "--no-style")
		require.Equal(t, 1, code, "%s", output)
		require.Contains(t, string(output), "json-schema-valid")
	}
}

func arazzoExternalSourceFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	root := `arazzo: 1.1.0
info: {title: Root, version: '1', description: Root workflow., summary: Root.}
sourceDescriptions: [{name: other, url: other.yaml, type: arazzo}]
workflows:
  - workflowId: run
    description: This is the ROOT description.
    summary: Run.
    steps: [{stepId: call, description: Call., workflowId: $sourceDescriptions.other.remote}]
`
	source := `arazzo: 1.1.0
info: {title: Other, version: '1'}
sourceDescriptions: []
workflows:
  - workflowId: remote
    steps: [{stepId: call, workflowId: absent, successCriteria: [{condition: "$steps.missing.outputs.value == 2"}]}]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.yaml"), []byte(root), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.yaml"), []byte(source), 0600))
	return filepath.Join(dir, "main.yaml")
}

func TestArazzoExternalSourceSnippetsAndReportReplay(t *testing.T) {
	input := arazzoExternalSourceFixture(t)
	output, code := runVacuum(t, "lint", input, "--details", "--snippets", "--no-update-check", "--no-banner", "--no-style")
	require.Equal(t, 1, code, "%s", output)
	require.Contains(t, string(output), "other.yaml:6")
	require.Contains(t, string(output), "$steps.missing.outputs.value")
	require.NotContains(t, string(output), "This is the ROOT description.")
	output, code = runVacuum(t, "report", input, "--stdout", "--no-update-check")
	require.Equal(t, 0, code, "%s", output)
	reportPath := filepath.Join(t.TempDir(), "report.json")
	require.NoError(t, os.WriteFile(reportPath, output, 0600))
	// A saved report must retain the authored snippet even after the source goes away.
	require.NoError(t, os.Remove(filepath.Join(filepath.Dir(input), "other.yaml")))
	output, code = runVacuum(t, "lint", reportPath, "--details", "--snippets", "--no-update-check", "--no-banner", "--no-style")
	require.Equal(t, 1, code, "%s", output)
	require.Contains(t, string(output), "$steps.missing.outputs.value")
	require.NotContains(t, string(output), "This is the ROOT description.")
}

func TestArazzoOriginalDoesNotMatchErrorsInDifferentSources(t *testing.T) {
	oldPath := arazzoExternalSourceFixture(t)
	newPath := arazzoExternalSourceFixture(t)
	source, err := os.ReadFile(filepath.Join(filepath.Dir(newPath), "other.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(newPath), "new-source.yaml"), source, 0600))
	root, err := os.ReadFile(newPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(newPath, []byte(strings.ReplaceAll(string(root), "url: other.yaml", "url: new-source.yaml")), 0600))
	output, code := runVacuum(t, "lint", newPath, "--original", oldPath, "--details", "--no-update-check", "--no-banner", "--no-style")
	require.Equal(t, 1, code, "%s", output)
	require.Contains(t, string(output), "new-source.yaml:6")
	// The same source-local finding is still suppressed across mirrored directories.
	require.NoError(t, os.WriteFile(newPath, root, 0600))
	output, code = runVacuum(t, "lint", newPath, "--original", oldPath, "--no-update-check", "--no-banner", "--no-style")
	require.Equal(t, 0, code, "%s", output)
}
