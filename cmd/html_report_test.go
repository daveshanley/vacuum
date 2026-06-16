//go:build html_report_ui

package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestGetHTMLReportCommand(t *testing.T) {
	cmd := GetHTMLReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	reportFile := filepath.Join(t.TempDir(), "report.html")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--no-banner",
		"--no-style",
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
		reportFile,
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	_, statErr := os.Stat(reportFile)
	assert.NoError(t, statErr)
}

func TestGetHTMLReportCommand_NoRuleset(t *testing.T) {
	cmd := GetHTMLReportCommand()

	b := bytes.NewBufferString("")
	reportFile := filepath.Join(t.TempDir(), "report.html")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--no-banner",
		"--no-style",
		"../model/test_files/burgershop.openapi.yaml",
		reportFile,
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)
	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	_, statErr := os.Stat(reportFile)
	assert.NoError(t, statErr)
}

func TestGetHTMLReportCommand_LoadReport(t *testing.T) {
	cmd := GetHTMLReportCommand()

	b := bytes.NewBufferString("")
	reportFile := filepath.Join(t.TempDir(), "report.html")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--no-banner",
		"--no-style",
		"../model/test_files/burgershop-report.json.gz",
		reportFile,
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)
	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	_, statErr := os.Stat(reportFile)
	assert.NoError(t, statErr)
}

func TestGetHTMLReportCommand_NoArgs(t *testing.T) {
	cmd := GetHTMLReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetHTMLReportCommand_BadWrite(t *testing.T) {
	cmd := GetHTMLReportCommand()

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--no-banner",
		"--no-style",
		"../model/test_files/burgershop-report.json.gz",
		"/cant-write-here/no/stop.html",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetHTMLReportCommand_UnparseableSpec(t *testing.T) {
	cmd := GetHTMLReportCommand()
	b := bytes.NewBufferString("")
	reportFile := filepath.Join(t.TempDir(), "bad-report.html")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--no-banner",
		"--no-style",
		"../rulesets/examples/all-ruleset.yaml",
		reportFile,
	})
	cmdErr := cmd.Execute()
	require.Error(t, cmdErr)
	var exitErr *ExitError
	require.ErrorAs(t, cmdErr, &exitErr)
	assert.Equal(t, ExitCodeInputError, exitErr.Code)
	// Ensure no report file was written
	_, statErr := os.Stat(reportFile)
	assert.True(t, os.IsNotExist(statErr))
}
