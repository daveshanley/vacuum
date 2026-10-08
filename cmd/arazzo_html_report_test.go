//go:build html_report_ui

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestArazzoHTMLReportIncludesFindings(t *testing.T) {
	path, spec := arazzoCommandFixture(t)
	dir := t.TempDir()
	input, report := filepath.Join(dir, "workflow.yaml"), filepath.Join(dir, "report.html")
	require.NoError(t, os.WriteFile(input, []byte(strings.ReplaceAll(spec, "$sourceDescriptions.api.read", "$sourceDescriptions.api.missing")), 0600))
	output, code := runVacuum(t, "html-report", input, report, "--base", filepath.Dir(path), "--no-update-check")
	require.Equal(t, 0, code, "%s", output)
	data, err := os.ReadFile(report)
	require.NoError(t, err)
	require.Contains(t, string(data), `message="operation &#34;missing&#34; does not exist in source &#34;api&#34;"`)
	require.Contains(t, string(data), "arazzo-reference")
	require.Contains(t, string(data), "workflows[0].steps[0].operationId")
	require.NotContains(t, string(data), "failed to render:")
}
