package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
