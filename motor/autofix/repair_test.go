// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/go-yaml"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestBuiltinRepairVerified(t *testing.T) {
	cases := []struct{ name, rule, spec string }{
		{"enum", "duplicated-entry-in-enum", `openapi: 3.1.0
info: {title: Test, version: '1'}
paths: {}
components:
  schemas:
    Pet: {type: string, enum: [cat, cat, dog]}
`},
		{"sort", "openapi-tags-alphabetical", `openapi: 3.1.0
info: {title: Test, version: '1'}
paths: {}
tags: [{name: z}, {name: a}]
`},
		{"required", "path-params", `openapi: 3.1.0
info: {title: Test, version: '1'}
paths:
  /pets/{id}:
    get:
      parameters: [{name: id, in: path, required: false, schema: {type: string}}]
      responses: {'200': {description: ok}}
`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			builtins := rulesets.GetAllBuiltInRules()
			e := &motor.RuleSetExecution{Spec: []byte(tc.spec), SpecFileName: "test.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{tc.rule: builtins[tc.rule]}}}
			plan, baseline, err := PlanFixes(e, nil)
			require.NoError(t, err)
			defer baseline.ReleaseOwnedResources()
			require.NotEmpty(t, baseline.Results)
			require.NotEmpty(t, plan.Targets, "plan: %+v baseline: %+v", plan, baseline.Results)
			_, result := RunRepairs(e, nil, nil, true)
			require.Empty(t, result.Errors)
			require.Empty(t, result.RepairWarnings)
			defer result.ReleaseOwnedResources()
			require.NotEmpty(t, result.FixedResults)
			require.Empty(t, result.Results)
			e.Spec = result.ModifiedSpec
			next, lint, err := PlanFixes(e, nil)
			require.NoError(t, err)
			defer lint.ReleaseOwnedResources()
			require.Empty(t, next.Operations)
			require.Empty(t, next.KeyTags)
		})
	}
}

func TestRepairKeepsUnchangedDiagnosticAcrossFormatting(t *testing.T) {
	spec := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1","description":"Shared description"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","description":"Shared description","enum":["a","a"]}}}}`)
	builtins := rulesets.GetAllBuiltInRules()
	e := &motor.RuleSetExecution{Spec: spec, SpecFileName: "test.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"duplicated-entry-in-enum": builtins["duplicated-entry-in-enum"], "description-duplication": builtins["description-duplication"]}}}
	_, result := RunRepairs(e, nil, nil, true)
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Len(t, result.FixedResults, 1)
	require.Len(t, result.Results, 1)
	require.Equal(t, "description-duplication", result.Results[0].RuleId)
}

func TestPluginRepairsAreVerifiedAndComposeWithBuiltins(t *testing.T) {
	for _, works := range []bool{false, true} {
		t.Run(fmt.Sprint(works), func(t *testing.T) {
			rule := &model.Rule{Id: "plugin", Given: "$.info.description", Severity: "warn", AutoFixFunction: "fix", Then: &model.RuleAction{Function: "truthy"}}
			e := &motor.RuleSetExecution{Spec: []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1","description":""},"paths":{},"components":{"schemas":{"Pet":{"enum":["a","a"]}}}}`), SpecFileName: "test.json", SilenceLogs: true, ApplyAutoFixes: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"plugin": rule, "duplicated-entry-in-enum": rulesets.GetAllBuiltInRules()["duplicated-entry-in-enum"]}}, AutoFixFunctions: map[string]model.AutoFixFunction{"fix": func(n, doc *yaml.Node, _ *model.RuleFunctionContext) (*yaml.Node, error) {
				if works {
					n.Value = "Useful description"
				} else {
					doc.Content[0].Content = append(doc.Content[0].Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x-unrelated"}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "changed"})
				}
				return n, nil
			}}}
			_, result := RunRepairs(e, nil, nil, true)
			defer result.ReleaseOwnedResources()
			if works {
				require.Empty(t, result.Errors)
				require.Len(t, result.FixedResults, 2)
				require.NotEmpty(t, result.ModifiedSpec)
			} else {
				require.Empty(t, result.Errors)
				require.NotEmpty(t, result.RepairWarnings)
				require.Len(t, result.Results, 2)
				require.Empty(t, result.FixedResults)
				require.Empty(t, result.ModifiedSpec)
			}
		})
	}
}

func TestRepairUsesFrozenRemoteDependency(t *testing.T) {
	var calls atomic.Int32
	var limit atomic.Int32
	limit.Store(100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= limit.Load() {
			fmt.Fprint(w, `{"type":"string"}`)
		} else {
			fmt.Fprint(w, `{"type":"invalid"}`)
		}
	}))
	defer server.Close()
	spec := []byte(fmt.Sprintf(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"$ref":%q}}}}`, server.URL+"/pet.json"))
	e := &motor.RuleSetExecution{Spec: spec, SpecFileName: filepath.Join(t.TempDir(), "api.json"), AllowLookup: true, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-contact": rulesets.GetAllBuiltInRules()["info-contact"]}}}
	policy, err := repair.ParsePolicy([]byte(fmt.Sprintf("version: 1\npolicies:\n  - id: contact\n    rule: info-contact\n    diagnostic: contact-missing\n    scope: {source: %q, pointer: /info}\n    value: {name: Team}\n", e.SpecFileName)), "", nil)
	require.NoError(t, err)
	_, preview := RunRepairs(e, nil, policy, false)
	preview.ReleaseOwnedResources()
	baselineCalls := calls.Swap(0)
	require.Positive(t, baselineCalls)
	limit.Store(baselineCalls)
	plan, result := RunRepairs(e, nil, policy, true)
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors, "plan: %+v", plan)
	require.Len(t, plan.Sources, 1)
	require.Len(t, result.FixedResults, 1)
	require.Equal(t, baselineCalls, calls.Load(), "verification must not refetch")
}

func TestAsyncAPIRepairProviders(t *testing.T) {
	cases := []struct {
		rule, spec string
		policy     bool
	}{
		{"asyncapi-info-contact", "asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {}\noperations: {}\n", true},
		{"asyncapi-3-tags-alphabetical", "asyncapi: 3.0.0\ninfo: {title: Test, version: '1', tags: [{name: z}, {name: a}]}\nchannels: {}\noperations: {}\n", false},
		{"asyncapi-3-tags-uniqueness", "asyncapi: 3.0.0\ninfo: {title: Test, version: '1', tags: [{name: a}, {name: a}]}\nchannels: {}\noperations: {}\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.rule, func(t *testing.T) {
			e := &motor.RuleSetExecution{Spec: []byte(tc.spec), SpecFileName: "/tmp/async-repair.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{tc.rule: rulesets.GetAllAsyncAPIRules()[tc.rule]}}}
			var policy *repair.Policy
			if tc.policy {
				var err error
				policy, err = repair.ParsePolicy([]byte("version: 1\npolicies:\n  - id: contact\n    rule: asyncapi-info-contact\n    diagnostic: contact-missing\n    scope: {source: /tmp/async-repair.yaml, pointer: /info}\n    value: {name: Team}\n"), "/tmp", nil)
				require.NoError(t, err)
			}
			plan, result := RunRepairs(e, nil, policy, true)
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors, "%+v", plan)
			require.NotEmpty(t, result.FixedResults, "%+v", plan)
			require.Empty(t, result.Results)
		})
	}
}

func TestRepairPreservesDiagnosticsAfterSequenceRemoval(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info: {title: Test, version: '1'}
paths: {}
tags:
  - {name: pets, description: Pets}
  - {name: pets, description: Pets}
  - {name: zoo}
x-sentinel: keep
`)
	all := rulesets.GetAllBuiltInRules()
	all["line-reference"] = &model.Rule{Id: "line-reference", Given: "$['x-sentinel']", Severity: "warn", Then: &model.RuleAction{Function: "line-reference"}}
	e := &motor.RuleSetExecution{Spec: source, SpecFileName: "test.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: all}, CustomFunctions: map[string]model.RuleFunction{"line-reference": repairLineFinding{}}}
	_, result := RunRepairs(e, nil, nil, true)
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Empty(t, result.RepairWarnings)
	require.NotEmpty(t, result.FixedResults)
	require.NotEmpty(t, result.ModifiedSpec)
	foundTag, foundLine := false, false
	for _, r := range result.Results {
		if r.RuleId == "tag-description" {
			foundTag = true
			require.Contains(t, r.Path, "[1]")
		}
		if r.RuleId == "line-reference" {
			foundLine = true
			require.Contains(t, r.Message, fmt.Sprintf("line `%d`", r.StartNode.Line))
		}
	}
	require.True(t, foundTag)
	require.True(t, foundLine)
}

type repairLineFinding struct{}

func (repairLineFinding) GetCategory() string { return model.CategoryValidation }
func (repairLineFinding) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "line-reference"}
}
func (repairLineFinding) RunRule(nodes []*yaml.Node, _ model.RuleFunctionContext) []model.RuleFunctionResult {
	return []model.RuleFunctionResult{{StartNode: nodes[0], Path: "$['x-sentinel']", Message: fmt.Sprintf("value at line `%d` needs review", nodes[0].Line)}}
}

func TestRepairIndependentParameterFixesInOnePass(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info: {title: Test, version: '1'}
paths:
  /pets/{id}:
    get:
      parameters:
        - {name: id, in: path, required: false, schema: {type: string}}
        - {name: kind, in: query, schema: {type: string, enum: [cat, cat, dog]}}
      responses: {'200': {description: ok}}
`)
	builtins := rulesets.GetAllBuiltInRules()
	e := &motor.RuleSetExecution{Spec: source, SpecFileName: "test.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"path-params": builtins["path-params"], "duplicated-entry-in-enum": builtins["duplicated-entry-in-enum"]}}}
	_, result := RunRepairs(e, nil, nil, true)
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Empty(t, result.RepairWarnings)
	require.Len(t, result.FixedResults, 2)
	require.Empty(t, result.Results)
}

type repairNewFinding struct{}

func (repairNewFinding) GetCategory() string { return model.CategoryValidation }
func (repairNewFinding) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "new-finding"}
}
func (repairNewFinding) RunRule(nodes []*yaml.Node, _ model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes[0].Content) != 2 {
		return nil
	}
	return []model.RuleFunctionResult{{StartNode: nodes[0], Message: "new constraint violation"}}
}
func TestRepairDeclinesNewDiagnosticAndKeepsBaseline(t *testing.T) {
	e := &motor.RuleSetExecution{Spec: []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["a","a","b"]}}}}`), SpecFileName: "test.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{
		"duplicated-entry-in-enum": rulesets.GetAllBuiltInRules()["duplicated-entry-in-enum"],
		"new-finding":              {Id: "new-finding", Given: "$.components.schemas.Pet.enum", Severity: "warn", Then: &model.RuleAction{Function: "new-finding"}},
	}}, CustomFunctions: map[string]model.RuleFunction{"new-finding": repairNewFinding{}}}
	_, result := RunRepairs(e, nil, nil, true)
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Len(t, result.RepairWarnings, 1)
	require.Contains(t, result.RepairWarnings[0], "introduces diagnostic")
	require.Empty(t, result.ModifiedSpec)
	require.Empty(t, result.FixedResults)
	require.Len(t, result.Results, 1)
	require.Equal(t, "duplicated-entry-in-enum", result.Results[0].RuleId)
}

func TestPluginRepairCannotSuppressFindings(t *testing.T) {
	rule := &model.Rule{Id: "plugin", Given: "$.info.description", Severity: "warn", AutoFixFunction: "fix", Then: &model.RuleAction{Function: "truthy"}}
	e := &motor.RuleSetExecution{Spec: []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1","description":""},"paths":{}}`), SpecFileName: "test.json", SilenceLogs: true, ApplyAutoFixes: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"plugin": rule}}, AutoFixFunctions: map[string]model.AutoFixFunction{"fix": func(n, doc *yaml.Node, _ *model.RuleFunctionContext) (*yaml.Node, error) {
		n.Value = "Fixed"
		doc.Content[0].Content = append(doc.Content[0].Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x-lint-ignore"}, &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "plugin"}}})
		return n, nil
	}}}
	_, result := RunRepairs(e, nil, nil, true)
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Len(t, result.RepairWarnings, 1)
	require.Contains(t, result.RepairWarnings[0], "ignore directives")
	require.Len(t, result.Results, 1)
	require.Empty(t, result.FixedResults)
	require.Empty(t, result.ModifiedSpec)
}

func TestPluginRepairTargetAfterBuiltinSequenceRemoval(t *testing.T) {
	rule := &model.Rule{Id: "plugin", Given: "$.tags[*].name", Severity: "warn", AutoFixFunction: "rename", Then: &model.RuleAction{Function: "pattern", FunctionOptions: map[string]interface{}{"notMatch": "^zoo$"}}}
	e := &motor.RuleSetExecution{Spec: []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"tags":[{"name":"pets"},{"name":"pets"},{"name":"zoo"}]}`), SpecFileName: "test.json", SilenceLogs: true, ApplyAutoFixes: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"plugin": rule, "openapi-tags": rulesets.GetAllBuiltInRules()["openapi-tags"]}}, AutoFixFunctions: map[string]model.AutoFixFunction{"rename": func(n, _ *yaml.Node, _ *model.RuleFunctionContext) (*yaml.Node, error) {
		n.Value = "animals"
		return n, nil
	}}}
	_, result := RunRepairs(e, nil, nil, true)
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Empty(t, result.RepairWarnings)
	require.Len(t, result.FixedResults, 2)
	require.Empty(t, result.Results)
}

func TestPluginRepairWithParsedRuleSet(t *testing.T) {
	rs, err := rulesets.CreateRuleSetFromData([]byte("rules:\n  title:\n    given: $.info.title\n    severity: error\n    autoFixFunction: clearTitle\n    then: {function: falsy}\n"))
	require.NoError(t, err)
	e := &motor.RuleSetExecution{Spec: []byte("openapi: 3.0.3\ninfo: {title: Bad, version: '1'}\npaths: {}\n"), SpecFileName: "test.yaml", SilenceLogs: true, ApplyAutoFixes: true, RuleSet: rs, AutoFixFunctions: map[string]model.AutoFixFunction{"clearTitle": func(n, _ *yaml.Node, _ *model.RuleFunctionContext) (*yaml.Node, error) { n.Value = ""; return n, nil }}}
	_, result := RunRepairs(e, nil, nil, true)
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Empty(t, result.RepairWarnings)
	require.Len(t, result.FixedResults, 1)
	require.Empty(t, result.Results)
}

func TestRepairExistingCircularDiagnosticSurvivesFormatting(t *testing.T) {
	source := []byte(`openapi: 3.0.3
info: {title: Circular, version: '1'}
paths: {}
components:
  schemas:
    Node:
      type: object
      required: [next]
      properties:
        next:
          $ref: '#/components/schemas/Node'
`)
	id := "info-description"
	review, err := NewFixReview(&motor.RuleSetExecution{Spec: source, SpecFileName: "circular.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rulesets.GetAllBuiltInRules()[id]}}}, nil)
	require.NoError(t, err)
	defer review.Close()
	var choice FixChoice
	found := false
	for _, o := range review.Options() {
		if o.Result.RuleId == id {
			choice = FixChoice{Index: o.Index}
			found = true
		}
	}
	require.True(t, found)
	p, err := review.Preview([]FixChoice{choice})
	require.NoError(t, err)
	defer p.Close()
	require.Equal(t, RepairImpactUnknown, p.Impact.Status)
}
