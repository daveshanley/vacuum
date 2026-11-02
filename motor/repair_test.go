package motor

import (
	"encoding/json"
	"fmt"
	"github.com/pb33f/go-yaml"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

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
			e := &RuleSetExecution{Spec: []byte(tc.spec), SpecFileName: "test.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{tc.rule: builtins[tc.rule]}}}
			plan, baseline, err := PlanFixes(e, nil)
			require.NoError(t, err)
			defer baseline.ReleaseOwnedResources()
			require.NotEmpty(t, baseline.Results)
			require.NotEmpty(t, plan.Targets, "plan: %+v baseline: %+v", plan, baseline.Results)
			result, err := ValidateFixPlan(e, plan, RepairApproval{}, nil)
			require.NoError(t, err)
			defer result.Final.ReleaseOwnedResources()
			require.NotEmpty(t, result.Verified)
			require.Empty(t, result.Final.Results)
			e.Spec = result.Candidate
			next, lint, err := PlanFixes(e, nil)
			require.NoError(t, err)
			defer lint.ReleaseOwnedResources()
			require.Empty(t, next.Operations)
			require.Empty(t, next.KeyTags)
		})
	}
}

func TestExternalPlanRequiresIndependentApproval(t *testing.T) {
	source := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n")
	e := &RuleSetExecution{Spec: source, SpecFileName: "test.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}}
	plan := repair.NewPlan(e.SpecFileName, source)
	plan.Operations = []repair.Operation{{Op: "add", Path: "/info/description", Value: json.RawMessage(`"Approved description"`)}}
	_, err := ValidateFixPlan(e, plan, RepairApproval{}, nil)
	require.ErrorContains(t, err, "differs")
	encoded, err := json.Marshal(plan)
	require.NoError(t, err)
	result, err := ValidateFixPlan(e, plan, RepairApproval{ApprovedSHA256: repair.Hash(encoded)}, nil)
	require.NoError(t, err)
	defer result.Final.ReleaseOwnedResources()
	require.Empty(t, result.Verified)
	require.Empty(t, result.Final.Results)
	plan.Operations[0].Path = "/x-lint-ignore"
	plan.Operations[0].Value = json.RawMessage(`"info-description"`)
	_, err = ValidateFixPlan(e, plan, RepairApproval{}, nil)
	require.ErrorContains(t, err, "differs")
}

func TestRepairKeepsUnchangedDiagnosticAcrossFormatting(t *testing.T) {
	spec := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1","description":"Shared description"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","description":"Shared description","enum":["a","a"]}}}}`)
	builtins := rulesets.GetAllBuiltInRules()
	e := &RuleSetExecution{Spec: spec, SpecFileName: "test.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"duplicated-entry-in-enum": builtins["duplicated-entry-in-enum"], "description-duplication": builtins["description-duplication"]}}}
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
			e := &RuleSetExecution{Spec: []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1","description":""},"paths":{},"components":{"schemas":{"Pet":{"enum":["a","a"]}}}}`), SpecFileName: "test.json", SilenceLogs: true, ApplyAutoFixes: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"plugin": rule, "duplicated-entry-in-enum": rulesets.GetAllBuiltInRules()["duplicated-entry-in-enum"]}}, AutoFixFunctions: map[string]model.AutoFixFunction{"fix": func(n, doc *yaml.Node, _ *model.RuleFunctionContext) (*yaml.Node, error) {
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
				require.NotEmpty(t, result.Errors)
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
	e := &RuleSetExecution{Spec: spec, SpecFileName: filepath.Join(t.TempDir(), "api.json"), AllowLookup: true, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-contact": rulesets.GetAllBuiltInRules()["info-contact"]}}}
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
			e := &RuleSetExecution{Spec: []byte(tc.spec), SpecFileName: "/tmp/async-repair.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{tc.rule: rulesets.GetAllAsyncAPIRules()[tc.rule]}}}
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

func TestApprovedPlanCannotClaimSuppressionAsRepair(t *testing.T) {
	e := &RuleSetExecution{Spec: []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"), SpecFileName: "test.yaml", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}}
	plan := repair.NewPlan(e.SpecFileName, e.Spec)
	plan.Operations = []repair.Operation{{Op: "add", Path: "/x-lint-ignore", Value: json.RawMessage(`["info-description"]`)}}
	data, err := json.Marshal(plan)
	require.NoError(t, err)
	_, err = ValidateFixPlan(e, plan, RepairApproval{ApprovedSHA256: repair.Hash(data)}, nil)
	require.ErrorContains(t, err, "ignore directives")
	plan.Operations = []repair.Operation{{Op: "add", Path: "/info/description", Value: json.RawMessage(`"Useful description"`)}}
	plan.Targets = []repair.Target{{RuleID: "info-description", Pointer: "/info", Code: "invented-code"}}
	data, err = json.Marshal(plan)
	require.NoError(t, err)
	_, err = ValidateFixPlan(e, plan, RepairApproval{ApprovedSHA256: repair.Hash(data)}, nil)
	require.ErrorContains(t, err, "target code")
}
