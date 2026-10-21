package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairEvidenceAndOutputAcrossFormats(t *testing.T) {
	for _, spec := range []string{good,
		"asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {}\n",
		"arazzo: 1.0.1\ninfo: {title: Test, version: '1'}\nsourceDescriptions: []\nworkflows: []\n"} {
		t.Run(strings.Split(spec, ":")[0], func(t *testing.T) {
			s, path := testSuite(t)
			dir := filepath.Dir(path)
			fixture(t, "good.yaml", spec, dir)
			fixture(t, "fixed.yaml", strings.Replace(spec, "title: Test", "title: ''", 1), dir)
			formats := ""
			if strings.HasPrefix(spec, "asyncapi:") {
				formats = "    formats: [asyncapi3]\n"
			}
			fixture(t, "rules.yaml", "rules:\n  title-required:\n"+formats+"    given: $.info.title\n    severity: error\n    autoFixFunction: title\n    then: {function: falsy}\n", dir)
			one := 1
			s.Cases[0].Expect = Expectations{Diagnostics: []DiagnosticMatch{{Rule: "title-required"}}}
			s.Cases[0].Fix = &FixExpectations{Output: "fixed.yaml", Expect: Expectations{Diagnostics: []DiagnosticMatch{}, Execution: map[string]ExecutionMatch{"title-required": {Status: "completed", Fixed: &Count{Min: &one, Max: &one}}}}}
			opts := RunOptions{AutoFixFunctions: map[string]model.AutoFixFunction{"title": func(n, doc *yaml.Node, ctx *model.RuleFunctionContext) (*yaml.Node, error) {
				n.Value = ""
				return n, nil
			}}}
			report := Run(s, path, opts)
			require.Equal(t, 0, report.ExitCode, fmt.Sprintf("%+v", report.Cases))
			require.Equal(t, 1, report.Cases[0].Fixed.Evidence[0].Fixed)
			require.Equal(t, 0, report.Cases[0].After.Evidence[0].Fixed)
			require.Equal(t, 0, report.Cases[0].SecondFix.Evidence[0].Fixed)
			s.Cases[0].Fix.Expect.Execution = map[string]ExecutionMatch{"typo": {Status: "completed"}}
			report = Run(s, path, opts)
			require.Equal(t, 2, report.ExitCode)
			require.Contains(t, report.Cases[0].Errors[0].Message, "unselected rule")
		})
	}
}

func TestRulesCapturedOnceAndReportedOnce(t *testing.T) {
	s, path := testSuite(t)
	r := &runner{suite: s, base: filepath.Dir(path), ctx: context.Background(), files: map[string]string{}, limitations: map[string]bool{}}
	first, err := r.loadRules(s.Cases[0])
	require.NoError(t, err)
	fixture(t, "rules.yaml", "not a valid ruleset", filepath.Dir(path))
	second, err := r.loadRules(s.Cases[0])
	require.NoError(t, err)
	require.Equal(t, first.Rules, second.Rules)
	first.Rules["title-required"].Message = "changed"
	require.NotEqual(t, first.Rules["title-required"].Message, second.Rules["title-required"].Message)
	fixture(t, "rules.yaml", testRules, filepath.Dir(path))
	report := Run(s, path, RunOptions{})
	require.Equal(t, 0, report.ExitCode)
	data, err := json.Marshal(report)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(data), `"rules":`))
	require.NotEmpty(t, report.Cases[0].Rules)
}

func TestParentDependencyIsNotReproducible(t *testing.T) {
	s, path := testSuite(t)
	parent := filepath.Dir(path)
	suitePath := fixture(t, "nested/suite.yaml", "", parent)
	s.Ruleset = "../rules.yaml"
	s.Cases[0].Input = "../good.yaml"
	report := Run(s, suitePath, RunOptions{})
	require.Equal(t, 0, report.ExitCode)
	require.False(t, report.Manifest.Reproducible)
	require.Contains(t, report.Manifest.Limitations, "dependency escapes suite directory")
}

type optionMutationProbe struct{ seen []any }

func (p *optionMutationProbe) GetCategory() string { return model.FunctionCategoryCore }
func (p *optionMutationProbe) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "probe", Properties: []model.RuleFunctionProperty{{Name: "flag"}}}
}
func (p *optionMutationProbe) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	options := ctx.Options.(map[string]any)
	p.seen = append(p.seen, options["flag"])
	options["flag"] = "mutated"
	return nil
}
func TestCachedRulesIsolateCallbackOptions(t *testing.T) {
	s, path := testSuite(t)
	fixture(t, "rules.yaml", "rules:\n  probe:\n    given: $\n    then:\n      function: probe\n      functionOptions: {flag: original}\n", filepath.Dir(path))
	s.Cases = []Case{{ID: "first", Input: "good.yaml", Expect: Expectations{Diagnostics: []DiagnosticMatch{}}}, {ID: "second", Input: "good.yaml", Expect: Expectations{Diagnostics: []DiagnosticMatch{}}}}
	probe := &optionMutationProbe{}
	report := Run(s, path, RunOptions{CustomFunctions: map[string]model.RuleFunction{"probe": probe}})
	require.Equal(t, 0, report.ExitCode)
	require.Equal(t, []any{"original", "original"}, probe.seen)
	for _, c := range report.Cases {
		data, err := json.Marshal(c.Rules)
		require.NoError(t, err)
		require.Contains(t, string(data), "original")
		require.NotContains(t, string(data), "mutated")
	}
}
