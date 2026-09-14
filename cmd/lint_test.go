package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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
			name: "github annotations suppresses warning",
			flags: &LintFlags{
				FixFlag:           true,
				FixFileFlag:       "fixed.yaml",
				GitHubAnnotations: true,
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

func TestRenderResultPathsTruncatedWarningRespectsOutputMode(t *testing.T) {
	tests := []struct {
		name       string
		flags      *LintFlags
		fileName   string
		wantOutput bool
	}{
		{name: "normal output", flags: &LintFlags{}, wantOutput: true},
		{name: "file context", flags: &LintFlags{}, fileName: "openapi.yaml", wantOutput: true},
		{name: "silent", flags: &LintFlags{SilentFlag: true}},
		{name: "pipeline output", flags: &LintFlags{PipelineOutput: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOSStreams(t, func() {
				renderResultPathsTruncatedWarning(tt.flags, tt.fileName)
			})
			output := stdout + stderr
			if !tt.wantOutput {
				assert.Empty(t, output)
				return
			}
			assert.Contains(t, output, "resolved result aliases were truncated")
			if tt.fileName != "" {
				assert.Contains(t, output, tt.fileName)
			}
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

func TestGetLintCommand_GitHubAnnotations_SingleFileSuppressesSummaryAndEmitsAnnotations(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::")
	assert.Contains(t, output, "title=")
	assert.NotContains(t, output, "vacuuming file")
	assert.NotContains(t, output, "RULE")
	assert.NotContains(t, output, "violations")
}

func TestGetLintCommand_GitHubAnnotations_SingleFileLoadErrorEmitsAnnotation(t *testing.T) {
	missingSpec := filepath.Join(t.TempDir(), "missing.yaml")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", missingSpec})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error ")
	assert.Contains(t, output, "no such file or directory")
	assert.NotContains(t, output, "\033[31mUnable to load file")
}

func TestGetLintCommand_GitHubAnnotations_SuppressesHardModeAndTurboAndShowRulesChrome(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)

	rulesetPath := filepath.Join(t.TempDir(), "ruleset.yaml")
	writeTestFile(t, rulesetPath, `extends: [[spectral:oas, recommended]]
`)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{
		"--github-annotations",
		"--no-style",
		"--hard-mode",
		"--turbo",
		"--show-rules",
		"-r", rulesetPath,
		specPath,
	})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})
	_ = err

	output := stdout + stderr
	assert.NotContains(t, output, "turbo mode")
	assert.NotContains(t, output, "OWASP Rules added to custom ruleset")
	assert.NotContains(t, output, "The following rules are going to be used")
}

func TestGetLintCommand_GitHubAnnotations_MultiFileWithPipelineOutputEmitsAnnotationsAndMarkdown(t *testing.T) {
	dir := t.TempDir()
	firstSpec := filepath.Join(dir, "first.yaml")
	secondSpec := filepath.Join(dir, "second.yaml")
	content := `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`
	writeTestFile(t, firstSpec, content)
	writeTestFile(t, secondSpec, content)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--pipeline-output", firstSpec, secondSpec})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "# 📄 `")
	assert.GreaterOrEqual(t, strings.Count(output, "::"), 2)
	assert.Contains(t, output, "title=")
}

func TestGetLintCommand_GitHubAnnotations_MultiFileWithPipelineOutputIncludesInputErrors(t *testing.T) {
	dir := t.TempDir()
	goodSpec := filepath.Join(dir, "good.yaml")
	missingSpec := filepath.Join(dir, "missing.yaml")
	writeTestFile(t, goodSpec, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--pipeline-output", goodSpec, missingSpec})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "# 📄 `")
	assert.Contains(t, output, "missing.yaml")
	assert.Contains(t, output, "::error ")
	assert.Contains(t, output, "no such file or directory")
	assert.Contains(t, output, "## ❌ Error processing `")
}

func TestGetLintCommand_GitHubAnnotations_SingleFilePipelineCombinedSuppressesLoadErrorChrome(t *testing.T) {
	dir := t.TempDir()
	missingSpec := filepath.Join(dir, "missing.yaml")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--pipeline-output", missingSpec})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error ")
	assert.Contains(t, output, "no such file or directory")
	assert.NotContains(t, output, "Unable to load file")
	assert.NotContains(t, output, "\033[31m")
}

func TestGetLintCommand_GitHubAnnotations_MultiFileAnnotationOnlyEmitsAnnotations(t *testing.T) {
	dir := t.TempDir()
	firstSpec := filepath.Join(dir, "first.yaml")
	secondSpec := filepath.Join(dir, "second.yaml")
	content := `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`
	writeTestFile(t, firstSpec, content)
	writeTestFile(t, secondSpec, content)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", firstSpec, secondSpec})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.GreaterOrEqual(t, strings.Count(output, "::"), 2)
	assert.Contains(t, output, "title=")
	assert.NotContains(t, output, "vacuuming")
	assert.NotContains(t, output, "# 📄 `")
	assert.NotContains(t, output, "RULE")

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for _, line := range lines {
		assert.Equal(t, 2, strings.Count(line, "::"), line)
	}
	assert.GreaterOrEqual(t, len(lines), 2)
	assert.Contains(t, output, "first.yaml")
	assert.Contains(t, output, "second.yaml")
}

func TestGetLintCommand_GitHubAnnotations_MultiFileAnnotationOnlyReportsInputErrors(t *testing.T) {
	dir := t.TempDir()
	goodSpec := filepath.Join(dir, "good.yaml")
	missingSpec := filepath.Join(dir, "missing.yaml")
	writeTestFile(t, goodSpec, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", goodSpec, missingSpec})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "missing.yaml")
	assert.Contains(t, output, "::error ")
	assert.Contains(t, output, "no such file or directory")
}

func TestGetLintCommand_GitHubAnnotations_AnnotationOnlySuppressesIgnoreFileInfo(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	ignoreFile := filepath.Join(dir, "ignore.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)
	writeTestFile(t, ignoreFile, "{}\n")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", "--ignore-file", ignoreFile, specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::")
	assert.NotContains(t, output, "Using ignore file")
	assert.NotContains(t, output, "ignored items")
}

func TestGetLintCommand_GitHubAnnotations_AnnotationOnlySuppressesUnknownCategoryWarning(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", "--category", "not-a-real-category", specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::")
	assert.NotContains(t, output, "unknown, all categories are being considered")
	assert.NotContains(t, output, "Warning")
}

func TestGetLintCommand_GitHubAnnotations_AnnotationOnlySuppressesMinScoreText(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "200":
          description: ok
`)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", "--min-score", "99", specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::")
	assert.NotContains(t, output, "SCORE THRESHOLD FAILED")
	assert.NotContains(t, output, "Overall score is")
}

func TestGetLintCommand_GitHubAnnotations_AnnotationOnlySuppressesChangesLoadWarning(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)
	missingChanges := filepath.Join(dir, "changes-does-not-exist.json")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", "--changes", missingChanges, specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::")
	assert.NotContains(t, output, "Warning: Failed to load changes")
	assert.NotContains(t, output, "Proceeding without change filtering")
}

func TestGetLintCommand_GitHubAnnotations_AnnotationOnlySuppressesOriginalLintWarning(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)
	missingOriginal := filepath.Join(dir, "original-does-not-exist.yaml")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", "--original", missingOriginal, specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::")
	assert.NotContains(t, output, "Warning: Failed to load changes")
	assert.NotContains(t, output, "Warning: Failed to lint original spec")
	assert.NotContains(t, output, "Proceeding without change filtering")
}

func TestGetLintCommand_GitHubAnnotations_AnnotationOnlySuppressesComparisonModeChrome(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	originalPath := filepath.Join(dir, "original.yaml")
	specBody := `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`
	writeTestFile(t, specPath, specBody)
	writeTestFile(t, originalPath, specBody)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", "--original", originalPath, specPath})

	stdout, stderr := captureOSStreams(t, func() {
		_ = cmd.Execute()
	})

	output := stdout + stderr
	assert.NotContains(t, output, WhatChangedModeMessage)
}

func TestGetLintCommand_GitHubAnnotations_AnnotationOnlySuppressesPrecompiledReportChrome(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)
	reportPrefix := filepath.Join(dir, "vacuum-report")

	reportCmd := GetVacuumReportCommand()
	registerPersistentFlags(reportCmd)
	reportCmd.SetOut(bytes.NewBuffer(nil))
	reportCmd.SetErr(bytes.NewBuffer(nil))
	reportCmd.SetArgs([]string{"--no-style", "--no-pretty", specPath, reportPrefix})
	require.NoError(t, reportCmd.Execute())

	reportFiles, globErr := filepath.Glob(reportPrefix + "-*.json")
	require.NoError(t, globErr)
	require.Len(t, reportFiles, 1)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", reportFiles[0]})

	stdout, stderr := captureOSStreams(t, func() {
		_ = cmd.Execute()
	})

	output := stdout + stderr
	assert.Contains(t, output, "::")
	assert.NotContains(t, output, "Loading pre-compiled vacuum report")
}

func TestGetLintCommand_GitHubAnnotations_EmitsAnnotationForBadIgnoreFile(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)
	missingIgnore := filepath.Join(dir, "missing-ignore.yaml")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", "--ignore-file", missingIgnore, specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.Contains(t, output, "missing-ignore.yaml")
	assert.NotContains(t, output, "\033[31m")
	assert.NotContains(t, output, "Error: Failed to read ignore file")
}

func TestGetLintCommand_GitHubAnnotations_CombinedPipelineSuppressesIgnoreFileErrorChrome(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)
	missingIgnore := filepath.Join(dir, "missing-ignore.yaml")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--pipeline-output", "--ignore-file", missingIgnore, specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.NotContains(t, output, "\033[31m")
	assert.NotContains(t, output, "Error: Failed to read ignore file")
}

func TestGetLintCommand_GitHubAnnotations_EmitsAnnotationForBadRuleset(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)
	missingRuleset := filepath.Join(dir, "missing-ruleset.yaml")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", "--ruleset", missingRuleset, specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.NotContains(t, output, "\033[31m")
	assert.NotContains(t, output, "Unable to load ruleset")
}

func TestGetLintCommand_GitHubAnnotations_NoFileEmitsAnnotation(t *testing.T) {
	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style"})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.Contains(t, output, "please supply an OpenAPI or AsyncAPI specification")
	assert.NotContains(t, output, "🚨")
}

func TestGetLintCommand_GitHubAnnotations_CombinedPipelineSuppressesRulesetErrorChrome(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)
	missingRuleset := filepath.Join(dir, "missing-ruleset.yaml")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--pipeline-output", "--ruleset", missingRuleset, specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.NotContains(t, output, "\033[31m")
	assert.NotContains(t, output, "Unable to load ruleset")
}

func TestResolveBasePathForFile(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "nested", "openapi.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(specPath), 0o755))
	writeTestFile(t, specPath, "openapi: 3.0.3\ninfo:\n  title: t\n  version: 1.0.0\npaths: {}\n")

	t.Run("uses explicit base flag", func(t *testing.T) {
		basePath, err := ResolveBasePathForFile(specPath, "..")
		require.NoError(t, err)
		expected, absErr := filepath.Abs("..")
		require.NoError(t, absErr)
		assert.Equal(t, expected, basePath)
	})

	t.Run("defaults to spec directory", func(t *testing.T) {
		basePath, err := ResolveBasePathForFile(specPath, "")
		require.NoError(t, err)
		assert.Equal(t, filepath.Dir(specPath), basePath)
	})
}

func TestResolveSpecPathForExecution(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	writeTestFile(t, specPath, "openapi: 3.0.3\ninfo:\n  title: t\n  version: 1.0.0\npaths: {}\n")

	resolvedPath, err := ResolveSpecPathForExecution(specPath)
	require.NoError(t, err)
	assert.Equal(t, specPath, resolvedPath)

	stdinPath, err := ResolveSpecPathForExecution("stdin")
	require.NoError(t, err)
	assert.Equal(t, "stdin", stdinPath)

	urlPath, err := ResolveSpecPathForExecution("https://example.com/openapi.yaml")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/openapi.yaml", urlPath)

	emptyPath, err := ResolveSpecPathForExecution("")
	require.NoError(t, err)
	assert.Empty(t, emptyPath)
}

func TestReadLintFlags_GitHubAnnotations(t *testing.T) {
	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	require.NoError(t, cmd.ParseFlags([]string{"--github-annotations", "--pipeline-output", "--no-style", "../model/test_files/burgershop.openapi.yaml"}))

	flags := ReadLintFlags(cmd)
	assert.True(t, flags.GitHubAnnotations)
	assert.True(t, flags.PipelineOutput)
	assert.True(t, flags.NoStyleFlag)
}

func TestGetLintCommand_GitHubAnnotations_DebugSuppressesBufferedLogs(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--debug", "--no-style", specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})
	_ = err

	output := stdout + stderr
	assert.Contains(t, output, "::")
	assert.NotContains(t, output, "DEV ")
	assert.NotContains(t, output, "applying rules to rule set")
	assert.NotContains(t, output, "building documents")
	assert.NotContains(t, output, "rolodex indexed")

	// every non-empty line must be a GitHub annotation command
	for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
		if line == "" {
			continue
		}
		assert.True(t, strings.HasPrefix(line, "::"), "unexpected non-annotation line on stdout: %q", line)
	}
}

func TestGetLintCommand_GitHubAnnotations_EmitsAnnotationForMinScoreFailure(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "200":
          description: ok
`)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--github-annotations", "--no-style", "--min-score", "99", specPath})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, ExitCodeViolations, exitErr.Code)

	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.Contains(t, output, "score threshold failed")
	assert.Contains(t, output, "threshold is 99")
	// human-oriented chrome must still be suppressed
	assert.NotContains(t, output, "SCORE THRESHOLD FAILED")
	assert.NotContains(t, output, "Overall score is")
}

func TestGetLintCommand_GitHubAnnotations_EmitsAnnotationForTLSConfigFailure(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths: {}
`)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	// setting only --cert-file (without --key-file) causes CreateCustomHTTPClient
	// to fail deterministically with "both cert-file and key-file must be provided together"
	cmd.SetArgs([]string{
		"--github-annotations", "--no-style",
		"--cert-file", filepath.Join(t.TempDir(), "does-not-exist.pem"),
		specPath,
	})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.Contains(t, output, "failed to create HTTP client")
	// no ANSI chrome should leak
	assert.NotContains(t, output, "\033[31m")
}

func TestGetLintCommand_GitHubAnnotations_EmitsAnnotationForFetchConfigFailure(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	writeTestFile(t, specPath, `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths: {}
`)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{
		"--github-annotations", "--no-style",
		"--fetch-timeout", "-1",
		specPath,
	})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.Contains(t, output, "failed to resolve fetch configuration")
}

func TestGetLintCommand_GitHubAnnotations_MultiFileEmitsAnnotationForFetchConfigFailure(t *testing.T) {
	dir := t.TempDir()
	firstSpec := filepath.Join(dir, "first.yaml")
	secondSpec := filepath.Join(dir, "second.yaml")
	content := `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths: {}
`
	writeTestFile(t, firstSpec, content)
	writeTestFile(t, secondSpec, content)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{
		"--github-annotations", "--no-style",
		"--fetch-timeout", "-1",
		firstSpec, secondSpec,
	})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.Contains(t, output, "failed to resolve fetch configuration")
}

// annotationTestSpec is a minimal, valid spec used by the loader-error tests below.
const annotationTestSpec = `
openapi: 3.0.3
info:
  title: Example API
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        "default":
          description: ok
`

func TestGetLintCommand_MultiFile_BadIgnoreFileFailsWithFailSeverityNone(t *testing.T) {
	dir := t.TempDir()
	firstSpec := filepath.Join(dir, "first.yaml")
	secondSpec := filepath.Join(dir, "second.yaml")
	writeTestFile(t, firstSpec, annotationTestSpec)
	writeTestFile(t, secondSpec, annotationTestSpec)
	missingIgnore := filepath.Join(dir, "missing-ignore.yaml")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{
		"--no-style", "--fail-severity", "none",
		"--ignore-file", missingIgnore,
		firstSpec, secondSpec,
	})

	var err error
	captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read ignore file")
}

func TestGetLintCommand_GitHubAnnotations_MultiFileEmitsAnnotationForBadIgnoreFile(t *testing.T) {
	dir := t.TempDir()
	firstSpec := filepath.Join(dir, "first.yaml")
	secondSpec := filepath.Join(dir, "second.yaml")
	writeTestFile(t, firstSpec, annotationTestSpec)
	writeTestFile(t, secondSpec, annotationTestSpec)
	missingIgnore := filepath.Join(dir, "missing-ignore.yaml")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{
		"--github-annotations", "--no-style", "--fail-severity", "none",
		"--ignore-file", missingIgnore,
		firstSpec, secondSpec,
	})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.Contains(t, output, "missing-ignore.yaml")
	assert.NotContains(t, output, "Error: Failed to read ignore file")
	// the run must abort before any file is linted
	assert.NotContains(t, output, "first.yaml")
}

func TestGetLintCommand_SingleFile_BadFunctionsPathFailsWithFailSeverityNone(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	writeTestFile(t, specPath, annotationTestSpec)
	missingFunctions := filepath.Join(dir, "missing-functions")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{
		"--no-style", "--fail-severity", "none",
		"--functions", missingFunctions,
		specPath,
	})

	var err error
	captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
}

func TestGetLintCommand_GitHubAnnotations_SingleFileEmitsAnnotationForBadFunctionsPath(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	writeTestFile(t, specPath, annotationTestSpec)
	missingFunctions := filepath.Join(dir, "missing-functions")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{
		"--github-annotations", "--no-style", "--fail-severity", "none",
		"--functions", missingFunctions,
		specPath,
	})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.Contains(t, output, "missing-functions")
	assert.NotContains(t, output, "✗")
}

func TestGetLintCommand_GitHubAnnotations_MultiFileEmitsAnnotationForBadFunctionsPath(t *testing.T) {
	dir := t.TempDir()
	firstSpec := filepath.Join(dir, "first.yaml")
	secondSpec := filepath.Join(dir, "second.yaml")
	writeTestFile(t, firstSpec, annotationTestSpec)
	writeTestFile(t, secondSpec, annotationTestSpec)
	missingFunctions := filepath.Join(dir, "missing-functions")

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{
		"--github-annotations", "--no-style", "--fail-severity", "none",
		"--functions", missingFunctions,
		firstSpec, secondSpec,
	})

	var err error
	stdout, stderr := captureOSStreams(t, func() {
		err = cmd.Execute()
	})

	require.Error(t, err)
	output := stdout + stderr
	assert.Contains(t, output, "::error")
	assert.Contains(t, output, "missing-functions")
	assert.NotContains(t, output, "✗")
	// the run must abort before any file is linted
	assert.NotContains(t, output, "first.yaml")
}

func TestGetLintCommand_GitHubAnnotations_ValidCustomFunctionsEmitNoTerminalChrome(t *testing.T) {
	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi.yaml")
	writeTestFile(t, specPath, annotationTestSpec)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{
		"--github-annotations", "--no-style",
		"--functions", filepath.Join("..", "plugin", "sample", "js"),
		specPath,
	})

	stdout, stderr := captureOSStreams(t, func() {
		_ = cmd.Execute()
	})

	output := stdout + stderr
	assert.NotContains(t, output, "Located custom javascript function")
	assert.NotContains(t, output, "Registered custom function")
	assert.NotContains(t, output, "Successfully validated JavaScript function")
	assert.NotContains(t, output, "custom function(s) successfully")
	assert.NotContains(t, output, "Available custom functions")
	assertAnnotationsOnly(t, output)
}

func TestGetLintCommand_GitHubAnnotations_MultiFileValidCustomFunctionsEmitNoTerminalChrome(t *testing.T) {
	dir := t.TempDir()
	firstSpec := filepath.Join(dir, "first.yaml")
	secondSpec := filepath.Join(dir, "second.yaml")
	writeTestFile(t, firstSpec, annotationTestSpec)
	writeTestFile(t, secondSpec, annotationTestSpec)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{
		"--github-annotations", "--no-style",
		"--functions", filepath.Join("..", "plugin", "sample", "js"),
		firstSpec, secondSpec,
	})

	stdout, stderr := captureOSStreams(t, func() {
		_ = cmd.Execute()
	})

	output := stdout + stderr
	assert.NotContains(t, output, "Located custom javascript function")
	assert.NotContains(t, output, "Registered custom function")
	assert.NotContains(t, output, "custom function(s) successfully")
	assertAnnotationsOnly(t, output)
}

// assertAnnotationsOnly fails the test if any non-empty line of output is not a
// GitHub Actions workflow command.
func assertAnnotationsOnly(t *testing.T, output string) {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		assert.True(t, strings.HasPrefix(line, "::"), "unexpected non-annotation output: %q", line)
	}
}
