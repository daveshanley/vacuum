package motor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func reviewedRecipeDocument(extra string) string {
	return `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{}` + extra + `}`
}
func reviewedRecipeSchema(schema string) string {
	return reviewedRecipeDocument(`,"components":{"schemas":{"Value":` + schema + `}}`)
}

func TestReviewedRecipesThroughRealDetectors(t *testing.T) {
	cases := []struct{ rule, source string }{
		{"json-schema-title-description-type", `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`},
		{"json-schema-composition-sanity", `{"$schema":"https://json-schema.org/draft/2020-12/schema","oneOf":[{"type":"string"},{"type":"string"}]}`},
		{"oas2-discriminator", `{"swagger":"2.0","info":{"title":"Review","version":"1"},"paths":{},"definitions":{"Value":{"type":"object","discriminator":"kind"}}}`},
		{"oas2-operation-formData-consume-check", `{"swagger":"2.0","info":{"title":"Review","version":"1"},"paths":{"/upload":{"post":{"parameters":[{"in":"formData","name":"file","type":"file"}],"responses":{"200":{"description":"ok"}}}}}}`},
		{"operation-operationId-valid-in-url", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"operationId":"list pets","responses":{"200":{"description":"ok","links":{"next":{"operationId":"list pets"},"examples":{"operationId":"list pets"}}}}}}}}`},
		{"operation-operationId-unique", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/a":{"get":{"operationId":"list","responses":{"200":{"description":"ok"}}}},"/b":{"get":{"operationId":"list","responses":{"200":{"description":"ok"}}}}}}`},
		{"paths-kebab-case", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/myPets":{"get":{"responses":{"200":{"description":"ok"}}}}}}`},
		{"path-keys-no-trailing-slash", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets/":{"get":{"responses":{"200":{"description":"ok"}}}}}}`},
		{"no-http-verbs-in-path", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/get/pets":{"get":{"responses":{"200":{"description":"ok"}}}}}}`},
		{"path-not-include-query", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets?":{"get":{"responses":{"200":{"description":"ok"}}}}}}`},
		{"asyncapi-3-channel-no-query-nor-fragment", `{"asyncapi":"3.0.0","info":{"title":"Review","version":"1"},"channels":{"orders":{"address":"orders?filter=active"}},"operations":{}}`},
		{"asyncapi-channel-parameters", `{"asyncapi":"3.0.0","info":{"title":"Review","version":"1"},"channels":{"orders":{"address":"orders/{tenant}"}},"operations":{}}`},
		{"required-fields-defined", reviewedRecipeSchema(`{"type":"object","properties":{"name":{"type":"string"}},"required":["id"],"example":{"id":5}}`)},
		{"camel-case-properties", reviewedRecipeSchema(`{"type":"object","properties":{"first_name":{"type":"string"}},"required":["first_name"],"example":{"first_name":"Example"}}`)},
		{"oas2-unused-definition", `{"swagger":"2.0","info":{"title":"Review","version":"1"},"paths":{},"definitions":{"Unused":{"type":"string"}}}`},
		{"oas3-unused-component", reviewedRecipeSchema(`{"type":"string"}`)},
		{"asyncapi-unused-components", `{"asyncapi":"3.0.0","info":{"title":"Review","version":"1"},"channels":{},"operations":{},"components":{"schemas":{"Unused":{"type":"string"}}}}`},
		{"no-eval-in-markdown", reviewedRecipeSchema(`{"type":"string","description":"<script>eval(1)</script>"}`)},
		{"license-url", `{"openapi":"3.1.0","info":{"title":"Review","version":"1","license":{"name":"MIT"}},"paths":{}}`},
		{"asyncapi-info-license-url", `{"asyncapi":"3.0.0","info":{"title":"Review","version":"1","license":{"name":"MIT"}},"channels":{},"operations":{}}`},
		{"info-license-spdx", `{"openapi":"3.1.0","info":{"title":"Review","version":"1","license":{"name":"MIT","identifier":"MIT","url":"https://opensource.org/license/mit/"}},"paths":{}}`},
		{"oas2-schema", `{"swagger":"2.0","info":{"title":"Review","version":"1"},"paths":{},"definitions":{"Value":{"type":"object","properties":{"id":{"type":"string"}},"required":["id","id"]}}}`},
		{"oas3-schema", strings.Replace(reviewedRecipeSchema(`{"type":"object","properties":{"id":{"type":"string"}},"required":["id","id"]}`), "3.1.0", "3.0.3", 1)},
		{"json-schema-valid", `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"id":{"type":"string"}},"required":["id","id"]}`},
		{"arazzo-structure", `{"arazzo":"1.1.0","info":{"title":"Review","version":"1"},"sourceDescriptions":[],"workflows":[{"workflowId":"run","inputs":{"type":"object","properties":{"id":{"type":"string"}},"required":["id","id"]},"steps":[{"stepId":"read","operationId":"read"}]}]}`},
		{"json-schema-type-constraint-compatible", `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string","minimum":1}`},
		{"migrate-zally-ignore", reviewedRecipeDocument(`,"x-zally-ignore":[]`)},
		{"no-$ref-siblings", strings.Replace(reviewedRecipeDocument(`,"components":{"schemas":{"Text":{"type":"string","description":"Text value."},"Alias":{"$ref":"#/components/schemas/Text","description":"Text value."}}}`), "3.1.0", "3.0.3", 1)},
		{"path-item-refs", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"$ref":"#/paths/~1source/get"}},"/source":{"get":{"responses":{"200":{"description":"ok"}}}}}}`},
		{"paths-specificity-order", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets/{id}":{"get":{"parameters":[{"in":"path","name":"id","required":true,"schema":{"type":"string"}}],"responses":{"200":{"description":"ok"}}}},"/pets/special":{"get":{"responses":{"200":{"description":"ok"}}}}}}`},
		{"arazzo-advisory", `{"arazzo":"1.1.0","info":{"title":"Review","version":"1"},"sourceDescriptions":[{"name":"api","url":"api.json","type":"openapi"}],"workflows":[{"workflowId":"run work","steps":[{"stepId":"read","operationId":"read"}]}]}`},
		{"arazzo-duplicate-id", `{"arazzo":"1.1.0","info":{"title":"Review","version":"1"},"sourceDescriptions":[{"name":"api","url":"api.json","type":"openapi"}],"workflows":[{"workflowId":"run","steps":[{"stepId":"first","operationId":"read"},{"stepId":"first","operationId":"write"}]}]}`},
		{"oas3-example-external-check", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"type":"string"},"examples":{"sample":{"value":"sample","externalValue":"data:application/json,%22sample%22"}}}}}}}}}}`},
		{"arazzo-source-type", `{"arazzo":"1.1.0","info":{"title":"Review","version":"1"},"sourceDescriptions":[{"name":"api","url":"api.json","type":"asyncapi"}],"workflows":[{"workflowId":"run","steps":[{"stepId":"read","operationId":"read"}]}]}`},
		{"arazzo-step-operationPath", `{"arazzo":"1.1.0","info":{"title":"Review","version":"1"},"sourceDescriptions":[{"name":"api","url":"api.json","type":"openapi"}],"workflows":[{"workflowId":"run","steps":[{"stepId":"read","operationPath":"{$sourceDescriptions.api.url}#/paths/~1read/get"}]}]}`},
		{"oas3-missing-example", reviewedRecipeSchema(`{"type":"object","required":["id"],"properties":{"id":{"type":"integer","minimum":1,"maximum":10}}}`)},
		{"oas3-valid-schema-example", reviewedRecipeSchema(`{"type":"integer","minimum":1,"maximum":10,"example":"wrong"}`)},
		{"json-schema-examples-valid", `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"integer","minimum":1,"maximum":10,"examples":["wrong"]}`},
		{"asyncapi-message-examples", `{"asyncapi":"3.0.0","info":{"title":"Review","version":"1"},"channels":{},"components":{"messages":{"Event":{"payload":{"type":"integer","minimum":1,"maximum":10},"examples":[{"summary":"Sample"}]}}}}`},
		{"nullable-enum-contains-null", reviewedRecipeSchema(`{"type":["string","null"],"enum":["a"]}`)},
		{"json-schema-required-properties-defined", `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"a":{"type":"string"}},"required":["a","a"]}`},
		{"operation-singular-tag", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"tags":["pets","pets"],"responses":{"200":{"description":"ok"}}}}}}`},
		{"oas2-anyOf", `{"swagger":"2.0","info":{"title":"Review","version":"1"},"paths":{},"definitions":{"Value":{"anyOf":[{"$ref":"#/definitions/Text"}]},"Text":{"type":"string"}}}`},
		{"oas2-oneOf", `{"swagger":"2.0","info":{"title":"Review","version":"1"},"paths":{},"definitions":{"Value":{"oneOf":[{"$ref":"#/definitions/Text"}]},"Text":{"type":"string"}}}`},
		{"oas2-host-trailing-slash", `{"swagger":"2.0","info":{"title":"Review","version":"1"},"paths":{},"host":"api.test/"}`},
		{"oas-missing-type", reviewedRecipeSchema(`{"minLength":1,"maxLength":30}`)},
		{"typed-enum", reviewedRecipeSchema(`{"type":"integer","enum":["2",3]}`)},
		{"owasp-integer-format", reviewedRecipeSchema(`{"type":"integer","minimum":0,"maximum":100}`)},
		{"owasp-integer-limit", reviewedRecipeSchema(`{"type":"integer","format":"int32","minimum":0}`)},
		{"owasp-auth-urls-https", reviewedRecipeDocument(`,"components":{"securitySchemes":{"oauth":{"type":"oauth2","flows":{"authorizationCode":{"authorizationUrl":"http://api.test/auth","tokenUrl":"https://api.test/token","scopes":{}}}}}}`)},
		{"owasp-security-hosts-https-oas3", reviewedRecipeDocument(`,"servers":[{"url":"http://api.test"}]`)},
		{"owasp-no-additionalProperties", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"post":{"requestBody":{"content":{"application/json":{"schema":{"type":"object","additionalProperties":true}}}},"responses":{"200":{"description":"ok"}}}}}}`},
		{"description-duplication", reviewedRecipeDocument(`,"components":{"schemas":{"A":{"type":"string","description":"An item."},"B":{"type":"string","description":"An item."}}}`)},
		{"owasp-jwt-best-practices", reviewedRecipeDocument(`,"components":{"securitySchemes":{"bearer":{"type":"http","scheme":"bearer","bearerFormat":"JWT"}}}`)},
		{"no-script-tags-in-markdown", reviewedRecipeSchema(`{"type":"string","description":"<script>alert(1)</script>"}`)},
		{"arazzo-no-script-tags-in-markdown", `{"arazzo":"1.1.0","info":{"title":"Review","version":"1","description":"<script>alert(1)</script>"},"sourceDescriptions":[],"workflows":[{"workflowId":"run","steps":[{"stepId":"read","operationId":"read"}]}]}`},
		{"oas3-server-trailing-slash", reviewedRecipeDocument(`,"servers":[{"url":"https://api.test/"}]`)},
		{"asyncapi-3-server-no-trailing-slash", `{"asyncapi":"3.0.0","info":{"title":"Review","version":"1"},"channels":{},"operations":{},"servers":{"test":{"host":"api.test/","protocol":"amqp"}}}`},
		{"asyncapi-3-channel-no-trailing-slash", `{"asyncapi":"3.0.0","info":{"title":"Review","version":"1"},"channels":{"orders":{"address":"orders/"}},"operations":{}}`},
		{"asyncapi-3-headers-schema-type-object", `{"asyncapi":"3.0.0","info":{"title":"Review","version":"1"},"channels":{},"operations":{},"components":{"messages":{"Event":{"headers":{"properties":{"id":{"type":"string"}}}}}}}`},
		{"asyncapi-3-tags", `{"asyncapi":"3.0.0","info":{"title":"Review","version":"1"},"channels":{},"operations":{}}`},
		{"operation-tags", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"ok"}}}}}}`},
		{"operation-tag-defined", `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"tags":["pets"],"responses":{"200":{"description":"ok"}}}}}}`},
	}
	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.rule] = true
		t.Run(tc.rule, func(t *testing.T) {
			all := rulesets.GetAllBuiltInRules()
			for k, v := range rulesets.GetAllOWASPRules() {
				all[k] = v
			}
			for k, v := range rulesets.GetAllAsyncAPIRules() {
				all[k] = v
			}
			for k, v := range rulesets.GetAllArazzoRules() {
				all[k] = v
			}
			for k, v := range rulesets.GenerateDefaultJSONSchemaRuleSet().Rules {
				all[k] = v
			}
			require.NotNil(t, all[tc.rule])
			execution := &RuleSetExecution{SpecFileName: "/tmp/reviewed-recipe.json", SilenceLogs: true, Spec: []byte(tc.source), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{tc.rule: all[tc.rule]}}}
			if strings.HasPrefix(tc.rule, "arazzo-") {
				directory := t.TempDir()
				execution.SpecFileName = filepath.Join(directory, "workflow.json")
				execution.AllowLookup = true
				require.NoError(t, os.WriteFile(filepath.Join(directory, "api.json"), []byte(`{"openapi":"3.1.0","info":{"title":"Source","version":"1"},"paths":{"/read":{"get":{"operationId":"read","responses":{"200":{"description":"ok"}}}},"/write":{"post":{"operationId":"write","responses":{"200":{"description":"ok"}}}}}}`), 0600))
			}
			if strings.HasPrefix(tc.rule, "json-schema-") {
				execution.SpecFormat = model.JSONSchemaDraft2020
				execution.SkipDocumentCheck = true
			}
			review, err := NewFixReview(execution, nil)
			require.NoError(t, err)
			defer review.Close()
			require.NotEmpty(t, review.Options(), "real detector did not emit a finding")
			var choices []FixChoice
			for _, option := range review.Options() {
				if option.Result.RuleId != tc.rule {
					continue
				}
				if !option.Available {
					t.Logf("declined %s: %s", option.Result.Path, option.Reason)
					continue
				}
				require.False(t, option.Selected)
				draft, err := review.Draft(FixChoice{Index: option.Index})
				require.NoError(t, err)
				require.NotEqual(t, draft.Before, draft.After)
				choices = append(choices, FixChoice{Index: option.Index})
			}
			require.NotEmpty(t, choices)
			preview, err := review.Preview(choices)
			require.NoError(t, err)
			defer preview.Close()
			require.NotEqual(t, tc.source, string(preview.Candidate))
			require.True(t, json.Valid(preview.Candidate))
			require.Equal(t, tc.source, string(execution.Spec))
			remaining := 0
			for _, result := range preview.Results {
				if result.RuleId == tc.rule {
					remaining++
				}
			}
			require.Less(t, remaining, len(review.Options()), "selected diagnostic was not removed")
			require.Contains(t, []RepairImpactStatus{RepairImpactBreaking, RepairImpactNoBreaking, RepairImpactUnknown}, preview.Impact.Status)
			if strings.HasPrefix(tc.rule, "asyncapi-") || strings.HasPrefix(tc.rule, "arazzo-") || strings.HasPrefix(tc.rule, "json-schema-") {
				require.Equal(t, RepairImpactUnknown, preview.Impact.Status)
			}
		})
	}
	for _, rule := range repair.ReviewedRuleIDs() {
		require.True(t, covered[rule], "missing detector fixture for %s", rule)
	}
}

func TestReviewedOperationIDKeepsPayloadDataAndUpdatesLink(t *testing.T) {
	source := `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"operationId":"list pets","responses":{"200":{"description":"ok","links":{"next":{"operationId":"list pets"},"examples":{"operationId":"list pets"}},"content":{"application/json":{"example":{"operationId":"list pets"},"schema":{"type":"object","properties":{"operationId":{"type":"string","default":"list pets"}}}}}}}}}}}`
	rule := rulesets.GetAllBuiltInRules()["operation-operationId-valid-in-url"]
	review, err := NewFixReview(&RuleSetExecution{SpecFileName: "/tmp/ids.json", Spec: []byte(source), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}}, nil)
	require.NoError(t, err)
	defer review.Close()
	require.Len(t, review.Options(), 1)
	option := review.Options()[0]
	require.True(t, option.Available)
	require.False(t, option.Editable, "compound linked rename must not expose a single-value editor")
	preview, err := review.Preview([]FixChoice{{Index: option.Index}})
	require.NoError(t, err)
	defer preview.Close()
	var document map[string]any
	require.NoError(t, json.Unmarshal(preview.Candidate, &document))
	op := document["paths"].(map[string]any)["/pets"].(map[string]any)["get"].(map[string]any)
	require.Equal(t, "list_pets", op["operationId"])
	response := op["responses"].(map[string]any)["200"].(map[string]any)
	require.Equal(t, "list_pets", response["links"].(map[string]any)["next"].(map[string]any)["operationId"])
	require.Equal(t, "list_pets", response["links"].(map[string]any)["examples"].(map[string]any)["operationId"])
	media := response["content"].(map[string]any)["application/json"].(map[string]any)
	require.Equal(t, "list pets", media["example"].(map[string]any)["operationId"])
	require.Equal(t, "list pets", media["schema"].(map[string]any)["properties"].(map[string]any)["operationId"].(map[string]any)["default"])
}

func TestReviewedCombinedGeneratedTagsShareGlobalCollection(t *testing.T) {
	source := `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/cats":{"get":{"responses":{"200":{"description":"ok"}}}},"/dogs":{"get":{"responses":{"200":{"description":"ok"}}}}}}`
	rule := rulesets.GetAllBuiltInRules()["operation-tags"]
	review, err := NewFixReview(&RuleSetExecution{SpecFileName: "/tmp/tags.json", Spec: []byte(source), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}}, nil)
	require.NoError(t, err)
	defer review.Close()
	require.Len(t, review.Options(), 2)
	for _, option := range review.Options() {
		require.True(t, option.Available)
		draft, err := review.Draft(FixChoice{Index: option.Index})
		require.NoError(t, err)
		require.NotEqual(t, draft.Before, draft.After)
	}
	preview, err := review.Preview([]FixChoice{{Index: 0}, {Index: 1}})
	require.NoError(t, err)
	require.Equal(t, 2, preview.Fixed)
	require.Empty(t, preview.Results)
	preview.Close()
	for _, option := range review.Options() {
		preview, err := review.Preview([]FixChoice{{Index: option.Index}})
		require.NoError(t, err)
		preview.Close()
	}
}

func TestReviewedDuplicateDescriptionUsesPublicOperationContext(t *testing.T) {
	source := `{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{"/pets":{"get":{"description":"Manage pets.","responses":{"200":{"description":"ok"}}},"post":{"description":"Manage pets.","responses":{"201":{"description":"created"}}}}}}`
	rule := rulesets.GetAllBuiltInRules()["description-duplication"]
	review, err := NewFixReview(&RuleSetExecution{SpecFileName: "/tmp/description.json", Spec: []byte(source), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}}, nil)
	require.NoError(t, err)
	defer review.Close()
	require.NotEmpty(t, review.Options())
	for _, option := range review.Options() {
		if !option.Available {
			continue
		}
		draft, err := review.Draft(FixChoice{Index: option.Index})
		require.NoError(t, err)
		require.NotContains(t, draft.After, "paths / ")
		require.True(t, strings.Contains(draft.After, "(POST /pets)") || strings.Contains(draft.After, "(GET /pets)"), draft.After)
	}
}
