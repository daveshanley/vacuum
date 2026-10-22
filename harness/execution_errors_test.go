package harness

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestJavaScriptCoreBridgePanicCannotPass(t *testing.T) {
	s, path := testSuite(t)
	dir := filepath.Dir(path)
	s.Functions = "functions"
	fixture(t, "functions/broken.js", `function getSchema(){return {name:"broken"};} function runRule(input){return vacuum_truthy(null, context);}`, dir)
	fixture(t, "rules.yaml", "rules:\n  title-required:\n    given: $\n    severity: error\n    then: {function: broken}\n", dir)
	r := Run(s, path, RunOptions{})
	require.Equal(t, 2, r.ExitCode, "%+v", r.Cases)
	require.Contains(t, r.Cases[0].Errors[0].Message, "core function")
	require.Equal(t, "error", r.Cases[0].Before.Evidence[0].Status)
}

func TestAutoFixCallbackErrorCannotPass(t *testing.T) {
	s, path := testSuite(t)
	fixture(t, "rules.yaml", "rules:\n  title-required:\n    given: $.info.title\n    severity: error\n    autoFixFunction: broken\n    then: {function: falsy}\n", filepath.Dir(path))
	expected := Expectations{Diagnostics: []DiagnosticMatch{{Rule: "title-required"}}}
	s.Cases[0].Expect = expected
	s.Cases[0].Fix = &FixExpectations{Output: "good.yaml", Expect: expected}
	r := Run(s, path, RunOptions{AutoFixFunctions: map[string]model.AutoFixFunction{"broken": func(*yaml.Node, *yaml.Node, *model.RuleFunctionContext) (*yaml.Node, error) {
		return nil, fmt.Errorf("repair failed")
	}}})
	require.Equal(t, 2, r.ExitCode, "%+v", r.Cases)
	require.Contains(t, r.Cases[0].Errors[0].Message, "repair failed")
	require.Equal(t, "error", r.Cases[0].Fixed.Evidence[0].Status)
}

func TestAutoFixPanicEvidenceCannotClaimCompletion(t *testing.T) {
	s, path := testSuite(t)
	fixture(t, "rules.yaml", "rules:\n  title-required:\n    given: $.info.title\n    severity: error\n    autoFixFunction: broken\n    then: {function: falsy}\n", filepath.Dir(path))
	expected := Expectations{Diagnostics: []DiagnosticMatch{{Rule: "title-required"}}}
	s.Cases[0].Expect = expected
	s.Cases[0].Fix = &FixExpectations{Output: "good.yaml", Expect: expected}
	r := Run(s, path, RunOptions{AutoFixFunctions: map[string]model.AutoFixFunction{"broken": func(*yaml.Node, *yaml.Node, *model.RuleFunctionContext) (*yaml.Node, error) {
		panic("repair failed")
	}}})
	require.Equal(t, 2, r.ExitCode, "%+v", r.Cases)
	require.Contains(t, r.Cases[0].Errors[0].Message, "repair failed")
	require.Equal(t, "error", r.Cases[0].Fixed.Evidence[0].Status)
}

func TestEmptyRuleWorkCannotPass(t *testing.T) {
	for _, body := range []string{"given: $\n    then: []", "given: []\n    then: {function: truthy}"} {
		t.Run(body, func(t *testing.T) {
			s, path := testSuite(t)
			fixture(t, "rules.yaml", "rules:\n  title-required:\n    severity: error\n    "+body+"\n", filepath.Dir(path))
			s.Cases[0].Expect = Expectations{Diagnostics: []DiagnosticMatch{}}
			require.Equal(t, 2, Run(s, path, RunOptions{}).ExitCode)
		})
	}
}

func TestJavaScriptMetadataFailuresCannotPass(t *testing.T) {
	for _, script := range []string{
		`function getSchema(){while(true){}} function runRule(){return [];}`,
		`while(true){} function getSchema(){return {name:"broken"};} function runRule(){return [];}`,
		`function getSchema(){throw new Error("bad metadata");} function runRule(){return [];}`,
	} {
		t.Run(script, func(t *testing.T) {
			s, path := testSuite(t)
			s.Functions = "functions"
			s.Options.TimeoutMS = 10
			fixture(t, "functions/broken.js", script, filepath.Dir(path))
			fixture(t, "rules.yaml", "rules:\n  title-required:\n    given: $\n    then: {function: broken}\n", filepath.Dir(path))
			r := Run(s, path, RunOptions{})
			require.Equal(t, 2, r.ExitCode)
			require.NotEmpty(t, r.Errors)
		})
	}
}

func TestNonIdempotentFixRetainsSecondExecutionEvidence(t *testing.T) {
	s, path := testSuite(t)
	dir := filepath.Dir(path)
	fixture(t, "rules.yaml", "rules:\n  title-required:\n    given: $.info.title\n    severity: error\n    autoFixFunction: append\n    then: {function: falsy}\n", dir)
	fixture(t, "fixed.yaml", strings.Replace(good, "title: Test", "title: Test!", 1), dir)
	expected := Expectations{Diagnostics: []DiagnosticMatch{{Rule: "title-required"}}}
	s.Cases[0].Expect = expected
	s.Cases[0].Fix = &FixExpectations{Output: "fixed.yaml", Expect: expected}
	r := Run(s, path, RunOptions{AutoFixFunctions: map[string]model.AutoFixFunction{"append": func(node, doc *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		node.Value += "!"
		return node, nil
	}}})
	require.Equal(t, 1, r.ExitCode, "%+v", r.Cases)
	require.Equal(t, "fix.idempotent", r.Cases[0].Failures[0].Assertion)
	require.NotNil(t, r.Cases[0].SecondFix)
	require.Equal(t, "completed", r.Cases[0].SecondFix.Evidence[0].Status)
	require.Equal(t, 1, r.Cases[0].SecondFix.Evidence[0].Fixed)
}
