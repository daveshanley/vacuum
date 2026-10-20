package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestStructuredErrorsIdentifyFailureBoundary(t *testing.T) {
	t.Run("missing input", func(t *testing.T) {
		s, path := testSuite(t)
		s.Cases[0].Input = "absent.yaml"
		r := Run(s, path, RunOptions{})
		require.Equal(t, 2, r.ExitCode)
		e := r.Cases[0].Errors[0]
		require.Equal(t, ErrorInput, e.Code)
		require.Equal(t, PhaseBefore, e.Phase)
		require.Equal(t, "valid", e.CaseID)
		require.Equal(t, "absent.yaml", e.Source)
	})
	t.Run("missing extended ruleset", func(t *testing.T) {
		s, path := testSuite(t)
		fixture(t, "rules.yaml", "extends: ./missing/base.yaml\nrules: {}\n", filepath.Dir(path))
		e := Run(s, path, RunOptions{}).Cases[0].Errors[0]
		require.Equal(t, ErrorRuleset, e.Code)
		require.Equal(t, "missing/base.yaml", e.Source)
	})
	t.Run("unknown function", func(t *testing.T) {
		s, path := testSuite(t)
		fixture(t, "rules.yaml", "rules:\n  title-required:\n    given: $\n    then: {function: absent}\n", filepath.Dir(path))
		e := Run(s, path, RunOptions{}).Cases[0].Errors[0]
		require.Equal(t, ErrorUnknownFunction, e.Code)
		require.Equal(t, "title-required", e.RuleID)
		require.Equal(t, "absent", e.Function)
		require.Equal(t, "rules.yaml", e.Source)
	})
	t.Run("repair function", func(t *testing.T) {
		s, path := testSuite(t)
		fixture(t, "rules.yaml", "rules:\n  title-required:\n    given: $.info.title\n    autoFixFunction: repair\n    then: {function: falsy}\n", filepath.Dir(path))
		s.Cases[0].Expect = Expectations{Diagnostics: []DiagnosticMatch{{Rule: "title-required"}}}
		s.Cases[0].Fix = &FixExpectations{Output: "good.yaml", Expect: s.Cases[0].Expect}
		report := Run(s, path, RunOptions{AutoFixFunctions: map[string]model.AutoFixFunction{"repair": func(*yaml.Node, *yaml.Node, *model.RuleFunctionContext) (*yaml.Node, error) {
			return nil, fmt.Errorf("message can change")
		}}})
		require.Len(t, report.Cases[0].Errors, 1)
		e := report.Cases[0].Errors[0]
		require.Equal(t, ErrorAutoFix, e.Code)
		require.Equal(t, PhaseFixed, e.Phase)
		require.Equal(t, "repair", e.Function)
		require.Equal(t, "title-required", e.RuleID)
		data, err := json.Marshal(report)
		require.NoError(t, err)
		var decoded Report
		require.NoError(t, json.Unmarshal(data, &decoded))
		require.Equal(t, e, decoded.Cases[0].Errors[0])
	})
	t.Run("invalid native plugin", func(t *testing.T) {
		s, path := testSuite(t)
		s.Functions = "functions"
		fixture(t, "functions/broken.so", "not a plugin", filepath.Dir(path))
		report := Run(s, path, RunOptions{})
		require.Equal(t, 2, report.ExitCode)
		require.Equal(t, ErrorFunctionLoad, report.Errors[0].Code)
		require.Equal(t, "functions/broken.so", report.Errors[0].Source)
	})
	t.Run("metadata timeout", func(t *testing.T) {
		s, path := testSuite(t)
		s.Functions = "functions"
		s.Options.TimeoutMS = 10
		fixture(t, "functions/broken.js", `function getSchema(){while(true){}} function runRule(){return [];}`, filepath.Dir(path))
		report := Run(s, path, RunOptions{})
		require.Equal(t, 2, report.ExitCode)
		require.Len(t, report.Errors, 1)
		e := report.Errors[0]
		require.Equal(t, ErrorDeadline, e.Code)
		require.Equal(t, PhaseLoad, e.Phase)
		require.Equal(t, "broken", e.Function)
		require.Equal(t, "functions/broken.js", e.Source)
	})
}

func TestTypedExecutionErrorsPreserveCauseIdentity(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code ErrorCode
		rule string
	}{
		{&motor.RuleTimeoutError{RuleId: "r"}, ErrorRuleTimeout, "r"},
		{&motor.RuleLookupError{RuleId: "r", Err: fmt.Errorf("lookup")}, ErrorLookup, "r"},
		{fmt.Errorf("wrapped: %w", context.Canceled), ErrorCancelled, ""},
		{&motor.RuleExecutionError{Kind: motor.RuleErrorFunction, RuleID: "r", Err: context.DeadlineExceeded}, ErrorDeadline, "r"},
	} {
		e := NewExecutionError(ErrorExecution, PhaseAfter, tc.err)
		require.Equal(t, tc.code, e.Code)
		require.Equal(t, tc.rule, e.RuleID)
	}
}
