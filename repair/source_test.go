package repair

import (
	"encoding/json"
	"github.com/pb33f/testify/require"
	"strings"
	"testing"
)

func TestApplyPreservesUntouchedSourceBytes(t *testing.T) {
	for name, source := range map[string]string{
		"blank lines and folded scalar": "title: Pets\n\ndescription: >\n  First line\n  second line\n\nschema:\n  enum: [a, a, b]\n\n# Keep this footer\n",
		"indentless sequence":           "info:\n    title: Pets\ntags:\n- name: pets\n  description: Pets\nschema:\n    enum: [a, a, b]\n",
		"mixed JSON spacing":            `{ "name" : "pets",  "schema": {"enum" : ["a", "a",  "b"]},"keep" : { } }`,
		"inline JSON and escapes":       "{\n    \"name\": \"caf\\u00e9\",\n    \"empty\": { },\n    \"schema\": {\"enum\": [\"a\", \"a\", \"b\"]}\n}\n",
	} {
		t.Run(name, func(t *testing.T) {
			for _, ending := range []string{"\n", "\r\n"} {
				data := []byte(strings.ReplaceAll(source, "\n", ending))
				plan := NewPlan("source", data)
				plan.Operations = []Operation{{Op: "remove", Path: "/schema/enum/1"}}
				got, err := Apply("source", data, plan)
				require.NoError(t, err)
				want := strings.Replace(string(data), "[a, a, b]", "[a, b]", 1)
				want = strings.Replace(want, `["a", "a", "b"]`, `["a", "b"]`, 1)
				want = strings.Replace(want, `["a", "a",  "b"]`, `["a",  "b"]`, 1)
				require.Equal(t, want, string(got))
			}
		})
	}
}
func TestApplyAddsOnlyRequestedField(t *testing.T) {
	source := []byte("info:\n  title: Pets\n\n  description: >\n    First line\n    second line\n\npaths: {}\n")
	plan := NewPlan("source", source)
	plan.Operations = []Operation{{Op: "add", Path: "/info/version", Value: json.RawMessage(`"v1"`)}}
	got, err := Apply("source", source, plan)
	require.NoError(t, err)
	require.Contains(t, string(got), "description: >\n    First line\n    second line\n")
	require.Contains(t, string(got), "  version: v1")
	require.Contains(t, string(got), "\n\npaths: {}")
}
func TestReviewedPreservesPathOrderValues(t *testing.T) {
	source := []byte("paths:\n  /{id}:\n    post:\n      summary: 'Keep quotes'\n      description: |\n        multi\n        line\n    get: {}\n  /pets: {}\n")
	plan, err := ReviewedPlan("source", source, []Finding{{RuleID: "paths-specificity-order", Pointer: "/paths"}})
	require.NoError(t, err)
	got, err := Apply("source", source, plan)
	require.NoError(t, err)
	require.True(t, strings.Index(string(got), "/pets") < strings.Index(string(got), "/{id}"))
	require.Contains(t, string(got), "post:\n      summary: 'Keep quotes'\n      description: |\n        multi\n        line\n    get: {}")
}
func TestReviewedMarkdownChangesOnlyMatchedTokens(t *testing.T) {
	for _, tc := range []struct{ rule, before, after string }{
		{"no-eval-in-markdown", "Tom &amp; Jerry eval(x) <br>", "Tom &amp; Jerry eval&#40;x) <br>"},
		{"no-script-tags-in-markdown", "> quote\n<br> <https://example.com> <SCRIPT>x</SCRIPT>", "> quote\n<br> <https://example.com> &lt;SCRIPT>x&lt;/SCRIPT>"},
	} {
		source, _ := json.Marshal(map[string]string{"description": tc.before})
		plan, err := ReviewedPlan("source", source, []Finding{{RuleID: tc.rule, Pointer: "/description"}})
		require.NoError(t, err)
		got, err := Apply("source", source, plan)
		require.NoError(t, err)
		var decoded map[string]string
		require.NoError(t, json.Unmarshal(got, &decoded))
		require.Equal(t, tc.after, decoded["description"])
	}
}
func TestReviewedInlineRefDeclinesOperationID(t *testing.T) {
	source := []byte(`{"paths":{"/a":{"get":{"operationId":"getA","responses":{}}},"/b":{"get":{"$ref":"#/paths/~1a/get"}}}}`)
	plan, err := ReviewedPlan("source", source, []Finding{{RuleID: "path-item-refs", Pointer: "/paths/~1b/get"}})
	require.NoError(t, err)
	require.Empty(t, plan.Targets)
	require.Contains(t, plan.Declined[0].Reason, "operationId")
}
func TestPolicyRejectsInvalidValueTypes(t *testing.T) {
	for _, value := range []string{"null", "2024-01-01", "42", "[text]", "&value text"} {
		source := "version: 1\npolicies:\n- id: docs\n  rule: info-description\n  diagnostic: description-missing\n  scope: {source: api.yaml, pointer: /info}\n  value: " + value + "\n"
		_, err := ParsePolicy([]byte(source), t.TempDir(), nil)
		require.Error(t, err, value)
	}
}

func TestSourceEditsRemainValidAcrossCollectionStyles(t *testing.T) {
	for _, source := range []string{
		"description: Hello, world [today]\nschema:\n  enum: [a, a, b]\n",
		"description: Hello\n  long world\nschema:\n  enum: [a, a, b]\n",
		"info:\n    title: Pets\n    contact: {}\npaths: {}\n",
		"tags:\n- name: pets\n  description: Pets\n",
	} {
		t.Run(source, func(t *testing.T) {
			data := []byte(source)
			plan := NewPlan("source", data)
			switch {
			case strings.HasPrefix(source, "description:"):
				plan.Operations = []Operation{{Op: "replace", Path: "/description", Value: json.RawMessage(`"Updated"`)}}
			case strings.HasPrefix(source, "info:"):
				plan.Operations = []Operation{{Op: "add", Path: "/info/contact/name", Value: json.RawMessage(`"Team"`)}}
			default:
				plan.Operations = []Operation{{Op: "add", Path: "/tags/-", Value: json.RawMessage(`{"name":"orders","description":"Orders"}`)}}
			}
			_, err := Apply("source", data, plan)
			require.NoError(t, err)
		})
	}
}

func TestPolicyCatalogIgnoresLinkOperationIDs(t *testing.T) {
	base := t.TempDir()
	source := []byte("paths:\n  /pets:\n    get:\n      operationId: listPets\ncomponents:\n  links:\n    next:\n      operationId: listPets\n")
	policy, err := ParsePolicy([]byte("version: 1\npolicies:\n- id: docs\n  rule: operation-description\n  diagnostic: description-missing\n  scope: {source: api.yaml, pointer: /paths/~1pets/get}\n  catalog: docs.yaml\n"), base, map[string][]byte{base + "/docs.yaml": []byte("api.yaml:\n  listPets: List pets.\n")})
	require.NoError(t, err)
	plan, err := ApplyPolicy(NewPlan(base+"/api.yaml", source), source, []Finding{{RuleID: "operation-description", Pointer: "/paths/~1pets/get"}}, policy)
	require.NoError(t, err)
	require.Len(t, plan.Targets, 1)
}

func TestApplyPreservesFlowCommentBeforeSurvivingItem(t *testing.T) {
	source := []byte("values: [a,  a, # Keep b comment\n  b]\n")
	plan := NewPlan("source", source)
	plan.Operations = []Operation{{Op: "remove", Path: "/values/1"}}
	got, err := Apply("source", source, plan)
	require.NoError(t, err)
	require.Equal(t, "values: [a, # Keep b comment\n  b]\n", string(got))
}
