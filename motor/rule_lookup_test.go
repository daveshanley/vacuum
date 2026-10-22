package motor

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

type lookupCallRecorder struct{ called bool }

func (r *lookupCallRecorder) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "lookupRecorder"}
}
func (r *lookupCallRecorder) GetCategory() string { return model.CategoryValidation }
func (r *lookupCallRecorder) RunRule(_ []*yaml.Node, _ model.RuleFunctionContext) []model.RuleFunctionResult {
	r.called = true
	return nil
}

func TestIssue955LookupFailureDoesNotUseUnresolvedNodes(t *testing.T) {
	var unresolved yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("parameters: [{'$ref': '#/components/parameters/Shared'}]"), &unresolved))
	recorder := &lookupCallRecorder{}
	var results, ignored, fixed []model.RuleFunctionResult
	var errs []error
	// A nil resolved root causes a recovered query panic. This exercises the
	// same lookup failure path as a timeout without depending on scheduler timing.
	runRule(ruleContext{
		rule:     &model.Rule{Id: "require-examples", Resolved: true, Given: "$.parameters[*]", Then: model.RuleAction{Function: "lookupRecorder"}},
		specNode: nil, specNodeUnresolved: &unresolved,
		builtinFunctions: functions.MapBuiltinFunctions(),
		customFunctions:  map[string]model.RuleFunction{"lookupRecorder": recorder},
		ruleResults:      &results, ignoredResults: &ignored, fixedResults: &fixed, errors: &errs,
		skipDocumentCheck: true, nodeLookupTimeout: time.Second,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	require.False(t, recorder.called, "a failed resolved query must not evaluate raw reference nodes")
	require.Empty(t, results)
	require.Len(t, errs, 1)
	var lookup *RuleLookupError
	require.True(t, errors.As(errs[0], &lookup))
	require.Equal(t, "require-examples", lookup.RuleId)
	require.Contains(t, lookup.Error(), "JSONPath query panic")
}
