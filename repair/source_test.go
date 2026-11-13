// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"encoding/json"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
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

func TestApplyDropsLineCommentOnRemovedFlowItem(t *testing.T) {
	source := []byte("values: [a,  a, # about second a\n  b]\n")
	plan := NewPlan("source", source)
	plan.Operations = []Operation{{Op: "remove", Path: "/values/1"}}
	got, err := Apply("source", source, plan)
	require.NoError(t, err)
	require.Equal(t, "values: [a,\n  b]\n", string(got))
}

func TestSourcePreservesTaggedScalarsAndDocumentMarkers(t *testing.T) {
	source := []byte("---\ndate: 2024-01-01\nfoo: old\n...\n")
	plan := NewPlan("source", source)
	plan.Operations = []Operation{{Op: "replace", Path: "/foo", Value: json.RawMessage(`"new"`)}}
	got, err := Apply("source", source, plan)
	require.NoError(t, err)
	require.Equal(t, strings.Replace(string(source), "old", "new", 1), string(got))
}
func TestSourceAppendPreservesKeepChompValue(t *testing.T) {
	source := []byte("info:\n  description: |+\n    Text\n\n\n")
	plan := NewPlan("source", source)
	plan.Operations = []Operation{{Op: "add", Path: "/info/title", Value: json.RawMessage(`"Pets"`)}}
	got, err := Apply("source", source, plan)
	require.NoError(t, err)
	require.Equal(t, "info:\n  description: |+\n    Text\n\n\n  title: Pets\n", string(got))
}
func TestSourceQuotesNewFlowScalarWhenRequired(t *testing.T) {
	source := []byte("parameter: {name: id, in: path}\n")
	plan := NewPlan("source", source)
	plan.Operations = []Operation{{Op: "add", Path: "/parameter/description", Value: json.RawMessage(`"Path parameter for /pets/{id}."`)}}
	got, err := Apply("source", source, plan)
	require.NoError(t, err)
	require.Contains(t, string(got), `description: "Path parameter for /pets/{id}."`)
}
func TestNewPolicyRejectsCyclicCallerNodes(t *testing.T) {
	node := yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	node.Content = []*yaml.Node{&node}
	_, err := NewPolicy([]PolicyEntry{{Value: node}}, t.TempDir())
	require.Error(t, err)
}

func TestSourcePreservesCommentedDocumentEndMarker(t *testing.T) {
	for _, ending := range []string{"\n", "\r\n"} {
		source := []byte(strings.ReplaceAll("---\nfoo: old\n... # document end\n", "\n", ending))
		plan := NewPlan("source", source)
		plan.Operations = []Operation{{Op: "replace", Path: "/foo", Value: json.RawMessage(`"new"`)}}
		got, err := Apply("source", source, plan)
		require.NoError(t, err)
		require.Equal(t, strings.Replace(string(source), "old", "new", 1), string(got))
	}
}

func TestBlockEntryCommentOwnership(t *testing.T) {
	for _, test := range []struct {
		name, source, want string
		op                 Operation
	}{
		{"middle", "a: 1\nb: 2 # about b\nc: 3\n", "a: 1\nc: 3\n", Operation{Op: "remove", Path: "/b"}},
		{"last", "a: 1 # about a\nb: 2 # about b\n", "a: 1 # about a\n", Operation{Op: "remove", Path: "/b"}},
		{"first", "# about a\na: 1\n# about b\nb: 2\n", "# about b\nb: 2\n", Operation{Op: "remove", Path: "/a"}},
		{"foot", "a: 1\n# foot a\n\nb: 2\n", "b: 2\n", Operation{Op: "remove", Path: "/a"}},
		{"sequence", "- name: b # bee\n# about a\n- name: a # ay\n", "# about a\n- name: a # ay\n- name: b # bee\n", Operation{Op: "move", From: "/1", Path: "/0"}},
		{"nested", "outer:\n  a: 1 # about a\n  # about b\n  b: 2\n", "outer:\n  # about b\n  b: 2\n", Operation{Op: "remove", Path: "/outer/a"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, newline := range []string{"\n", "\r\n"} {
				source := []byte(strings.ReplaceAll(test.source, "\n", newline))
				plan := NewPlan("source", source)
				plan.Operations = []Operation{test.op}
				out, err := Apply("source", source, plan)
				require.NoError(t, err)
				require.Equal(t, strings.ReplaceAll(test.want, "\n", newline), string(out))
			}
		})
	}
}

func TestSourceChecksCommentOwnershipAfterRendering(t *testing.T) {
	for _, comment := range []func(*yaml.Node){func(n *yaml.Node) { n.HeadComment = "# head" }, func(n *yaml.Node) { n.LineComment = "# line" }, func(n *yaml.Node) { n.FootComment = "# foot" }} {
		data := []byte("key: value\n")
		doc, err := ParseDocument(data)
		require.NoError(t, err)
		root := doc.Content[0]
		after := clone(doc)
		comment(after.Content[0].Content[0])
		out, err := renderSource(root, after, data, false, utils.CaptureRepairYAMLFormat(root))
		require.ErrorContains(t, err, "differs from the requested edits")
		require.Nil(t, out)
	}
}

func TestApplyPreservesBOMAndOriginalFormat(t *testing.T) {
	for _, test := range []struct {
		source, want string
		op           Operation
	}{
		{"title: Pets\n", "title: Dogs\n", Operation{Op: "replace", Path: "/title", Value: json.RawMessage(`"Dogs"`)}},
		{`{"title":"Pets"}`, `{"title":"Pets","tags":["pets"]}`, Operation{Op: "add", Path: "/tags", Value: json.RawMessage(`["pets"]`)}},
	} {
		source := []byte("\ufeff" + test.source)
		plan := NewPlan("source", source)
		plan.Operations = []Operation{test.op}
		out, err := Apply("source", source, plan)
		require.NoError(t, err)
		require.Equal(t, "\ufeff"+test.want, string(out))
	}
}
