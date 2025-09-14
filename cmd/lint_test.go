package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLintCommand(t *testing.T) {
	cmd := GetLintCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "lint <your-openapi-file.yaml>", cmd.Use)
	assert.Contains(t, cmd.Short, "Lint an OpenAPI")
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
