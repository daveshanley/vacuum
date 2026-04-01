//go:build html_report_ui

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTMLReport_OriginalSameSpec(t *testing.T) {
	spec := "../model/test_files/burgershop.openapi.yaml"
	reportFile := filepath.Join(t.TempDir(), "html-original-test.html")

	cmd := GetHTMLReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"-b",
		spec,
		reportFile,
	})

	err := cmd.Execute()
	assert.NoError(t, err)

	data, readErr := os.ReadFile(reportFile)
	require.NoError(t, readErr)
	assert.True(t, len(data) > 0)
}

func TestHTMLReport_PrecompiledReport_OriginalFallback(t *testing.T) {
	report := "../model/test_files/burgershop-report.json.gz"
	spec := "../model/test_files/burgershop.openapi.yaml"
	reportFile := filepath.Join(t.TempDir(), "html-precompiled-test.html")

	cmd := GetHTMLReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"-b",
		report,
		reportFile,
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}
