//go:build html_report_ui

package cmd

import (
	"bytes"
	"golang.org/x/net/html"
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

func TestArazzoHTMLReportUsesExternalSourceSnippet(t *testing.T) {
	input := arazzoExternalSourceFixture(t)
	output, code := runVacuum(t, "report", input, "--stdout", "--no-update-check")
	require.Equal(t, 0, code, "%s", output)
	dir := t.TempDir()
	saved := filepath.Join(dir, "report.json")
	require.NoError(t, os.WriteFile(saved, output, 0600))
	for i, document := range []string{input, saved} {
		if i == 1 {
			require.NoError(t, os.Remove(filepath.Join(filepath.Dir(input), "other.yaml")))
		}
		report := filepath.Join(dir, "report.html")
		output, code = runVacuum(t, "html-report", document, report, "--no-update-check")
		require.Equal(t, 0, code, "%s", output)
		data, err := os.ReadFile(report)
		require.NoError(t, err)
		require.Contains(t, string(data), "Source: "+filepath.Join(filepath.Dir(input), "other.yaml")+":6")
		require.Contains(t, string(data), "$steps.missing.outputs.value")
		snippet := firstReportSnippetText(t, data)
		require.Contains(t, snippet, "other.yaml:6")
		require.Contains(t, snippet, "$steps.missing.outputs.value")
		require.NotContains(t, string(data), "This is the ROOT description.")
	}
}

// The report component moves its first slotted element into the details pane.
// Verify the visible element, not merely text elsewhere in the generated file.
func firstReportSnippetText(t *testing.T, data []byte) string {
	t.Helper()
	document, err := html.Parse(bytes.NewReader(data))
	require.NoError(t, err)
	var find func(*html.Node) *html.Node
	find = func(node *html.Node) *html.Node {
		if node.Type == html.ElementNode && node.Data == "category-rule-result" {
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode {
					return child
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if found := find(child); found != nil {
				return found
			}
		}
		return nil
	}
	snippet := find(document)
	require.NotNil(t, snippet)
	var text strings.Builder
	var collect func(*html.Node)
	collect = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			collect(child)
		}
	}
	collect(snippet)
	return text.String()
}
