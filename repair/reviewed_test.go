// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestReviewedRecipes(t *testing.T) {
	cases := []struct{ name, rule, code, source, pointer, want string }{
		{"jwt", "owasp-jwt-best-practices", "", `{"bearerFormat":"JWT"}`, "", `{"bearerFormat":"JWT","description":"Implementations should follow the JWT security guidance in RFC8725 (https://www.rfc-editor.org/rfc/rfc8725)."}`},
		{"script", "no-script-tags-in-markdown", "", `{"description":"<script>alert(1)</script>"}`, "/description", `{"description":"&lt;script>alert(1)&lt;/script>"}`},
		{"arazzoScript", "arazzo-no-script-tags-in-markdown", "", `{"description":"<SCRIPT>x</SCRIPT>"}`, "/description", `{"description":"&lt;SCRIPT>x&lt;/SCRIPT>"}`},
		{"serverSlash", "oas3-server-trailing-slash", "", `{"url":"https://api.test/v1/"}`, "/url", `{"url":"https://api.test/v1"}`},
		{"asyncServerSlash", "asyncapi-3-server-no-trailing-slash", "", `{"pathname":"/v1/"}`, "/pathname", `{"pathname":"/v1"}`},
		{"channelSlash", "asyncapi-3-channel-no-trailing-slash", "", `{"address":"orders/"}`, "", `{"address":"orders"}`},
		{"header", "asyncapi-3-headers-schema-type-object", "", `{"headers":{"properties":{"id":{"type":"string"}}}}`, "/headers", `{"headers":{"properties":{"id":{"type":"string"}},"type":"object"}}`},
		{"asyncTags", "asyncapi-3-tags", "", `{"info":{"title":"Events"}}`, "/info", `{"info":{"title":"Events","tags":[{"name":"Events","description":"Operations for Events."}]}}`},
		{"operationTags", "operation-tags", "", `{"paths":{"/pets":{"get":{"responses":{}}}}}`, "/paths/~1pets/get", `{"paths":{"/pets":{"get":{"responses":{},"tags":["pets"]}}},"tags":[{"name":"pets","description":"Operations for pets."}]}`},
		{"globalTag", "operation-tag-defined", "", `{"paths":{"/pets":{"get":{"tags":["pets"]}}}}`, "/paths/~1pets/get/tags/0", `{"paths":{"/pets":{"get":{"tags":["pets"]}}},"tags":[{"name":"pets","description":"Operations for pets."}]}`},
		{"nullable", "nullable-enum-contains-null", "", `{"schema":{"type":"string","nullable":true,"enum":["a"]}}`, "/schema", `{"schema":{"type":"string","nullable":true,"enum":["a",null]}}`},
		{"required", "json-schema-required-properties-defined", "", `{"required":["a","b","a"]}`, "/required/2", `{"required":["a","b"]}`},
		{"tags", "operation-singular-tag", "", `{"tags":["a","a"]}`, "/tags", `{"tags":["a"]}`},
		{"anyOf", "oas2-anyOf", "", `{"swagger":"2.0","schema":{"anyOf":[{"type":"string"}]}}`, "/schema/anyOf/0", `{"swagger":"2.0","schema":{"allOf":[{"type":"string"}]}}`},
		{"oneOf", "oas2-oneOf", "", `{"swagger":"2.0","schema":{"oneOf":[{"type":"string"}]}}`, "/schema/oneOf/0", `{"swagger":"2.0","schema":{"allOf":[{"type":"string"}]}}`},
		{"host", "oas2-host-trailing-slash", "", `{"host":"api.test///"}`, "/host", `{"host":"api.test"}`},
		{"type", "oas-missing-type", "", `{"schema":{"maxLength":30}}`, "/schema", `{"schema":{"maxLength":30,"type":"string"}}`},
		{"enum", "typed-enum", "", `{"type":"integer","enum":["2",3]}`, "/enum/0", `{"type":"integer","enum":[2,3]}`},
		{"boolean", "typed-enum", "", `{"type":"boolean","enum":["false",true]}`, "/enum", `{"type":"boolean","enum":[false,true]}`},
		{"format", "owasp-integer-format", "", `{"type":"integer","minimum":0,"maximum":4000}`, "/type", `{"type":"integer","minimum":0,"maximum":4000,"format":"int32"}`},
		{"bounds", "owasp-integer-limit", "", `{"type":"integer","format":"int32","minimum":0}`, "/type", `{"type":"integer","format":"int32","minimum":0,"maximum":2147483647}`},
		{"https", "owasp-auth-urls-https", "", `{"authorizationUrl":"http://api.test/oauth"}`, "/authorizationUrl", `{"authorizationUrl":"https://api.test/oauth"}`},
		{"server", "owasp-security-hosts-https-oas3", "", `{"url":"http://api.test/v1"}`, "", `{"url":"https://api.test/v1"}`},
		{"closed", "owasp-no-additionalProperties", "", `{"type":"object","additionalProperties":true}`, "/additionalProperties", `{"type":"object","additionalProperties":false}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := []Finding{{RuleID: tc.rule, Code: tc.code, Pointer: tc.pointer}}
			plan, err := ReviewedPlan("test.json", []byte(tc.source), f)
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Targets) != 1 {
				t.Fatalf("no proposal: %+v", plan.Declined)
			}
			again, err := ReviewedPlan("test.json", []byte(tc.source), f)
			if err != nil {
				t.Fatal(err)
			}
			a, _ := json.Marshal(plan)
			b, _ := json.Marshal(again)
			if !bytes.Equal(a, b) {
				t.Fatal("proposal is not deterministic")
			}
			got, err := Apply("test.json", []byte(tc.source), plan)
			if err != nil {
				t.Fatal(err)
			}
			var actual, expected any
			if err := json.Unmarshal(got, &actual); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.want), &expected); err != nil {
				t.Fatal(err)
			}
			a, _ = json.Marshal(actual)
			b, _ = json.Marshal(expected)
			if !bytes.Equal(a, b) {
				t.Fatalf("got %s want %s", got, tc.want)
			}
			legacy, err := BuiltinPlan("test.json", []byte(tc.source), f)
			if err != nil {
				t.Fatal(err)
			}
			if len(legacy.Targets) != 0 {
				t.Fatal("review-only recipe entered unattended provider")
			}
		})
	}
}

func TestReviewedRecipesDeclineAmbiguousChanges(t *testing.T) {
	cases := []struct{ rule, source, pointer string }{
		{"operation-singular-tag", `{"tags":["a","b","a"]}`, "/tags"},
		{"typed-enum", `{"type":"integer","enum":["001"]}`, "/enum"},
		{"oas-missing-type", `{"minLength":1,"minimum":1}`, ""},
		{"oas2-anyOf", `{"swagger":"2.0","anyOf":[{"type":"string"},{"type":"integer"}]}`, "/anyOf/0"},
		{"oas2-oneOf", `{"swagger":"2.0","oneOf":[{"type":"string"}],"allOf":[{}]}`, "/oneOf/0"},
		{"nullable-enum-contains-null", `{"type":"string","enum":["a"]}`, ""},
		{"oas2-host-trailing-slash", `{"host":"api.test/path/"}`, "/host"},
		{"owasp-integer-limit", `{"type":"integer"}`, "/type"},
		{"owasp-integer-limit", `{"type":"integer","format":"int32","minimum":99999999999}`, "/type"},
		{"owasp-integer-format", `{"type":"integer","minimum":0}`, "/type"},
		{"owasp-auth-urls-https", `{"url":"http://api.test:8080"}`, "/url"},
		{"owasp-no-additionalProperties", `{"type":"object","additionalProperties":{"type":"string"}}`, "/additionalProperties"},
	}
	for _, tc := range cases {
		t.Run(tc.rule, func(t *testing.T) {
			p, err := ReviewedPlan("t.json", []byte(tc.source), []Finding{{RuleID: tc.rule, Pointer: tc.pointer}})
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Operations) != 0 || len(p.Targets) != 0 || len(p.Declined) != 1 {
				t.Fatalf("ambiguous proposal admitted: %+v", p)
			}
		})
	}
}

func TestReviewedPreservesCommentsAndRejectsCommentDeletion(t *testing.T) {
	source := []byte("schema:\n  # retained\n  type: string\n  nullable: true\n  enum: [a]\n")
	p, err := ReviewedPlan("t.yaml", source, []Finding{{RuleID: "nullable-enum-contains-null", Pointer: "/schema"}})
	if err != nil {
		t.Fatal(err)
	}
	out, err := Apply("t.yaml", source, p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("# retained")) {
		t.Fatalf("comment lost: %s", out)
	}
	source = []byte("required:\n  - a\n  - a # explains duplicate\n")
	p, err = ReviewedPlan("t.yaml", source, []Finding{{RuleID: "json-schema-required-properties-defined", Pointer: "/required/1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Targets) != 0 {
		t.Fatal("commented duplicate was deleted")
	}
}

func TestReviewedDeduplicatesProposalsAndKeepsUnavailableFindings(t *testing.T) {
	source := []byte(`{"required":["a","a","a"]}`)
	p, err := ReviewedPlan("t.json", source, []Finding{{RuleID: "json-schema-required-properties-defined", Pointer: "/required/1"}, {RuleID: "json-schema-required-properties-defined", Pointer: "/required/2"}, {RuleID: "nullable-enum-contains-null", Pointer: "", Unavailable: "external source"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Targets) != 2 || len(p.Operations) != 2 || len(p.Declined) != 1 {
		t.Fatalf("bad grouping: %+v", p)
	}
	if _, err := Apply("t.json", source, p); err != nil {
		t.Fatal(err)
	}
}

func TestReviewedIndependentTagsShareGlobalCreation(t *testing.T) {
	source := []byte(`{"paths":{"/a":{"get":{}},"/b":{"get":{}}}}`)
	p, err := ReviewedPlan("t.json", source, []Finding{{RuleID: "operation-tags", Pointer: "/paths/~1a/get"}, {RuleID: "operation-tags", Pointer: "/paths/~1b/get"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Targets) != 2 || len(p.Declined) != 0 {
		t.Fatalf("independent global tag writes were declined: %+v", p)
	}
	if _, err := Apply("t.json", source, p); err != nil {
		t.Fatal(err)
	}
}

func TestReviewedNewRecipeGuards(t *testing.T) {
	cases := []struct{ rule, source, pointer string }{
		{"operation-operationId-unique", `{"paths":{"/a":{"get":{"operationId":"same"}},"/b":{"get":{"operationId":"same"}}},"components":{"links":{"next":{"operationId":"same"}}}}`, "/paths/~1b/get"},
		{"paths-kebab-case", `{"paths":{"/myPets":{},"/my-pets":{}}}`, "/paths/~1myPets"},
		{"paths-kebab-case", `{"paths":{"/myPets":{}},"components":{"schemas":{"X":{"$ref":"#/components/schemas/Y"}}}}`, "/paths/~1myPets"},
		{"path-not-include-query", `{"paths":{"/pets?limit=1":{}}}`, "/paths/~1pets?limit=1"},
		{"camel-case-properties", `{"type":"object","properties":{"first_name":{"type":"string"},"firstName":{"type":"string"}}}`, "/properties/first_name"},
		{"camel-case-properties", `{"type":"object","properties":{"first_name":{"type":"string"}},"dependentRequired":{"first_name":["other"]}}`, "/properties/first_name"},
		{"required-fields-defined", `{"type":"object","required":["id"],"example":{"id":{}}}`, "/required"},
		{"asyncapi-channel-parameters", `{"address":"orders/{id}","parameters":{"unused":{}}}`, ""},
		{"oas3-unused-component", `{"components":{"schemas":{"X":{"type":"string"}}},"schema":{"$ref":"#/components/schemas/X"}}`, "/components/schemas/X"},
		{"oas3-unused-component", `{"components":{"schemas":{"X":{"type":"string"}}},"schema":{"$ref":"other.yaml"}}`, "/components/schemas/X"},
		{"oas3-unused-component", `{"components":{"securitySchemes":{"X":{"type":"http","scheme":"bearer"}}}}`, "/components/securitySchemes/X"},
		{"oas2-discriminator", `{"discriminator":"kind","properties":{"kind":{"type":"integer"}}}`, "/discriminator"},
		{"json-schema-composition-sanity", `{"allOf":[{"type":"string"},{"type":"integer"}]}`, "/allOf/1"},
		{"json-schema-title-description-type", `{"type":"object"}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.rule, func(t *testing.T) {
			plan, err := ReviewedPlan("test.json", []byte(tc.source), []Finding{{RuleID: tc.rule, Pointer: tc.pointer}})
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Targets) != 0 || len(plan.Operations) != 0 {
				t.Fatalf("unsafe proposal admitted: %+v", plan)
			}
		})
	}
}

func TestReviewedProposalGroupsPreserveCompoundDependencies(t *testing.T) {
	source := []byte(`{"paths":{"/pets":{"get":{"operationId":"list pets","responses":{"200":{"links":{"next":{"operationId":"list pets"}}}}}}}}`)
	plan, err := ReviewedPlan("test.json", source, []Finding{{RuleID: "operation-operationId-valid-in-url", Pointer: "/paths/~1pets/get/operationId"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 2 || len(plan.ReviewedProposals["operation-operationId-valid-in-url\x00/paths/~1pets/get/operationId"]) != 2 {
		t.Fatalf("compound edit group lost: %+v", plan)
	}
	source = []byte(`{"type":"string","nullable":true,"enum":["a"]}`)
	plan, err = ReviewedPlan("test.json", source, []Finding{{RuleID: "nullable-enum-contains-null", Pointer: ""}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.ReviewedProposals["nullable-enum-contains-null\x00"]) != 1 || plan.ReviewedProposals["nullable-enum-contains-null\x00"][0].Path != "/enum/-" {
		t.Fatal("single proposed value unavailable")
	}
}

func TestReviewedNativeDedupePreservesDistinctLargeNumbers(t *testing.T) {
	for _, tc := range []struct{ rule, source, pointer string }{
		{"json-schema-valid", `{"enum":[9007199254740992,9007199254740993]}`, "/enum"},
		{"json-schema-composition-sanity", `{"oneOf":[{"const":9007199254740992},{"const":9007199254740993}]}`, "/oneOf/1"},
	} {
		plan, err := ReviewedPlan("test.json", []byte(tc.source), []Finding{{RuleID: tc.rule, Pointer: tc.pointer}})
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.Operations) > 0 {
			t.Fatalf("distinct values collapsed: %+v", plan)
		}
	}
}

func TestReviewedNativeDedupeDoesNotEditExamplePayload(t *testing.T) {
	source := []byte(`{"type":"object","required":["id","id"],"example":{"required":["keep","keep"]}}`)
	plan, err := ReviewedPlan("test.json", source, []Finding{{RuleID: "json-schema-valid", Pointer: "/required"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) != 1 || plan.Operations[0].Path != "/required/1" {
		t.Fatalf("unrelated payload changed: %+v", plan)
	}
}

func TestReviewedSharesIdenticalValueEditAcrossProposalGroups(t *testing.T) {
	source := []byte(`{"paths":{"/pets":{"get":{},"post":{"tags":["pets"]}}}}`)
	findings := []Finding{{RuleID: "operation-tags", Pointer: "/paths/~1pets/get"}, {RuleID: "operation-tag-defined", Pointer: "/paths/~1pets/post/tags/0"}}
	plan, err := ReviewedPlan("test.json", source, findings)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Targets) != 2 || len(plan.Operations) != 3 || len(plan.Declined) != 0 {
		t.Fatalf("shared value was treated as a conflict: %+v", plan)
	}
	if len(plan.ReviewedProposals["operation-tags\x00/paths/~1pets/get"]) != 3 || len(plan.ReviewedProposals["operation-tag-defined\x00/paths/~1pets/post/tags/0"]) != 2 {
		t.Fatal("original per-finding groups were lost")
	}
	if _, err := Apply("test.json", source, plan); err != nil {
		t.Fatal(err)
	}
}

func TestReviewedNativeAndMigrationGuards(t *testing.T) {
	cases := []struct{ rule, source, pointer string }{
		{"license-url", `{"info":{"license":{"name":"Commercial"}}}`, "/info/license"},
		{"info-license-spdx", `{"info":{"license":{"name":"MIT","identifier":"MIT","url":"https://example.com/custom-terms"}}}`, "/info/license"},
		{"migrate-zally-ignore", `{"x-zally-ignore":[123]}`, "/x-zally-ignore"},
		{"json-schema-type-constraint-compatible", `{"type":"string","$anchor":"value","minimum":1}`, "/minimum"},
		{"json-schema-type-constraint-compatible", `{"type":["string","number"],"minimum":1}`, "/minimum"},
		{"paths-specificity-order", `{"paths":{"/pets/{id}.json":{},"/pets/special.json":{}}}`, "/paths/~1pets~1special.json"},
	}
	for _, tc := range cases {
		t.Run(tc.rule, func(t *testing.T) {
			p, err := ReviewedPlan("test.json", []byte(tc.source), []Finding{{RuleID: tc.rule, Pointer: tc.pointer}})
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Targets) > 0 {
				t.Fatalf("ambiguous change was proposed: %+v", p)
			}
		})
	}
}

func TestReviewedDeclinesHaveStablePrecedence(t *testing.T) {
	for _, tc := range []struct{ rule, pointer, source, reason string }{
		{"oas3-unused-component", "/components/schemas/X", `{"components":{"schemas":{"X":{"type":"string"}}},"local":{"$ref":"#/components/schemas/X"},"external":{"$ref":"other.yaml"},"scoped":{"$id":"x"}}`, "schema has an incoming reference"},
		{"oas3-unused-component", "/components/schemas/X", `{"components":{"schemas":{"X":{"type":"string"}}},"external":{"$ref":"other.yaml"},"scoped":{"$id":"x"}}`, "schema has scoped or implicit reference semantics"},
		{"arazzo-advisory", "/workflows/0/workflowId", `{"workflows":[{"workflowId":"bad id","dependsOn":["read"],"outputs":{"id":"$request.body"}}]}`, "workflow dependencies require a coordinated naming migration"},
	} {
		t.Run(tc.rule, func(t *testing.T) {
			for range 64 {
				plan, err := ReviewedPlan("source.json", []byte(tc.source), []Finding{{RuleID: tc.rule, Pointer: tc.pointer}})
				if err != nil || len(plan.Declined) != 1 || plan.Declined[0].Reason != tc.reason {
					t.Fatalf("got %+v and %v, want %q", plan.Declined, err, tc.reason)
				}
			}
		})
	}
}

func TestRepresentativeExamples(t *testing.T) {
	for _, tc := range []struct{ name, schema, want string }{
		{"enum", `{"type":"string","enum":["paid","pending"]}`, `"paid"`},
		{"constant", `{"const":"order-created"}`, `"order-created"`},
		{"default before format", `{"type":"string","format":"email","default":"support@example.org"}`, `"support@example.org"`},
		{"invalid authored hint", `{"type":"string","default":8,"format":"email"}`, `"user@example.com"`},
		{"uri", `{"type":"string","format":"uri"}`, `"https://example.com/resources/1"`},
		{"timestamp", `{"type":"string","format":"date-time"}`, `"2026-01-15T12:00:00Z"`},
		{"union from authored branch", `{"oneOf":[{"type":"string"},{"type":"integer","const":17}]}`, `17`},
		{"branch must satisfy full schema", `{"allOf":[{"const":"paid"},{"const":"pending"}]}`, ""},
		{"member hints", `{"type":"object","properties":{"first":{"type":"string"},"tags":{"type":"array","items":{"type":"string","enum":["public"]}}}}`, `{"tags":["public"]}`},
		{"unknown string", `{"type":"string"}`, ""},
		{"object without fields", `{"type":"object","additionalProperties":true}`, ""},
		{"required free-form field", `{"type":"object","required":["title"],"properties":{"title":{"type":"string"}}}`, ""},
		{"free-form items", `{"type":"array","items":{"type":"string"}}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := ParseDocument([]byte(tc.schema))
			if err != nil {
				t.Fatal(err)
			}
			root := doc.Content[0]
			ops, _, reason := reviewedExample(root, Finding{RuleID: "oas3-valid-schema-example"}, sourceNodes(root))
			if tc.want == "" {
				if len(ops) > 0 || reason == "" {
					t.Fatalf("expected decline: %v, %s", ops, reason)
				}
			} else if len(ops) != 1 || string(ops[0].Value) != tc.want {
				t.Fatalf("expected %s, got %+v, %s", tc.want, ops, reason)
			}
		})
	}
}
