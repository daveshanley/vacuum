// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestLint_Issue946SchemaPropertyWildcardIgnore(t *testing.T) {
	specPath, rulesetPath, ignorePath := writeIssue946IgnoreFixtures(t)

	baselineCmd := GetLintCommand()
	registerPersistentFlags(baselineCmd)
	baselineOutput := bytes.NewBuffer(nil)
	baselineCmd.SetOut(baselineOutput)
	baselineCmd.SetErr(baselineOutput)
	baselineCmd.SetArgs([]string{
		"--no-banner",
		"--no-style",
		"--ruleset", rulesetPath,
		specPath,
	})
	var baselineErr error
	captureOSStreams(t, func() {
		baselineErr = baselineCmd.Execute()
	})
	require.Error(t, baselineErr)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{
		"--no-banner",
		"--no-style",
		"--silent",
		"--ignore-file", ignorePath,
		"--ruleset", rulesetPath,
		specPath,
	})

	require.NoError(t, cmd.Execute())
}

func TestVacuumReport_Issue946SchemaPropertyWildcardIgnore(t *testing.T) {
	specPath, rulesetPath, ignorePath := writeIssue946IgnoreFixtures(t)
	baselineReport := executeIssue946VacuumReport(t, specPath, rulesetPath, "")
	require.NotNil(t, baselineReport.ResultSet)
	require.Len(t, baselineReport.ResultSet.Results, 2)

	ignoredReport := executeIssue946VacuumReport(t, specPath, rulesetPath, ignorePath)
	require.NotNil(t, ignoredReport.ResultSet)
	assert.Empty(t, ignoredReport.ResultSet.Results)
}

func executeIssue946VacuumReport(
	t *testing.T,
	specPath string,
	rulesetPath string,
	ignorePath string,
) *vacuum_report.VacuumReport {
	t.Helper()

	reportPrefix := filepath.Join(t.TempDir(), "issue-946-report")

	cmd := GetVacuumReportCommand()
	registerPersistentFlags(cmd)
	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)
	args := []string{
		"--no-style",
		"--no-pretty",
		"--ruleset", rulesetPath,
		specPath,
		reportPrefix,
	}
	if ignorePath != "" {
		args = append(args, "--ignore-file", ignorePath)
	}
	cmd.SetArgs(args)

	require.NoError(t, cmd.Execute())

	reportPath := requireSingleGeneratedFile(t, reportPrefix+"-*.json")
	reportBytes, err := os.ReadFile(reportPath)
	require.NoError(t, err)

	var report vacuum_report.VacuumReport
	require.NoError(t, json.Unmarshal(reportBytes, &report))
	return &report
}

func TestSpectralReport_Issue946SchemaPropertyWildcardIgnore(t *testing.T) {
	specPath, rulesetPath, ignorePath := writeIssue946IgnoreFixtures(t)
	reportPath := filepath.Join(t.TempDir(), "issue-946-spectral-report.json")

	cmd := GetSpectralReportCommand()
	registerPersistentFlags(cmd)
	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{
		"--no-style",
		"--no-pretty",
		"--ignore-file", ignorePath,
		"--ruleset", rulesetPath,
		specPath,
		reportPath,
	})

	require.NoError(t, cmd.Execute())

	reportBytes, err := os.ReadFile(reportPath)
	require.NoError(t, err)

	var results []json.RawMessage
	require.NoError(t, json.Unmarshal(reportBytes, &results))
	assert.Empty(t, results)
}

func writeIssue946IgnoreFixtures(t *testing.T) (string, string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "openapi.yaml")
	rulesetPath := filepath.Join(tmpDir, "ruleset.yaml")
	ignorePath := filepath.Join(tmpDir, "ignore.yaml")

	require.NoError(t, os.WriteFile(specPath, []byte(`openapi: 3.1.0
info:
  title: Issue 946
  version: 1.0.0
paths: {}
components:
  schemas:
    User:
      type: object
      properties:
        bad_name:
          type: string
        also_bad:
          type: string
`), 0o600))
	require.NoError(t, os.WriteFile(rulesetPath, []byte(`extends: [[vacuum:oas, off]]
rules:
  camel-case-properties:
    description: Schema properties must use camelCase
    severity: error
    given: $
    then:
      function: oasCamelCaseProperties
`), 0o600))
	require.NoError(t, os.WriteFile(ignorePath, []byte(`camel-case-properties:
  - $.components.schemas[*].properties[*]
`), 0o600))

	return specPath, rulesetPath, ignorePath
}
