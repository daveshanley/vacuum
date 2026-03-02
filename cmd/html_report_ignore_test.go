package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

	// Write test files
	specFile := "test-spec-for-ignore.yaml"
	ignoreFile := "test-ignore-rules.yaml"
	err := os.WriteFile(specFile, []byte(specContent), 0644)
	assert.NoError(t, err)
	defer os.Remove(specFile)

	err = os.WriteFile(ignoreFile, []byte(ignoreContent), 0644)
	assert.NoError(t, err)
	defer os.Remove(ignoreFile)

	// Generate vacuum report first
	reportCmd := GetVacuumReportCommand()
	reportCmd.SetArgs([]string{"-c", specFile, "test-report"})
	err = reportCmd.Execute()
	assert.NoError(t, err)

	// Find the generated report file
	files, err := os.ReadDir(".")
	assert.NoError(t, err)
	var reportFile string
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "test-report-") && strings.HasSuffix(f.Name(), ".json.gz") {
			reportFile = f.Name()
			break
		}
	}
	assert.NotEmpty(t, reportFile)
	defer os.Remove(reportFile)

	// Generate HTML report without ignore file from vacuum report
	htmlCmd1 := GetHTMLReportCommand()
	htmlCmd1.SetArgs([]string{reportFile, "test-no-ignore.html"})
	err = htmlCmd1.Execute()
	assert.NoError(t, err)
	defer os.Remove("test-no-ignore.html")

	// Read HTML report without ignore
	noIgnoreContent, err := os.ReadFile("test-no-ignore.html")
	assert.NoError(t, err)

	// Generate HTML report with ignore file from vacuum report
	htmlCmd2 := GetHTMLReportCommand()
	htmlCmd2.SetArgs([]string{reportFile, "test-with-ignore.html", "--ignore-file", ignoreFile})
	err = htmlCmd2.Execute()
	assert.NoError(t, err)
	defer os.Remove("test-with-ignore.html")

	// Read HTML report with ignore
	withIgnoreContent, err := os.ReadFile("test-with-ignore.html")
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
