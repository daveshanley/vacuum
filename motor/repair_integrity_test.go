package motor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

const integritySpec = `{"openapi":"3.1.0","info":{"title":"Repair","version":"1"},"paths":{"/a":{"get":{"operationId":"a","responses":{"200":{"description":"ok"}}}},"/b":{"get":{"operationId":"b","responses":{"200":{"description":"ok"}}}}},"components":{"schemas":{"Item":{"type":"string"}}}}`

func TestRepairIntegrityIndependentOfRules(t *testing.T) {
	for _, tc := range []struct{ name, path, value, want string }{
		{"schema", "/components/schemas/Item/type", `7`, "schema"},
		{"identifier", "/paths/~1b/get/operationId", `"a"`, "duplicate operationId"},
		{"reference", "/components/schemas/Item/$ref", `"#/components/schemas/Missing"`, "reference"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			execution := &RuleSetExecution{SpecFileName: "repair.json", Spec: []byte(integritySpec), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}}
			_, before, err := PlanFixes(execution, nil)
			require.NoError(t, err)
			defer before.ReleaseOwnedResources()
			plan := repair.NewPlan(execution.SpecFileName, execution.Spec)
			plan.Operations = []repair.Operation{{Op: "add", Path: "/info/description", Value: json.RawMessage(`"Repaired description"`)}, {Op: "add", Path: tc.path, Value: json.RawMessage(tc.value)}}
			checked, err := validatePreparedPlan(execution, plan, nil, before)
			require.Error(t, err)
			require.Nil(t, checked)
			require.Contains(t, err.Error(), tc.want)
			require.Equal(t, integritySpec, string(execution.Spec))
		})
	}
}

func TestRepairIntegrityPreservesExistingSchemaDefect(t *testing.T) {
	execution := &RuleSetExecution{SpecFileName: "repair.json", Spec: []byte(integritySpec), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}}
	existing := repair.NewPlan(execution.SpecFileName, execution.Spec)
	existing.Operations = []repair.Operation{{Op: "add", Path: "/components/schemas/Item/type", Value: json.RawMessage(`7`)}}
	data, err := repair.Apply(execution.SpecFileName, execution.Spec, existing)
	require.NoError(t, err)
	execution.Spec = data
	_, before, err := PlanFixes(execution, nil)
	require.NoError(t, err)
	defer before.ReleaseOwnedResources()
	plan := repair.NewPlan(execution.SpecFileName, execution.Spec)
	plan.Operations = []repair.Operation{{Op: "add", Path: "/info/description", Value: json.RawMessage(`"Repaired description"`)}}
	checked, err := validatePreparedPlan(execution, plan, nil, before)
	require.NoError(t, err)
	defer checked.Final.ReleaseOwnedResources()
	require.NotEmpty(t, checked.Candidate)
}

func TestRepairIntegrityJSONSchemaWithEmptyRules(t *testing.T) {
	execution := &RuleSetExecution{SpecFileName: "schema.json", SpecFormat: model.JSONSchemaDraft2020, SkipDocumentCheck: true, Spec: []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{}}}
	_, before, err := PlanFixes(execution, nil)
	require.NoError(t, err)
	defer before.ReleaseOwnedResources()
	plan := repair.NewPlan(execution.SpecFileName, execution.Spec)
	plan.Operations = []repair.Operation{{Op: "replace", Path: "/type", Value: json.RawMessage(`7`)}}
	checked, err := validatePreparedPlan(execution, plan, nil, before)
	require.Nil(t, checked)
	require.ErrorContains(t, err, "schema")
}

func TestRepairIntegrityDuplicateKeys(t *testing.T) {
	execution := &RuleSetExecution{SpecFileName: "repair.json", Spec: []byte(integritySpec), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{}}}
	result := ApplyRulesToRuleSet(execution)
	defer result.ReleaseOwnedResources()
	root, err := repair.ParseDocument([]byte(`{"a":1,"a":2}`))
	require.NoError(t, err)
	result.RuleSetExecution.CanonicalDocument = root
	defects, err := repairIntegrityDefects(result)
	require.NoError(t, err)
	found := false
	for _, d := range defects {
		if d.kind == "duplicate key" && d.pointer == "/a" {
			found = true
		}
	}
	require.True(t, found)
}

func TestRepairIntegrityAsyncAPIMessageMembership(t *testing.T) {
	execution := &RuleSetExecution{SpecFileName: "events.json", Spec: []byte(`{"asyncapi":"3.0.0","info":{"title":"Events","version":"1"},"channels":{"events":{"address":"events","messages":{"Ping":{"payload":{"type":"string"}}}}},"operations":{"receive":{"action":"receive","channel":{"$ref":"#/channels/events"},"messages":[{"$ref":"#/channels/events/messages/Ping"}]}},"components":{"messages":{"Other":{"payload":{"type":"string"}}}}}`), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{}}}
	_, before, err := PlanFixes(execution, nil)
	require.NoError(t, err)
	defer before.ReleaseOwnedResources()
	plan := repair.NewPlan(execution.SpecFileName, execution.Spec)
	plan.Operations = []repair.Operation{{Op: "replace", Path: "/operations/receive/messages/0/$ref", Value: json.RawMessage(`"#/components/messages/Other"`)}}
	checked, err := validatePreparedPlan(execution, plan, nil, before)
	require.Nil(t, checked)
	require.ErrorContains(t, err, "AsyncAPI")
}

func TestRepairExternalSourceDescriptions(t *testing.T) {
	for _, tc := range []struct {
		value    string
		external bool
	}{
		{`[]`, false}, {`[{"name":"api","url":"api.yaml","type":"openapi"}]`, true},
		{`null`, true}, {`{}`, true}, {`""`, true}, {`false`, true},
	} {
		t.Run(tc.value, func(t *testing.T) {
			require.Equal(t, tc.external, externalRepairBytes([]byte(`{"sourceDescriptions":`+tc.value+`}`)))
		})
	}
}

func TestRepairIntegrityOperationLinks(t *testing.T) {
	for _, tc := range []struct {
		name, link, path, value string
		wantError               bool
	}{
		{"renamed ID", `{"operationId":"a"}`, "/paths/~1a/get/operationId", `"renamed"`, true},
		{"bad operationRef", `{"operationRef":"#/paths/~1a/get"}`, "/paths/~1b/get/responses/200/links/next/operationRef", `"#/components/schemas/Item"`, true},
		{"external operationRef", `{"operationRef":"#/paths/~1a/get"}`, "/paths/~1b/get/responses/200/links/next/operationRef", `"https://example.test/api.yaml#/paths/~1a/get"`, true},
		{"valid operationRef", `{"operationRef":"#/paths/~1a/get"}`, "/info/description", `"documentation"`, false},
		{"preexisting missing ID", `{"operationId":"missing"}`, "/info/description", `"documentation"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			execution := &RuleSetExecution{SpecFileName: "links.json", Spec: []byte(integritySpec), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{}}}
			setup := repair.NewPlan(execution.SpecFileName, execution.Spec)
			setup.Operations = []repair.Operation{{Op: "add", Path: "/paths/~1b/get/responses/200/links", Value: json.RawMessage(`{"next":` + tc.link + `}`)}}
			var err error
			execution.Spec, err = repair.Apply(execution.SpecFileName, execution.Spec, setup)
			require.NoError(t, err)
			_, before, err := PlanFixes(execution, nil)
			require.NoError(t, err)
			defer before.ReleaseOwnedResources()
			plan := repair.NewPlan(execution.SpecFileName, execution.Spec)
			plan.Operations = []repair.Operation{{Op: "add", Path: tc.path, Value: json.RawMessage(tc.value)}}
			result, err := validatePreparedPlan(execution, plan, nil, before)
			if tc.wantError {
				require.Nil(t, result)
				require.ErrorContains(t, err, "operation link")
			} else {
				require.NoError(t, err)
				result.Final.ReleaseOwnedResources()
			}
		})
	}
}

func TestRepairIntegrityOperationIDsAcrossOwners(t *testing.T) {
	for _, owner := range []string{"webhook", "callback"} {
		t.Run(owner, func(t *testing.T) {
			execution := &RuleSetExecution{SpecFileName: "operations.json", Spec: []byte(integritySpec), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{}}}
			setup := repair.NewPlan(execution.SpecFileName, execution.Spec)
			pointer := "/webhooks/event/post/operationId"
			if owner == "webhook" {
				setup.Operations = []repair.Operation{{Op: "add", Path: "/webhooks", Value: json.RawMessage(`{"event":{"post":{"operationId":"event","responses":{"200":{"description":"ok"}}}}}`)}}
			} else {
				pointer = "/paths/~1a/get/callbacks/event/{$request.body#~1url}/post/operationId"
				setup.Operations = []repair.Operation{{Op: "add", Path: "/paths/~1a/get/callbacks", Value: json.RawMessage(`{"event":{"{$request.body#/url}":{"post":{"operationId":"event","responses":{"200":{"description":"ok"}}}}}}`)}}
			}
			var err error
			execution.Spec, err = repair.Apply(execution.SpecFileName, execution.Spec, setup)
			require.NoError(t, err)
			_, before, err := PlanFixes(execution, nil)
			require.NoError(t, err)
			defer before.ReleaseOwnedResources()
			plan := repair.NewPlan(execution.SpecFileName, execution.Spec)
			plan.Operations = []repair.Operation{{Op: "replace", Path: pointer, Value: json.RawMessage(`"a"`)}}
			checked, err := validatePreparedPlan(execution, plan, nil, before)
			require.Nil(t, checked)
			require.ErrorContains(t, err, "duplicate operationId")
			plan.Operations[0].Value = json.RawMessage(`"newUnique"`)
			checked, err = validatePreparedPlan(execution, plan, nil, before)
			require.NoError(t, err)
			checked.Final.ReleaseOwnedResources()
		})
	}
}

func TestRepairIntegrityExternalOperationIDCollision(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "external.yaml"), []byte("get:\n  operationId: remoteId\n  responses:\n    '200':\n      description: OK\n"), 0600))
	spec := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{"/remote":{"$ref":"external.yaml"},"/local":{"get":{"responses":{"200":{"description":"OK"}}}}}}`)
	id := "operation-operationId"
	execution := &RuleSetExecution{Spec: spec, SpecFileName: filepath.Join(dir, "api.json"), Base: dir, AllowLookup: true, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rulesets.GetAllBuiltInRules()[id]}}}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	require.NoError(t, os.Remove(filepath.Join(dir, "external.yaml")), "verification must use the captured external operation")
	var choice FixChoice
	found := false
	for _, o := range review.Options() {
		if o.Available {
			choice.Index = o.Index
			found = true
			break
		}
	}
	require.True(t, found)
	duplicate := "remoteId"
	choice.Value = &duplicate
	preview, err := review.Preview([]FixChoice{choice})
	require.Nil(t, preview)
	require.ErrorContains(t, err, "duplicate operationId")
	unique := "localId"
	choice.Value = &unique
	preview, err = review.Preview([]FixChoice{choice})
	require.NoError(t, err)
	defer preview.Close()
	require.NotEmpty(t, preview.Candidate)
}

func TestRepairIntegrityPreservesExistingExternalIDCollision(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "external.yaml"), []byte("get:\n  operationId: remoteId\n  responses:\n    '200':\n      description: OK\n"), 0600))
	spec := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{"/remote":{"$ref":"external.yaml"},"/local":{"get":{"operationId":"remoteId","responses":{"200":{"description":"OK"}}}}}}`)
	id := "info-description"
	execution := &RuleSetExecution{Spec: spec, SpecFileName: filepath.Join(dir, "api.json"), Base: dir, AllowLookup: true, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{id: rulesets.GetAllBuiltInRules()[id]}}}
	review, err := NewFixReview(execution, nil)
	require.NoError(t, err)
	defer review.Close()
	var choices []FixChoice
	for _, o := range review.Options() {
		if o.Available {
			choices = append(choices, FixChoice{Index: o.Index})
		}
	}
	require.NotEmpty(t, choices)
	preview, err := review.Preview(choices)
	require.NoError(t, err)
	defer preview.Close()
}

// Exercise the public review path: only path-item-refs is enabled, and the
// reviewer edits the replacement operation before validation.
func TestRepairIntegrityEditedOperationSemantics(t *testing.T) {
	const valid = `{"responses":{"200":{"description":"ok"}}}`
	const duplicate = `{"parameters":[{"name":"id","in":"query","schema":{"type":"string"}},{"name":"id","in":"query","schema":{"type":"integer"}}],"responses":{"200":{"description":"ok"}}}`
	const unused = `{"parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"ok"}}}`
	const undefined = `{"security":[{"Unknown":[]}],"responses":{"200":{"description":"ok"}}}`
	for _, tc := range []struct {
		name, value, existing, want string
		skipCheck, skipResolve      bool
	}{
		{name: "duplicate parameters", value: duplicate, want: "operation parameter"},
		{name: "unused path parameter", value: unused, want: "oasPathParam"},
		{name: "undefined security scheme", value: undefined, want: "oasOpSecurityDefined"},
		{name: "skip document check cannot bypass", value: undefined, want: "oasOpSecurityDefined", skipCheck: true},
		{name: "skip resolve cannot bypass", value: undefined, want: "oasOpSecurityDefined", skipResolve: true},
		{name: "undefined second security scheme", value: `{"security":[{"Known":[],"Unknown":[]}],"responses":{"200":{"description":"ok"}}}`, want: "oasOpSecurityDefined"},
		{name: "defined security scheme", value: `{"security":[{"Known":[]}],"responses":{"200":{"description":"ok"}}}`},
		{name: "valid replacement", value: valid},
		{name: "existing duplicate parameters", value: valid, existing: duplicate},
		{name: "existing unused path parameter", value: valid, existing: unused},
		{name: "existing undefined security scheme", value: valid, existing: undefined},
		{name: "existing defect does not permit another", value: duplicate, existing: duplicate, want: "operation parameter"},
		{name: "same parameter name in different locations", value: `{"parameters":[{"name":"id","in":"query","schema":{"type":"string"}},{"name":"id","in":"header","schema":{"type":"integer"}}],"responses":{"200":{"description":"ok"}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extra := ""
			if tc.existing != "" {
				extra = `,"/existing":{"get":` + tc.existing + `}`
			}
			source := []byte(`{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"$ref":"#/paths/~1source/get"}},"/source":{"get":` + valid + `}` + extra + `},"components":{"securitySchemes":{"Known":{"type":"apiKey","name":"key","in":"header"}}}}`)
			execution := &RuleSetExecution{Spec: source, SpecFileName: filepath.Join(t.TempDir(), "api.json"), SkipDocumentCheck: tc.skipCheck, SkipResolve: tc.skipResolve, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"path-item-refs": rulesets.GetAllBuiltInRules()["path-item-refs"]}}}
			review, err := NewFixReview(execution, nil)
			require.NoError(t, err)
			defer review.Close()
			var choices []FixChoice
			for _, option := range review.Options() {
				if option.Available {
					require.True(t, option.Editable)
					choices = append(choices, FixChoice{Index: option.Index, Value: &tc.value})
				}
			}
			require.Len(t, choices, 1)
			preview, err := review.Preview(choices)
			if tc.want != "" {
				require.ErrorContains(t, err, tc.want)
				require.Nil(t, preview)
			} else {
				require.NoError(t, err)
				require.NotNil(t, preview)
				preview.Close()
			}
			require.Equal(t, source, execution.Spec)
		})
	}
}

func TestRepairIntegrityRequiresOpenAPISemanticModel(t *testing.T) {
	execution := &RuleSetExecution{SpecFileName: "repair.json", Spec: []byte(integritySpec), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{}}}
	result := ApplyRulesToRuleSet(execution)
	defer result.ReleaseOwnedResources()
	doctor := result.RuleSetExecution.DrDocument
	require.NotNil(t, doctor)
	result.RuleSetExecution.DrDocument = nil
	_, err := repairIntegrityDefects(result)
	result.RuleSetExecution.DrDocument = doctor
	require.ErrorContains(t, err, "integrity Doctor model is unavailable")
}

func TestRepairIntegrityOAS2Security(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new undefined scheme", true: "existing undefined scheme"}[existing], func(t *testing.T) {
			security := ""
			if existing {
				security = `,"security":[{"Unknown":[]}]`
			}
			execution := &RuleSetExecution{SpecFileName: "swagger.json", Spec: []byte(`{"swagger":"2.0","info":{"title":"Repair","version":"1"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"ok"}}` + security + `}}},"securityDefinitions":{"Known":{"type":"apiKey","name":"key","in":"header"}}}`), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}}
			_, before, err := PlanFixes(execution, nil)
			require.NoError(t, err)
			defer before.ReleaseOwnedResources()
			plan := repair.NewPlan(execution.SpecFileName, execution.Spec)
			plan.Operations = []repair.Operation{{Op: "add", Path: "/info/description", Value: json.RawMessage(`"Repaired description"`)}}
			if !existing {
				plan.Operations = append(plan.Operations, repair.Operation{Op: "add", Path: "/paths/~1pets/get/security", Value: json.RawMessage(`[{"Known":[],"Unknown":[]}]`)})
			}
			checked, err := validatePreparedPlan(execution, plan, nil, before)
			if existing {
				require.NoError(t, err)
				checked.Final.ReleaseOwnedResources()
			} else {
				require.ErrorContains(t, err, "oas2OpSecurityDefined")
				require.Nil(t, checked)
			}
		})
	}
}

func TestRepairIntegrityPreservesExternalParameterDefects(t *testing.T) {
	for _, parameter := range []string{
		`[{"name":"id","in":"query","schema":{"type":"string"}},{"name":"id","in":"query","schema":{"type":"string"}}]`,
		`[{"name":"unused","in":"path","required":true,"schema":{"type":"string"}}]`,
	} {
		t.Run(parameter, func(t *testing.T) {
			dir := t.TempDir()
			external := filepath.Join(dir, "external.json")
			require.NoError(t, os.WriteFile(external, []byte(`{"get":{"parameters":`+parameter+`,"responses":{"200":{"description":"ok"}}}}`), 0600))
			execution := &RuleSetExecution{SpecFileName: filepath.Join(dir, "api.json"), Base: dir, AllowLookup: true, Spec: []byte(`{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/remote":{"$ref":"external.json"},"/pets":{"get":{"$ref":"#/paths/~1source/get"}},"/source":{"get":{"responses":{"200":{"description":"ok"}}}}}}`), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"path-item-refs": rulesets.GetAllBuiltInRules()["path-item-refs"]}}}
			review, err := NewFixReview(execution, nil)
			require.NoError(t, err)
			defer review.Close()
			require.NoError(t, os.Remove(external))
			value := `{"responses":{"200":{"description":"ok"}}}`
			var choices []FixChoice
			for _, option := range review.Options() {
				if option.Available {
					choices = append(choices, FixChoice{Index: option.Index, Value: &value})
				}
			}
			require.Len(t, choices, 1)
			preview, err := review.Preview(choices)
			require.NoError(t, err)
			preview.Close()
		})
	}
}
