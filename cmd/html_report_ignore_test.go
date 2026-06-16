//go:build html_report_ui

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/assert"
)

func TestHTMLReport_WithIgnoreFile_FromVacuumReport(t *testing.T) {
	// Create test OpenAPI spec with violations
	specContent := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  description: Test API
  license:
    name: MIT
servers:
  - url: https://api.example.com
paths:
  /test:
    get:
      summary: Test endpoint
      operationId: testOp
      tags:
        - test
      responses:
        '200':
          description: OK
  /another:
    post:
      summary: Another test endpoint
      operationId: anotherOp
      description: This is a test endpoint
      tags:
        - test
      responses:
        '201':
          description: Created`

	// Create ignore file that ignores one of the tag violations
	ignoreContent := `operation-tag-defined:
  - "$.paths['/test'].get.tags[0]"`

	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.yaml")
	ignoreFile := filepath.Join(tmpDir, "ignore.yaml")
	err := os.WriteFile(specFile, []byte(specContent), 0644)
	assert.NoError(t, err)

	err = os.WriteFile(ignoreFile, []byte(ignoreContent), 0644)
	assert.NoError(t, err)

	// Generate vacuum report first
	reportCmd := GetVacuumReportCommand()
	reportPrefix := filepath.Join(tmpDir, "report")
	reportCmd.SetArgs([]string{"-c", "--no-style", specFile, reportPrefix})
	err = reportCmd.Execute()
	assert.NoError(t, err)

	reportFile := requireSingleGeneratedFile(t, reportPrefix+"-*.json.gz")
	assert.NotEmpty(t, reportFile)

	// Generate HTML report without ignore file from vacuum report
	htmlCmd1 := GetHTMLReportCommand()
	noIgnoreFile := filepath.Join(tmpDir, "no-ignore.html")
	htmlCmd1.SetArgs([]string{"--no-banner", "--no-style", reportFile, noIgnoreFile})
	err = htmlCmd1.Execute()
	assert.NoError(t, err)

	// Read HTML report without ignore
	noIgnoreContent, err := os.ReadFile(noIgnoreFile)
	assert.NoError(t, err)

	// Generate HTML report with ignore file from vacuum report
	htmlCmd2 := GetHTMLReportCommand()
	withIgnoreFile := filepath.Join(tmpDir, "with-ignore.html")
	htmlCmd2.SetArgs([]string{"--no-banner", "--no-style", reportFile, withIgnoreFile, "--ignore-file", ignoreFile})
	err = htmlCmd2.Execute()
	assert.NoError(t, err)

	// Read HTML report with ignore
	withIgnoreContent, err := os.ReadFile(withIgnoreFile)
	assert.NoError(t, err)

	// Extract warning counts from headers
	extractWarningCount := func(content []byte) string {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.Contains(line, "header-statistic") && strings.Contains(line, "Warnings") {
				// Extract value from header-statistic value='X' label='Warnings'
				start := strings.Index(line, "value='") + 7
				end := strings.Index(line[start:], "'")
				if start > 6 && end > 0 {
					return line[start : start+end]
				}
			}
		}
		return ""
	}

	noIgnoreWarnings := extractWarningCount(noIgnoreContent)
	withIgnoreWarnings := extractWarningCount(withIgnoreContent)

	// The report without ignore should have 2 warnings (both tag violations)
	assert.Equal(t, "2", noIgnoreWarnings, "Expected 2 warnings without ignore file")

	// The report with ignore should have 1 warning (one tag violation ignored)
	assert.Equal(t, "1", withIgnoreWarnings, "Expected 1 warning with ignore file")

	// Also check that the body content has filtered results
	// The ignored rule should not appear in the with-ignore report
	assert.Contains(t, string(noIgnoreContent), "operation-tag-defined", "Non-ignored report should contain operation-tag-defined rule")
	// Note: The HTML report might still show the rule but with no violations, or might hide it entirely
	// This depends on the implementation details
}
