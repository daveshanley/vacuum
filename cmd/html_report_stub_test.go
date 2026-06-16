//go:build !html_report_ui

package cmd

import (
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/assert"
)

func TestGetHTMLReportCommand_WithoutUIAssets(t *testing.T) {
	cmd := GetHTMLReportCommand()
	cmd.SetArgs([]string{
		"../model/test_files/burgershop-report.json.gz",
		filepath.Join(t.TempDir(), "report.html"),
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.ErrorContains(t, err, "html-report support is not included in this build")
}
