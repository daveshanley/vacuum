package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/html-report"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
	"github.com/spf13/cobra"
)

func TestIssue955OriginalExecutionFailures(t *testing.T) {
	spec := []byte("openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}\n")
	original := filepath.Join(t.TempDir(), "original.yaml")
	require.NoError(t, os.WriteFile(original, spec, 0o600))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, failure := range []struct {
		name, given, function string
		options               *motor.ExecutionOptions
	}{
		{name: "lookup", given: "$..[", function: "truthy"},
		{name: "unknown function", given: "$", function: "missingFunction"},
		{name: "canceled", given: "$", function: "truthy", options: &motor.ExecutionOptions{Context: canceled}},
	} {
		t.Run(failure.name, func(t *testing.T) {
			execution := &motor.RuleSetExecution{RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"test": {Id: "test", Given: failure.given, Then: model.RuleAction{Function: failure.function}}}}, CustomFunctions: map[string]model.RuleFunction{"unused": originalShortcutTestFunction{}}}
			warned := false
			warning := func(error) { warned = true }
			current := []model.RuleFunctionResult{{Message: "current finding", RuleId: "test"}}
			_, _, err := applyOriginalDiffToValues(originalValueDiffOptions{OriginalPath: original, CurrentPath: original, Results: current, Execution: execution, ExecutionOptions: failure.options, WarnOriginalLintFailure: warning})
			requireOriginalInputError(t, err)
			require.Contains(t, err.Error(), "original spec")
			resultSet := model.NewRuleResultSet(current)
			_, filtered, err := applyOriginalDiffToResultSet(originalResultSetDiffOptions{OriginalPath: original, CurrentPath: original, CurrentBytes: spec, ResultSet: resultSet, Execution: execution, ExecutionOptions: failure.options, WarnOriginalLintFailure: warning})
			requireOriginalInputError(t, err)
			require.False(t, filtered)
			require.Len(t, resultSet.Results, 1, "failed baseline must not be used for comparison")
			require.False(t, warned, "execution failures must be returned, not downgraded to warnings")
		})
	}
}

func requireOriginalInputError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var exit *ExitError
	require.True(t, errors.As(err, &exit))
	require.Equal(t, ExitCodeInputError, exit.Code)
}

func TestIssue955OriginalParseFailureKeepsFallback(t *testing.T) {
	original := filepath.Join(t.TempDir(), "original.yaml")
	require.NoError(t, os.WriteFile(original, []byte("openapi: ["), 0o600))
	results, err := LintOriginalSpec(original, &motor.RuleSetExecution{RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{}}}, nil)
	require.NoError(t, err)
	require.Nil(t, results)
}

func TestIssue955OriginalCommandsRejectIncompleteLint(t *testing.T) {
	const openapi = "openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}\n"
	const swagger = "swagger: '2.0'\ninfo: {title: Test, version: '1'}\npaths: {}\n"
	for _, phase := range []string{"current", "original"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			currentPath := filepath.Join(dir, "current.yaml")
			originalPath := filepath.Join(dir, "original.yaml")
			rulesPath := filepath.Join(dir, "rules.yaml")
			current := openapi
			// Only the original matches the rule's format in this case. Its
			// invalid selector fails after the current lint has completed.
			if phase == "original" {
				current = swagger
			}
			require.NoError(t, os.WriteFile(currentPath, []byte(current), 0o600))
			require.NoError(t, os.WriteFile(originalPath, []byte(openapi), 0o600))
			require.NoError(t, os.WriteFile(rulesPath, []byte("rules:\n  invalid-selector:\n    formats: [oas3]\n    given: '$..['\n    severity: error\n    then:\n      function: truthy\n"), 0o600))
			for _, surface := range []struct {
				name    string
				command func() *cobra.Command
				output  string
			}{
				{"lint", GetLintCommand, ""}, {"dashboard", GetDashboardCommand, ""},
				{"report", GetVacuumReportCommand, "report"},
				{"spectral-report", GetSpectralReportCommand, "spectral.json"},
				{"html-report", GetHTMLReportCommand, "report.html"},
			} {
				t.Run(surface.name, func(t *testing.T) {
					if surface.name == "html-report" && !html_report.AssetsAvailable() {
						t.Skip("requires html_report_ui build")
					}
					command := surface.command()
					registerPersistentFlags(command)
					var output bytes.Buffer
					command.SetOut(&output)
					command.SetErr(&output)
					args := []string{"--original", originalPath, "-r", rulesPath, currentPath}
					outputPath := filepath.Join(dir, surface.output)
					if surface.output != "" {
						args = append(args, outputPath)
					}
					command.SetArgs(args)
					err := command.Execute()
					requireOriginalInputError(t, err)
					if phase == "original" {
						require.Contains(t, err.Error(), "original spec")
					}
					if surface.output != "" {
						files, err := filepath.Glob(outputPath + "*")
						require.NoError(t, err)
						require.Empty(t, files, "must not write a successful comparison report")
					}
				})
			}
		})
	}
}
