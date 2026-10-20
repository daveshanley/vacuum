package harness

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

const testRules = `rules:
  title-required:
    given: $.info
    severity: error
    then:
      function: defined
      field: title
`
const good = "openapi: 3.0.3\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}\n"
const bad = "openapi: 3.0.3\ninfo:\n  version: 1.0.0\npaths: {}\n"

func fixture(t *testing.T, name, body string, dir string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))
	return path
}
func testSuite(t *testing.T) (*Suite, string) {
	t.Helper()
	dir := t.TempDir()
	fixture(t, "rules.yaml", testRules, dir)
	fixture(t, "good.yaml", good, dir)
	fixture(t, "bad.yaml", bad, dir)
	path := fixture(t, "suite.yaml", "version: 1\nruleset: rules.yaml\ncases: []\n", dir)
	n := 1
	return &Suite{Version: 1, Ruleset: "rules.yaml", Cases: []Case{{ID: "valid", Input: "good.yaml", Expect: Expectations{Diagnostics: []DiagnosticMatch{}, Execution: map[string]ExecutionMatch{"title-required": {Status: "completed", Matches: &Count{Min: &n}, Invocations: &Count{Min: &n}}}}}}}, path
}
func TestRunPassAndDiagnosticMismatch(t *testing.T) {
	s, path := testSuite(t)
	r := Run(s, path, RunOptions{})
	require.Equal(t, 0, r.ExitCode, "%+v", r.Cases)
	require.False(t, r.Manifest.Reproducible)
	s.Cases[0].Input = "bad.yaml"
	r = Run(s, path, RunOptions{})
	require.Equal(t, 1, r.ExitCode)
	require.Equal(t, "unexpected_diagnostic", r.Cases[0].Failures[0].Code)
	s.Cases[0].Expect.Diagnostics = []DiagnosticMatch{{Rule: "title-required", Severity: "error", Path: "$.info", Source: "bad.yaml", Line: 3}}
	r = Run(s, path, RunOptions{})
	require.Equal(t, 0, r.ExitCode, "%+v", r.Cases[0])
}
func TestRunNoMatchesFailsExecutionAssertion(t *testing.T) {
	s, path := testSuite(t)
	fixture(t, "rules.yaml", strings.Replace(testRules, "$.info", "$.missing", 1), filepath.Dir(path))
	r := Run(s, path, RunOptions{})
	require.Equal(t, 1, r.ExitCode)
	require.Len(t, r.Cases[0].Failures, 2)
	require.Empty(t, r.Cases[0].Before.Diagnostics)
}
func TestUnknownRuleAndFunctionAreToolErrors(t *testing.T) {
	s, path := testSuite(t)
	s.Cases[0].Rules = []string{"not-real"}
	r := Run(s, path, RunOptions{})
	require.Equal(t, 2, r.ExitCode)
	s.Cases[0].Rules = nil
	fixture(t, "rules.yaml", strings.Replace(testRules, "defined", "not-real", 1), filepath.Dir(path))
	r = Run(s, path, RunOptions{})
	require.Equal(t, 2, r.ExitCode)
}
func TestBeforeAfterRepair(t *testing.T) {
	s, path := testSuite(t)
	c := &s.Cases[0]
	c.Input = "bad.yaml"
	c.After = "good.yaml"
	c.Expect.Diagnostics = []DiagnosticMatch{{Rule: "title-required"}}
	c.AfterExpect = &Expectations{Diagnostics: []DiagnosticMatch{}}
	c.Change = &ChangeExpectations{NoNew: true, Removed: []DiagnosticMatch{{Rule: "title-required"}}}
	r := Run(s, path, RunOptions{})
	require.Equal(t, 0, r.ExitCode, "%+v", r.Cases[0])
	c.Change.Unchanged = true
	r = Run(s, path, RunOptions{})
	require.Equal(t, 1, r.ExitCode)
}
func TestExactMatchingHandlesOverlapAndDuplicates(t *testing.T) {
	m := "specific"
	a := []Diagnostic{{Rule: "r", Message: m}, {Rule: "r", Message: "other"}}
	require.Empty(t, exactDiagnostics("d", []DiagnosticMatch{{Rule: "r"}, {Rule: "r", Message: &m}}, a))
	require.Len(t, exactDiagnostics("d", []DiagnosticMatch{{Rule: "r"}}, a), 1)
	a[0].PathsTruncated = true
	require.NotEmpty(t, exactDiagnostics("d", []DiagnosticMatch{{Rule: "r", Paths: []string{}}}, a[:1]))
}
func TestSuiteValidation(t *testing.T) {
	for _, input := range []string{
		"version: 2\nruleset: r\ncases: []",
		"version: 1\nruleset: r\ncases: []\nunknown: yes",
		"version: 1\nruleset: r\ncases:\n- id: a\n  input: x\n  expect: {}",
		"version: 1\nruleset: r\ncases:\n- id: a\n  input: x\n  expect: {diagnostics: []}\n  typo: true",
		"version: 1\nruleset: r\ncases:\n- id: a\n  input: x\n  expect: {diagnostics: []}\n---\n{}",
	} {
		_, err := Parse([]byte(input))
		require.Error(t, err, input)
	}
	_, err := Parse([]byte("version: 1\nruleset: r\ncases:\n- id: a\n  input: x\n  expect: {diagnostics: []}"))
	require.NoError(t, err)
}
func TestSuiteRelativePathsAndSelection(t *testing.T) {
	s, path := testSuite(t)
	r := Run(s, path, RunOptions{CaseIDs: []string{"missing"}})
	require.Equal(t, 2, r.ExitCode)
	r = Run(s, path, RunOptions{CaseIDs: []string{"valid"}})
	require.Equal(t, 0, r.ExitCode)
	require.Equal(t, "good.yaml", r.Manifest.Files[0].Path)
}
func TestCancelledRunCannotPass(t *testing.T) {
	s, path := testSuite(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := Run(s, path, RunOptions{Context: ctx})
	require.Equal(t, 2, r.ExitCode)
}
func TestIgnoreEvidence(t *testing.T) {
	s, path := testSuite(t)
	s.Cases[0].Input = "bad.yaml"
	s.IgnoreFile = "ignore.yaml"
	fixture(t, "ignore.yaml", "title-required:\n  - $.info\n", filepath.Dir(path))
	n := 1
	s.Cases[0].Expect.Execution["title-required"] = ExecutionMatch{Status: "completed", Ignored: &Count{Min: &n, Max: &n}}
	r := Run(s, path, RunOptions{})
	require.Equal(t, 0, r.ExitCode, "%+v", r.Cases[0])
	require.Equal(t, 1, r.Cases[0].Before.Evidence[0].Ignored)
}
func TestInheritedRulesetUsesItsOwnDirectory(t *testing.T) {
	s, path := testSuite(t)
	dir := filepath.Dir(path)
	fixture(t, "nested/base.yaml", testRules, dir)
	fixture(t, "rules.yaml", "extends: ./nested/base.yaml\nrules: {}\n", dir)
	r := Run(s, path, RunOptions{})
	require.Equal(t, 0, r.ExitCode, "%+v", r.Cases[0])
	require.Len(t, r.Manifest.Files, 4)
}
func TestJSONSchemaAndAsyncAPIEvidence(t *testing.T) {
	for _, tc := range []struct{ name, format, spec, formats string }{
		{"schema", "json-schema", "$schema: https://json-schema.org/draft/2020-12/schema\ntype: object\n", "[json-schema]"},
		{"asyncapi", "auto", "asyncapi: 3.0.0\ninfo: {title: Test, version: 1.0.0}\nchannels: {}\n", "[asyncapi3]"},
		{"arazzo", "auto", "arazzo: 1.0.1\ninfo: {title: Test, version: 1.0.0}\nsourceDescriptions: []\nworkflows: []\n", "[arazzo]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, path := testSuite(t)
			dir := filepath.Dir(path)
			fixture(t, "good.yaml", tc.spec, dir)
			fixture(t, "rules.yaml", "rules:\n  title-required:\n    given: $\n    formats: "+tc.formats+"\n    severity: error\n    then: {function: truthy}\n", dir)
			s.Cases[0].Format = tc.format
			s.Cases[0].Expect.Diagnostics = nil
			r := Run(s, path, RunOptions{})
			require.Equal(t, 0, r.ExitCode, "%+v", r.Cases[0])
		})
	}
}
func TestAutoFixVerifiesOutputAndIdempotence(t *testing.T) {
	s, path := testSuite(t)
	dir := filepath.Dir(path)
	fixture(t, "rules.yaml", "rules:\n  title-required:\n    given: $.info.title\n    severity: error\n    autoFixFunction: title\n    then: {function: falsy}\n", dir)
	fixture(t, "fixed.yaml", strings.Replace(good, "title: Test", "title: ''", 1), dir)
	c := &s.Cases[0]
	c.Expect = Expectations{Diagnostics: []DiagnosticMatch{{Rule: "title-required"}}}
	c.Fix = &FixExpectations{Output: "fixed.yaml", Expect: Expectations{Diagnostics: []DiagnosticMatch{}}}
	opts := RunOptions{AutoFixFunctions: map[string]model.AutoFixFunction{"title": func(n, doc *yaml.Node, ctx *model.RuleFunctionContext) (*yaml.Node, error) {
		n.Value = ""
		return n, nil
	}}}
	r := Run(s, path, opts)
	require.Equal(t, 0, r.ExitCode, "%+v", r.Cases[0])
	data, err := os.ReadFile(filepath.Join(dir, "good.yaml"))
	require.NoError(t, err)
	require.Equal(t, good, string(data))
	require.False(t, r.Manifest.Reproducible)
}
func TestReportJSONIsStable(t *testing.T) {
	s, path := testSuite(t)
	a, e := json.Marshal(Run(s, path, RunOptions{}))
	require.NoError(t, e)
	b, e := json.Marshal(Run(s, path, RunOptions{}))
	require.NoError(t, e)
	require.Equal(t, string(a), string(b))
}

func TestBundleReplaysAndRejectsChangedInputs(t *testing.T) {
	s, path := testSuite(t)
	b, err := json.Marshal(s)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0600))
	r := RunFile(path, RunOptions{Version: "test"})
	require.Equal(t, 0, r.ExitCode)
	require.True(t, r.Manifest.Reproducible)
	dest := filepath.Join(t.TempDir(), "replay")
	require.NoError(t, WriteBundle(r, path, dest))
	replay := RunFile(filepath.Join(dest, "suite.yaml"), RunOptions{Version: "test"})
	require.Equal(t, 0, replay.ExitCode)
	require.Error(t, WriteBundle(r, path, dest))
	fixture(t, "good.yaml", bad, filepath.Dir(path))
	require.ErrorContains(t, WriteBundle(r, path, filepath.Join(t.TempDir(), "changed")), "changed")
}
func TestInvalidOptionsCannotBecomeExpectedDiagnostic(t *testing.T) {
	s, path := testSuite(t)
	fixture(t, "rules.yaml", strings.Replace(testRules, "      field: title\n", "", 1), filepath.Dir(path))
	s.Cases[0].Expect = Expectations{Diagnostics: []DiagnosticMatch{{Rule: "title-required"}}}
	r := Run(s, path, RunOptions{})
	require.Equal(t, 2, r.ExitCode)
}
