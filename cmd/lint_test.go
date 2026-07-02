package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestGetLintCommand(t *testing.T) {
	cmd := GetLintCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "lint <your-api-file.yaml>", cmd.Use)
	assert.Contains(t, cmd.Short, "Lint an OpenAPI or AsyncAPI")
}

func TestGetLintCommand_NoSpec(t *testing.T) {
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.Error(t, err)

	// error message is printed to stderr, not stdout
	// the actual error is returned
	assert.Contains(t, err.Error(), "no file supplied")
}

func TestGetLintCommand_MissingSpec(t *testing.T) {
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"does-not-exist.yaml"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestGetLintCommand_WithRuleset(t *testing.T) {
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../rulesets/examples/custom-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
		"-d",
	})

	err := cmd.Execute()
	assert.Error(t, err) // this should fail, will not match title.
}

func TestGetLintCommand_BadRuleset(t *testing.T) {
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"-r",
		"../rulesets/examples/nope.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.NotNil(t, err)
}

func TestGetLintCommand_WithDetails(t *testing.T) {
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-d",
		"../model/test_files/burgershop.openapi.yaml",
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestGetLintCommand_WithSnippets(t *testing.T) {
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"--snippets",
		"-d",
		"../model/test_files/burgershop.openapi.yaml",
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestGetLintCommand_BadSpec(t *testing.T) {
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"../model/test_files/badspec.yaml",
	})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestGetLintCommand_WithVacuumReport(t *testing.T) {
	// test with pre-compiled vacuum report
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/burgershop-report.json.gz",
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestResolveLintCategoryFlagIsCaseInsensitive(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantID string
	}{
		{name: "lowercase id", input: "tags", wantID: model.CategoryTags},
		{name: "display name", input: "Tags", wantID: model.CategoryTags},
		{name: "uppercase id", input: "TAGS", wantID: model.CategoryTags},
		{name: "mixed case display name", input: "ScHeMaS", wantID: model.CategorySchemas},
		{name: "multi word display name", input: "contract information", wantID: model.CategoryInfo},
		{name: "owasp lowercase", input: "owasp", wantID: model.CategoryOWASP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categories, ok := resolveLintCategoryFlag(tt.input)

			require.True(t, ok)
			require.Len(t, categories, 1)
			assert.Equal(t, tt.wantID, categories[0].Id)
		})
	}
}

func TestResolveLintCategoryFlagUnknownFallsBackToAllCategories(t *testing.T) {
	categories, ok := resolveLintCategoryFlag("not-a-real-category")

	assert.False(t, ok)
	assert.Equal(t, model.RuleCategoriesOrdered, categories)
}

func TestGetLintCommand_FixFileWarnsWhenNoReportedViolationsSupportAutoFix(t *testing.T) {
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)

	fixPath := filepath.Join(t.TempDir(), "fixed.yaml")
	cmd.SetArgs([]string{
		"--no-banner",
		"--no-style",
		"--fix",
		"--fix-file", fixPath,
		"../model/test_files/burgershop.openapi.yaml",
	})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.NoError(t, err)
	output := stdout + stderr + b.String()
	assert.Contains(t, output, "▲ No fixes were written to")
	assert.Contains(t, output, fixPath)
	assert.Contains(t, output, "none of the reported violations support auto-fix")

	_, statErr := os.Stat(fixPath)
	assert.True(t, os.IsNotExist(statErr))
}

func TestGetLintCommand_FixWarnsWhenNoReportedViolationsSupportAutoFix(t *testing.T) {
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--no-banner",
		"--no-style",
		"--fix",
		"../model/test_files/burgershop.openapi.yaml",
	})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.NoError(t, err)
	output := stdout + stderr + b.String()
	assert.Contains(t, output, "▲ No fixes were applied")
	assert.Contains(t, output, "none of the reported violations support auto-fix")
}

func TestRenderNoFixesAppliedWarningRespectsOutputMode(t *testing.T) {
	resultSet := &model.RuleResultSet{
		Results: []*model.RuleFunctionResult{
			{Rule: &model.Rule{}},
		},
	}

	tests := []struct {
		name       string
		flags      *LintFlags
		wantOutput bool
	}{
		{
			name: "normal terminal output",
			flags: &LintFlags{
				FixFlag:     true,
				FixFileFlag: "fixed.yaml",
			},
			wantOutput: true,
		},
		{
			name: "silent suppresses warning",
			flags: &LintFlags{
				FixFlag:     true,
				FixFileFlag: "fixed.yaml",
				SilentFlag:  true,
			},
		},
		{
			name: "pipeline output suppresses warning",
			flags: &LintFlags{
				FixFlag:        true,
				FixFileFlag:    "fixed.yaml",
				PipelineOutput: true,
			},
		},
		{
			name: "fix without fix file warns",
			flags: &LintFlags{
				FixFlag: true,
			},
			wantOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOSStreams(t, func() {
				renderNoFixesAppliedWarning(tt.flags, resultSet, 0)
			})

			output := stdout + stderr
			if tt.wantOutput {
				assert.Contains(t, output, "No fixes were")
				return
			}
			assert.Empty(t, output)
		})
	}
}

func TestGetLintCommand_QuotedResponseExampleDoesNotReportMarshalIssues(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    post:
      responses:
        "400":
          description: "Invalid input"
        "200":
          description: "Calculation successful"
          content:
            application/json:
              schema:
                type: object
                properties:
                  values:
                    type: array
                    items:
                      type: object
                      properties:
                        label:
                          type: string
                        value:
                          type: number
                        description:
                          type: string
                    example:
                      - label: "Sample"
                        value: 3.14
                        description: “score"
`)

	for range 5 {
		cmd := GetLintCommand()
		b := bytes.NewBufferString("")
		cmd.SetOut(b)
		cmd.SetErr(b)
		cmd.SetArgs([]string{
			"--fail-severity", "none",
			"--no-banner",
			"--no-style",
			"--details",
			specPath,
		})

		var err error
		stdout, stderr := captureOSStreams(t, func() {
			err = cmd.Execute()
		})

		require.NoError(t, err)
		output := stdout + stderr + b.String()
		assert.NotContains(t, output, "cannot marshal")
		assert.NotContains(t, output, "schema invalid: cannot marshal")
	}
}
