package motor

import (
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestRepairIgnoresFollowAuthoredTags(t *testing.T) {
	for _, hasDescription := range []bool{false, true} {
		source := "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ntags:\n  - name: z\n"
		if hasDescription {
			source += "    description: Zed tag\n"
		}
		source += "  - name: a\n"
		builtins := rulesets.GetAllBuiltInRules()
		e := &RuleSetExecution{Spec: []byte(source), SpecFileName: "api.yaml", SilenceLogs: true, RepairIgnores: model.IgnoredItems{"tag-description": {"$.tags[0]"}}, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"openapi-tags-alphabetical": builtins["openapi-tags-alphabetical"], "tag-description": builtins["tag-description"]}}}
		_, result := RunRepairs(e, nil, nil, true)
		require.Empty(t, result.Errors)
		require.Empty(t, result.RepairWarnings)
		require.Len(t, result.FixedResults, 1)
		require.Len(t, result.Results, 1)
		require.Equal(t, "tag-description", result.Results[0].RuleId)
		require.Contains(t, result.Results[0].Path, "tags[0]")
		result.ReleaseOwnedResources()
	}
}

func TestRepairNodeMustBelongToRoot(t *testing.T) {
	root, err := repair.ParseDocument([]byte("info:\n  title: Test\n"))
	require.NoError(t, err)
	other, err := repair.ParseDocument([]byte("info:\n  title: Test\n"))
	require.NoError(t, err)
	for _, source := range []string{"", "api.yaml"} {
		findings, err := repairFindings(&RuleSetExecution{CanonicalDocument: root, SpecFileName: source}, []model.RuleFunctionResult{{RuleId: "path-params", StartNode: root.Content[0].Content[0], RepairNode: other.Content[0].Content[1]}})
		require.NoError(t, err)
		require.NotEmpty(t, findings[0].Unavailable)
		require.Empty(t, findings[0].Pointer)
	}
}

func TestCopyConfigDoesNotInheritWorkers(t *testing.T) {
	source := reviewFixture()
	source.ruleWaitGroup.Add(1)
	source.activeRuleWorkers.Add(1)
	defer source.ruleWaitGroup.Done()
	next := source.CopyConfig()
	require.Zero(t, next.activeRuleWorkers.Load())
	done := make(chan struct{})
	go func() { next.ruleWaitGroup.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("copy inherited active workers")
	}
	require.NotSame(t, source.RuleSet, next.RuleSet)
}

func TestFixReviewDuplicateTargetsShareProposal(t *testing.T) {
	r, err := NewFixReview(reviewFixture(), nil)
	require.NoError(t, err)
	defer r.Close()
	var index int
	for _, option := range r.Options() {
		if option.Editable {
			index = option.Index
			break
		}
	}
	duplicate := r.choices[index]
	duplicate.Index = len(r.choices)
	finding := r.findings[index]
	finding.Index = duplicate.Index
	r.choices = append(r.choices, duplicate)
	r.findings = append(r.findings, finding)
	r.fields[duplicate.Index] = r.fields[index]
	plan, err := r.planChoices([]FixChoice{{Index: index}, {Index: duplicate.Index}})
	require.NoError(t, err)
	require.Len(t, plan.Targets, 1)
	changed := "Different value"
	_, err = r.planChoices([]FixChoice{{Index: index}, {Index: duplicate.Index, Value: &changed}})
	require.ErrorContains(t, err, "conflicting proposed values")
}

func TestNewRepairNodesCannotInheritIgnores(t *testing.T) {
	original := []byte("tags: [{name: old}]\n")
	candidate := []byte("tags: [{name: new}, {name: old}]\n")
	plan := repair.NewPlan("api.yaml", original)
	plan.Operations = []repair.Operation{{Op: "add", Path: "/tags/0", Value: []byte(`{"name":"new"}`)}}
	changes, err := repair.TraceChanges(original, candidate, plan)
	require.NoError(t, err)
	root, err := repair.ParseDocument(original)
	require.NoError(t, err)
	next, err := repair.ParseDocument(candidate)
	require.NoError(t, err)
	require.Empty(t, originalRepairPath("$.tags[0]", root, next, changes))
	require.Empty(t, originalRepairPath("$['tags'][0]", root, next, changes))
	require.Equal(t, "$.tags[0].description", originalRepairPath("$['tags'][1]['description']", root, next, changes))
	require.Empty(t, originalRepairPath("$.tags[0].description", root, next, changes))
	require.Equal(t, "$.tags[0].description", originalRepairPath("$.tags[1].description", root, next, changes))
}

func TestRepairPlanningPreservesExplicitStrictErrors(t *testing.T) {
	invalid := &model.Rule{Id: "invalid-schema-dialect", Given: "$.info", Severity: "warn", Then: &model.RuleAction{Function: "schema", FunctionOptions: map[string]any{"dialect": "draft99", "schema": map[string]any{"type": "object"}}}}
	for _, strict := range []bool{false, true} {
		e := reviewFixture()
		e.RuleSet.Rules[invalid.Id] = invalid
		options := &ExecutionOptions{ReportFunctionErrors: strict}
		baseline := ApplyRulesToRuleSetWithOptions(e.CopyConfig(), options)
		_, result := RunRepairs(e, options, nil, true)
		require.Len(t, result.Errors, len(baseline.Errors))
		require.NotEmpty(t, result.RepairWarnings)
		baseline.ReleaseOwnedResources()
		result.ReleaseOwnedResources()
	}
}

func TestRepairWithoutBaselineRetainsInputError(t *testing.T) {
	_, result := RunRepairs(nil, nil, nil, true)
	require.Len(t, result.Errors, 1)
	require.Empty(t, result.RepairWarnings)
}
