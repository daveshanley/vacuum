package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	vacuumreport "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/require"
)

func exportCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := GetRootCommand()
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(append(args, "--no-update-check"))
	var err error
	noise, _ := captureOSStreams(t, func() { err = root.Execute() })
	require.Empty(t, noise, "report output must not contain terminal chrome")
	return out.String(), errOut.String(), err
}
func exportInputs(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.yaml")
	rules := filepath.Join(dir, "rules.yaml")
	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"), 0600))
	require.NoError(t, os.WriteFile(rules, []byte("rules:\n  require-contact:\n    given: $.info\n    severity: warn\n    then: {field: contact, function: truthy}\n"), 0600))
	return spec, rules
}
func TestLintOutputAllFormats(t *testing.T) {
	spec, rules := exportInputs(t)
	for _, format := range vacuumreport.Formats() {
		t.Run(format, func(t *testing.T) {
			out, errOut, err := exportCommand(t, "lint", spec, "-r", rules, "--format", format, "--fail-severity", "none")
			require.NoError(t, err)
			require.Empty(t, errOut)
			require.Contains(t, out, "require-contact")
			if format == "json" {
				var rows []vacuumreport.Diagnostic
				require.NoError(t, json.Unmarshal([]byte(out), &rows))
				require.Len(t, rows, 1)
				require.Equal(t, []any{"info", "contact"}, rows[0].Path)
			}
		})
	}
}
func TestLintOutputMultipleDestinations(t *testing.T) {
	spec, rules := exportInputs(t)
	dest := filepath.Join(t.TempDir(), "report.sarif")
	out, _, err := exportCommand(t, "lint", spec, "-r", rules, "--format", "json,sarif", "--output", "sarif="+dest, "--fail-severity", "none")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(out)))
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Contains(t, string(data), `"version": "2.1.0"`)
}
func TestLintOutputMultipleFiles(t *testing.T) {
	spec, rules := exportInputs(t)
	other := filepath.Join(t.TempDir(), "other.yaml")
	data, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(other, data, 0600))
	out, _, err := exportCommand(t, "lint", spec, other, "-r", rules, "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	var rows []vacuumreport.Diagnostic
	require.NoError(t, json.Unmarshal([]byte(out), &rows))
	require.Len(t, rows, 2)
	require.NotEqual(t, rows[0].Source, rows[1].Source)
}
func TestLintOutputValidationAndWriteFailures(t *testing.T) {
	spec, rules := exportInputs(t)
	for _, args := range [][]string{{"--format", "bad"}, {"--output", "x"}, {"--format", "json,sarif"}, {"--format", "json", "--silent"}, {"--format", "json", "--output", spec}, {"--format", "json", "--output", filepath.Join(t.TempDir(), "missing", "x.json")}} {
		out, errOut, err := exportCommand(t, append([]string{"lint", spec, "-r", rules}, args...)...)
		require.Error(t, err)
		require.Empty(t, out)
		require.NotEmpty(t, errOut)
	}
}
func TestLintOutputMissingInputAndViolation(t *testing.T) {
	out, errOut, err := exportCommand(t, "lint", "missing-formatter-test.yaml", "--format", "json")
	require.Error(t, err)
	require.Empty(t, out)
	require.NotEmpty(t, errOut)
	spec, rules := exportInputs(t)
	out, _, err = exportCommand(t, "lint", spec, "-r", rules, "--format", "json", "--fail-severity", "warn")
	require.Error(t, err)
	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, ExitCodeViolations, exitErr.Code)
	require.True(t, json.Valid([]byte(out)))
}
func TestExportDiagnosticLiteralPathsAndOrigin(t *testing.T) {
	r := &model.RuleFunctionResult{RuleId: "test", RuleSeverity: "hint", Path: `$.components.schemas['a.b'][0]['x/y']`, Message: "test", Range: reports.Range{Start: reports.RangeItem{Line: 3, Char: 2}, End: reports.RangeItem{Line: 3, Char: 5}}, Origin: &index.NodeOrigin{AbsoluteLocation: "external.yaml"}}
	d := exportDiagnostic(r, "root.yaml", map[*yaml.Node]cachedNodeSpan{})
	require.Equal(t, []any{"components", "schemas", "a.b", int64(0), "x/y"}, d.Path)
	require.Equal(t, "external.yaml", d.Source)
	require.Equal(t, 3, d.Severity)
	require.Equal(t, 2, d.Range.Start.Line)
	require.Equal(t, 1, d.Range.Start.Char)
	require.Equal(t, 4, d.Range.End.Char)
	require.True(t, strings.Contains(d.Source, "external"))
}

func TestLintOutputCategoryOutcome(t *testing.T) {
	spec, rules := exportInputs(t)
	data, err := os.ReadFile(rules)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(rules, []byte(strings.ReplaceAll(string(data), "severity: warn", "severity: error")), 0600))
	second := filepath.Join(t.TempDir(), "second.yaml")
	data, err = os.ReadFile(spec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(second, data, 0600))
	for _, inputs := range [][]string{{spec}, {spec, second}} {
		args := append([]string{"lint"}, inputs...)
		args = append(args, "-r", rules, "--format", "json", "--category", "descriptions")
		out, _, err := exportCommand(t, args...)
		require.NoError(t, err)
		require.JSONEq(t, "[]", out)
	}
}
func TestExportOriginCoordinates(t *testing.T) {
	r := &model.RuleFunctionResult{RuleId: "rule", Path: "$", StartNode: &yaml.Node{Line: 40, Column: 9}, Origin: &index.NodeOrigin{AbsoluteLocation: "external.yaml", Line: 7, Column: 2}}
	d := exportDiagnostic(r, "root.yaml", map[*yaml.Node]cachedNodeSpan{})
	require.Equal(t, 6, d.Range.Start.Line)
	require.Equal(t, 1, d.Range.Start.Char)
	require.Equal(t, d.Range.Start, d.Range.End)
}
func TestAnnotationUnicodeScalarEnd(t *testing.T) {
	require.Equal(t, 7, scalarNodeEnd(&yaml.Node{Line: 1, Column: 3, Value: "café"}).Char)
}

func TestLintOutputMalformedSpecDetails(t *testing.T) {
	spec, rules := exportInputs(t)
	require.NoError(t, os.WriteFile(spec, []byte("openapi: [broken\n"), 0600))
	out, errOut, err := exportCommand(t, "lint", spec, "-r", rules, "--format", "json")
	require.Error(t, err)
	require.Empty(t, out)
	require.Contains(t, errOut, "Unable to process spec")
	require.Contains(t, errOut, "unable to parse specification")
}
func TestLintOutputAliasedDestinations(t *testing.T) {
	spec, rules := exportInputs(t)
	dir := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	require.NoError(t, os.Symlink(dir, alias))
	out, errOut, err := exportCommand(t, "lint", spec, "-r", rules, "--format", "json,sarif", "--output", "json="+filepath.Join(dir, "result"), "--output", "sarif="+filepath.Join(alias, "result"))
	require.Error(t, err)
	require.Empty(t, out)
	require.Contains(t, errOut, "distinct")
}
func TestReportAtomicFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report")
	require.NoError(t, os.WriteFile(path, []byte("previous"), 0600))
	require.Error(t, writeReportFile(path, func(w io.Writer) error {
		_, err := io.WriteString(w, "partial")
		if err != nil {
			return err
		}
		return errors.New("render failed")
	}))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "previous", string(data))
}
