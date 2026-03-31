//go:build !html_report_ui

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHTMLReportCommand_WithoutUIAssets(t *testing.T) {
	cmd := GetHTMLReportCommand()
	cmd.SetArgs([]string{
		"../model/test_files/burgershop-report.json.gz",
		"test-report.html",
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.ErrorContains(t, err, "html-report support is not included in this build")
}
