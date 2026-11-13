// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"testing"

	"github.com/daveshanley/vacuum/motor"

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
		e := &motor.RuleSetExecution{Spec: []byte(source), SpecFileName: "api.yaml", SilenceLogs: true, RepairIgnores: model.IgnoredItems{"tag-description": {"$.tags[0]"}}, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"openapi-tags-alphabetical": builtins["openapi-tags-alphabetical"], "tag-description": builtins["tag-description"]}}}
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
		findings, err := repairFindings(&motor.RuleSetExecution{CanonicalDocument: root, SpecFileName: source}, []model.RuleFunctionResult{{RuleId: "path-params", StartNode: root.Content[0].Content[0], RepairNode: other.Content[0].Content[1]}})
		require.NoError(t, err)
		require.NotEmpty(t, findings[0].Unavailable)
		require.Empty(t, findings[0].Pointer)
	}
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
		options := &motor.ExecutionOptions{ReportFunctionErrors: strict}
		baseline := motor.ApplyRulesToRuleSetWithOptions(e.CopyConfig(), options)
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

func TestReviewShowsOneChoiceForReferencedParameter(t *testing.T) {
	const source = `openapi: 3.1.0
info: {title: Ports, version: '1'}
components:
  parameters:
    port_id: {name: id, in: path, required: true, schema: {type: integer}}
paths:
  /ports/{id}:
    parameters: [{$ref: '#/components/parameters/port_id'}]
    get:
      responses: {200: {description: Good}}
`
	for _, id := range []string{"oas3-parameter-description", "operation-success-response"} {
		t.Run(id, func(t *testing.T) {
			e := &motor.RuleSetExecution{Spec: []byte(source), SpecFileName: "ports.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rulesets.GetAllBuiltInRules()[id]}}}
			review, err := NewFixReview(e, nil)
			require.NoError(t, err)
			defer review.Close()
			choices := review.Options()
			require.Len(t, choices, 1)
			require.True(t, choices[0].Available)
			p, err := review.PreviewOutput([]FixChoice{{Index: choices[0].Index}}, FixOutputOptions{Overlay: true})
			require.NoError(t, err)
			defer p.Close()
			require.Equal(t, 1, p.Fixed)
			require.Empty(t, p.Results)
			require.NotEmpty(t, p.Overlay)
		})
	}
}

func TestFixReviewVerifiesOnlyTheSelectedField(t *testing.T) {
	id := "info-description"
	rule := &model.Rule{Id: id, Given: "$.info", Message: "Add metadata", Severity: "warn", Then: []model.RuleAction{{Function: "truthy", Field: "summary"}, {Function: "truthy", Field: "description"}}}
	e := &motor.RuleSetExecution{Spec: []byte(`{"openapi":"3.1.0","info":{"title":"Pets","version":"1"},"paths":{}}`), SpecFileName: "pets.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rule}}}
	plain := motor.ApplyRulesToRuleSet(e.CopyConfig())
	defer plain.ReleaseOwnedResources()
	require.Len(t, plain.Results, 1)
	review, err := NewFixReview(e, nil)
	require.NoError(t, err)
	defer review.Close()
	require.Len(t, review.Options(), 2)
	var selected []FixChoice
	for _, option := range review.Options() {
		require.Equal(t, option.Result.DiagnosticCode == "description-missing", option.Available)
		if option.Available {
			selected = append(selected, FixChoice{Index: option.Index})
		}
	}
	require.Len(t, selected, 1)
	p, err := review.Preview(selected)
	require.NoError(t, err)
	defer p.Close()
	require.Equal(t, 1, p.Fixed)
	require.Len(t, p.Results, 1)
	require.Equal(t, "summary-missing", p.Results[0].DiagnosticCode)
	require.Len(t, p.fixed, 1)
	require.Equal(t, "description-missing", p.fixed[0].DiagnosticCode)
	baseline := review.TakeBaselineResult()
	defer baseline.ReleaseOwnedResources()
	require.Len(t, baseline.Results, len(plain.Results))
	require.Equal(t, plain.Results[0].DiagnosticCode, baseline.Results[0].DiagnosticCode)
	require.Equal(t, plain.Results[0].Path, baseline.Results[0].Path)
	require.Equal(t, plain.Results[0].Paths, baseline.Results[0].Paths)
}
