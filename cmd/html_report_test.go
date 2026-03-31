//go:build html_report_ui

package cmd

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func TestGetHTMLReportCommand(t *testing.T) {
	cmd := GetHTMLReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
		"test-report.html",
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	defer os.Remove("test-report.html")
}

func TestGetHTMLReportCommand_NoRuleset(t *testing.T) {
	cmd := GetHTMLReportCommand()

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/burgershop.openapi.yaml",
		"test-report.html",
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)
	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	defer os.Remove("test-report.html")
}

func TestGetHTMLReportCommand_LoadReport(t *testing.T) {
	cmd := GetHTMLReportCommand()

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/burgershop-report.json.gz",
		"test-report.html",
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)
	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	defer os.Remove("test-report.html")
}

func TestGetHTMLReportCommand_NoArgs(t *testing.T) {
	cmd := GetHTMLReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetHTMLReportCommand_BadWrite(t *testing.T) {
	cmd := GetHTMLReportCommand()

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/burgershop-report.json.gz",
		"/cant-write-here/no/stop.html",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetHTMLReportCommand_UnparseableSpec(t *testing.T) {
	cmd := GetHTMLReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../rulesets/examples/all-ruleset.yaml",
		"test-report-bad.html",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
	var exitErr *ExitError
	assert.ErrorAs(t, cmdErr, &exitErr)
	assert.Equal(t, ExitCodeInputError, exitErr.Code)
	// Ensure no report file was written
	_, statErr := os.Stat("test-report-bad.html")
	assert.True(t, os.IsNotExist(statErr))
}
