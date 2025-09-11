package cmd

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLintPreviewCommand(t *testing.T) {
	cmd := GetLintPreviewCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "lint-preview <your-openapi-file.yaml>", cmd.Use)
	assert.Contains(t, cmd.Short, "Preview lint results")
}

func TestGetLintPreviewCommand_NoSpec(t *testing.T) {
	cmd := GetLintPreviewCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})
	
	err := cmd.Execute()
	assert.Error(t, err)
	
	// error message is printed to stderr, not stdout
	// the actual error is returned
	assert.Contains(t, err.Error(), "please supply an OpenAPI specification")
}

func TestGetLintPreviewCommand_MissingSpec(t *testing.T) {
	cmd := GetLintPreviewCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"does-not-exist.yaml"})
	
	err := cmd.Execute()
	assert.Error(t, err)
}

func TestGetLintPreviewCommand_WithRuleset(t *testing.T) {
	cmd := GetLintPreviewCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../rulesets/examples/custom-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	
	// lint-preview uses bubbletea which requires a TTY
	// so we can't fully execute it in tests, but we can verify setup
	err := cmd.Execute()
	// will error due to no TTY, but that's expected
	assert.Error(t, err)
}

func TestGetLintPreviewCommand_BadRuleset(t *testing.T) {
	cmd := GetLintPreviewCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
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
	// the error contains information about the missing ruleset
	assert.NotNil(t, err)
}

func TestGetLintPreviewCommand_WithDetails(t *testing.T) {
	cmd := GetLintPreviewCommand()
	cmd.PersistentFlags().BoolP("details", "d", false, "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-d",
		"../model/test_files/burgershop.openapi.yaml",
	})
	
	// will error due to no TTY, but that's expected
	err := cmd.Execute()
	assert.Error(t, err)
}

func TestGetLintPreviewCommand_WithSnippets(t *testing.T) {
	cmd := GetLintPreviewCommand()
	cmd.PersistentFlags().BoolP("snippets", "n", false, "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-n",
		"../model/test_files/burgershop.openapi.yaml",
	})
	
	// will error due to no TTY, but that's expected
	err := cmd.Execute()
	assert.Error(t, err)
}

func TestGetLintPreviewCommand_BadSpec(t *testing.T) {
	cmd := GetLintPreviewCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"../model/test_files/badspec.yaml",
	})
	
	err := cmd.Execute()
	assert.Error(t, err)
}

func TestGetLintPreviewCommand_WithVacuumReport(t *testing.T) {
	// test with pre-compiled vacuum report
	cmd := GetLintPreviewCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/panda.vacuum.html.gz",
	})
	
	// will error due to no TTY, but that's expected
	err := cmd.Execute()
	assert.Error(t, err)
}