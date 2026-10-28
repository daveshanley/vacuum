package vacuum_report

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/testify/require"
)

func exportFixture() []Diagnostic {
	return []Diagnostic{{Code: "test-rule", Path: []any{"paths", "/pets", 0, "a.b"}, Message: "missing <tag> & 'value' |\nnext", Severity: 0, Range: reports.Range{Start: reports.RangeItem{Line: 4, Char: 2}, End: reports.RangeItem{Line: 4, Char: 8}}, Source: "api/spec.yaml"}, {Code: "warning", Path: []any{}, Message: "warning", Severity: 1, Range: reports.Range{Start: reports.RangeItem{}, End: reports.RangeItem{}}, Source: "api/other.yaml"}}
}
func TestReportFormats(t *testing.T) {
	for _, format := range Formats() {
		t.Run(format, func(t *testing.T) {
			for _, results := range [][]Diagnostic{nil, exportFixture()} {
				var b bytes.Buffer
				require.NoError(t, WriteFormat(&b, format, results, FormatOptions{Version: "test", FailSeverity: 0}))
				switch format {
				case "json", "sarif", "code-climate", "gitlab":
					require.True(t, json.Valid(b.Bytes()), b.String())
				case "junit":
					var suites TestSuites
					require.NoError(t, xml.Unmarshal(b.Bytes(), &suites))
					require.Equal(t, len(results), suites.Tests)
					if len(results) > 0 {
						require.Equal(t, 1, suites.Failures)
						require.Equal(t, 5, suites.TestSuites[0].TestCases[0].Line)
						require.Equal(t, results[0].Message, suites.TestSuites[0].TestCases[0].Failure.Message)
					}
				case "html":
					require.Contains(t, b.String(), "<!doctype html>")
					require.NotContains(t, b.String(), "<tag>")
					if len(results) > 0 {
						require.Contains(t, b.String(), "&lt;tag&gt;")
					}
				case "markdown":
					require.Contains(t, b.String(), "| Code")
					require.NotContains(t, b.String(), "<tag>")
					if len(results) > 0 {
						require.Contains(t, b.String(), "\\|<br>")
					}
				case "teamcity":
					if len(results) > 0 {
						require.Contains(t, b.String(), "line='5'")
						require.Contains(t, b.String(), "|'value|' |||nnext")
						require.Len(t, strings.Split(strings.TrimSpace(b.String()), "\n"), 4)
					}
				case "github-actions":
					if len(results) > 0 {
						require.Contains(t, b.String(), "line=5")
						require.Contains(t, b.String(), "%0Anext")
					}
				case "text":
					if len(results) > 0 {
						require.Contains(t, b.String(), "api/spec.yaml:5:3 error")
					}
				default:
					require.Contains(t, b.String(), "findings")
				}
			}
		})
	}
}
func TestReportSARIFLocationsAndRules(t *testing.T) {
	r := exportFixture()
	r[0].Source = "/repo/api/a b.yaml"
	var b bytes.Buffer
	require.NoError(t, WriteFormat(&b, "sarif", r, FormatOptions{Version: "1.2.3", BaseDirectory: "/repo"}))
	var doc sarifDocument
	require.NoError(t, json.Unmarshal(b.Bytes(), &doc))
	require.Equal(t, "2.1.0", doc.Version)
	run := doc.Runs[0]
	require.Equal(t, "vacuum", run.Tool.Driver.Name)
	require.Len(t, run.Tool.Driver.Rules, 2)
	require.Equal(t, "error", run.Results[0].Level)
	loc := run.Results[0].Locations[0].PhysicalLocation
	require.Equal(t, "api/a%20b.yaml", loc.ArtifactLocation.URI)
	require.Equal(t, 5, loc.Region.StartLine)
	require.Equal(t, 3, loc.Region.StartColumn)
	require.Equal(t, 9, loc.Region.EndColumn)
}
func TestReportQualityStableFingerprintAndAlias(t *testing.T) {
	r := exportFixture()
	r[0].Source = "/repo/api/spec.yaml"
	var a, b, c bytes.Buffer
	opts := FormatOptions{BaseDirectory: "/repo"}
	require.NoError(t, WriteFormat(&a, "gitlab", r, opts))
	require.NoError(t, WriteFormat(&b, "code-climate", r, opts))
	require.Equal(t, a.String(), b.String())
	var before []qualityIssue
	require.NoError(t, json.Unmarshal(a.Bytes(), &before))
	require.Equal(t, "api/spec.yaml", before[0].Location.Path)
	require.Equal(t, 5, before[0].Location.Lines.Begin)
	require.Len(t, before[0].Fingerprint, 64)
	r[0].Range.Start.Line++
	r[0].Range.End.Line++
	require.NoError(t, WriteFormat(&c, "gitlab", r, opts))
	var after []qualityIssue
	require.NoError(t, json.Unmarshal(c.Bytes(), &after))
	require.Equal(t, before[0].Fingerprint, after[0].Fingerprint)
}
func TestReportJUnitThreshold(t *testing.T) {
	r := exportFixture()
	for threshold, want := range map[int]int{-1: 0, 0: 1, 1: 2, 3: 2} {
		var b bytes.Buffer
		require.NoError(t, WriteFormat(&b, "junit", r, FormatOptions{FailSeverity: threshold}))
		var suites TestSuites
		require.NoError(t, xml.Unmarshal(b.Bytes(), &suites))
		require.Equal(t, want, suites.Failures)
	}
}

type failingReportWriter struct{}

func (failingReportWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
func TestReportWriterErrors(t *testing.T) {
	for _, format := range Formats() {
		t.Run(format, func(t *testing.T) {
			require.ErrorContains(t, WriteFormat(failingReportWriter{}, format, exportFixture(), FormatOptions{}), "write failed")
		})
	}
}
func BenchmarkReportFormats(b *testing.B) {
	sample := exportFixture()[0]
	records := make([]Diagnostic, 22000)
	for i := range records {
		records[i] = sample
	}
	for _, format := range []string{"json", "sarif", "gitlab", "html", "text", "markdown", "teamcity", "github-actions"} {
		b.Run(format, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := WriteFormat(&bytes.Buffer{}, format, records, FormatOptions{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestReportMarkdownDoesNotPadEveryRowForLongMessage(t *testing.T) {
	rows := make([]Diagnostic, 1000)
	for i := range rows {
		rows[i] = exportFixture()[0]
		rows[i].Message = "short"
	}
	rows[0].Message = strings.Repeat("x", 8192)
	var b bytes.Buffer
	require.NoError(t, WriteFormat(&b, "markdown", rows, FormatOptions{}))
	require.Less(t, b.Len(), 200000, "a single long message must not expand all rows with padding")
	require.Equal(t, 1002, strings.Count(b.String(), "\n"))
}
