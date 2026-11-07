// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestSequentialPatch(t *testing.T) {
	source := []byte(`{"items":["a","b","c"],"n":9007199254740993}`)
	plan := NewPlan("api.json", source)
	plan.Operations = []Operation{{Op: "remove", Path: "/items/0"}, {Op: "replace", Path: "/items/1", Value: json.RawMessage(`"new"`)}, {Op: "test", Path: "/n", Value: json.RawMessage(`9007199254740993.0`)}, {Op: "move", From: "/items/1", Path: "/items/0"}, {Op: "copy", From: "/n", Path: "/copy"}}
	out, err := Apply("api.json", source, plan)
	require.NoError(t, err)
	require.JSONEq(t, `{"items":["new","b"],"n":9007199254740993,"copy":9007199254740993}`, string(out))
	require.Equal(t, `{"items":["a","b","c"],"n":9007199254740993}`, string(source))
}

func TestPatchPreconditionsAndRollback(t *testing.T) {
	source := []byte(`{"items":[1,2],"a/b":null}`)
	for _, tc := range []struct {
		name   string
		ops    []Operation
		absent []string
	}{
		{"failed test", []Operation{{Op: "remove", Path: "/items/0"}, {Op: "test", Path: "/items/0", Value: json.RawMessage(`1`)}}, nil},
		{"missing test", []Operation{{Op: "test", Path: "/missing", Value: json.RawMessage(`null`)}}, nil},
		{"bad escape", []Operation{{Op: "remove", Path: "/a~2b"}}, nil},
		{"descendant move", []Operation{{Op: "move", From: "/items", Path: "/items/0"}}, nil},
		{"leading zero", []Operation{{Op: "remove", Path: "/items/01"}}, nil},
		{"unknown op", []Operation{{Op: "reorder", Path: "/items"}}, nil},
		{"present null", nil, []string{"/a~1b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPlan("s", source)
			p.Operations = tc.ops
			p.Absent = tc.absent
			out, err := Apply("s", source, p)
			require.Error(t, err)
			require.Nil(t, out)
		})
	}
	p := NewPlan("s", source)
	p.Absent = []string{"/new"}
	p.Operations = []Operation{{Op: "add", Path: "/new", Value: json.RawMessage(`false`)}}
	out, err := Apply("s", source, p)
	require.NoError(t, err)
	require.True(t, json.Valid(out))
	_, err = Apply("other", source, p)
	require.Error(t, err)
	_, err = Apply("s", append(source, ' '), p)
	require.Error(t, err)
}

func TestYAMLKeyRetagAndPreservation(t *testing.T) {
	source := []byte("# API\nresponses:\n  200: # success\n    description: ok\n")
	p := NewPlan("s", source)
	p.KeyTags = []KeyTagEdit{{Parent: "/responses", Index: 0, Key: "200", Tag: "!!int"}}
	out, err := Apply("s", source, p)
	require.NoError(t, err)
	require.Contains(t, string(out), `"200": # success`)
	require.True(t, strings.HasPrefix(string(out), "# API"))
	p.KeyTags[0].Key = "201"
	_, err = Apply("s", source, p)
	require.Error(t, err)
}

func TestPatchRefusesAmbiguousYAML(t *testing.T) {
	for _, data := range []string{"x: 1\nx: 2\n", "x: &v [1]\ny: *v\n"} {
		p := NewPlan("s", []byte(data))
		_, err := Apply("s", []byte(data), p)
		require.Error(t, err)
	}
}

func TestPatchDataModelCases(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		ops                []Operation
	}{
		{"append", `{"a":[1]}`, `{"a":[1,2]}`, []Operation{{Op: "add", Path: "/a/-", Value: json.RawMessage(`2`)}}},
		{"replace root", `{"old":true}`, `[null,false]`, []Operation{{Op: "replace", Path: "", Value: json.RawMessage(`[null,false]`)}}},
		{"add replaces member", `{"a":1}`, `{"a":2}`, []Operation{{Op: "add", Path: "/a", Value: json.RawMessage(`2`)}}},
		{"escaped keys", `{"a/b":{"~key":1}}`, `{"a/b":{"~key":2}}`, []Operation{{Op: "replace", Path: "/a~1b/~0key", Value: json.RawMessage(`2`)}}},
		{"move forward uses post remove index", `["a","b","c"]`, `["b","c","a"]`, []Operation{{Op: "move", From: "/0", Path: "/2"}}},
		{"copy independent", `{"a":{"x":1}}`, `{"a":{"x":1},"b":{"x":2}}`, []Operation{{Op: "copy", From: "/a", Path: "/b"}, {Op: "replace", Path: "/b/x", Value: json.RawMessage(`2`)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPlan("s", []byte(tc.source))
			p.Operations = tc.ops
			out, err := Apply("s", []byte(tc.source), p)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(out))
		})
	}
	source := []byte(`{"n":9007199254740993}`)
	p := NewPlan("s", source)
	p.Operations = []Operation{{Op: "test", Path: "/n", Value: json.RawMessage(`9007199254740992`)}}
	_, err := Apply("s", source, p)
	require.Error(t, err)
}

func TestRepairRejectsStreamsAndExcessiveNesting(t *testing.T) {
	for _, source := range []string{"a: 1\n---\nb: 2", strings.Repeat("[", 257) + "0" + strings.Repeat("]", 257)} {
		_, err := ParseDocument([]byte(source))
		require.Error(t, err)
	}
}

func TestConflictIndexOwnsArrays(t *testing.T) {
	parsed, err := ParseDocument([]byte(`{"items":[{"a":1},{"b":2}]}`))
	require.NoError(t, err)
	nodes := sourceNodes(parsed.Content[0])
	require.Equal(t, "/items/0/a", editFootprint("/items/0/a", nodes))
	require.Equal(t, "/items", editFootprint("/items/0", nodes))
	require.Equal(t, "/items/0", operationFootprint(Operation{Op: "replace", Path: "/items/0"}, nodes))
	index := &editIndex{}
	index.add("/items")
	require.True(t, index.conflicts("/items/1/b"))
	require.True(t, index.conflicts(""))
	require.False(t, index.conflicts("/items2"))
}

func TestPatchJSONRequiredMembers(t *testing.T) {
	for _, data := range []string{`{"op":"copy","path":"/x"}`, `{"op":"remove"}`, `{"op":"add","path":""}`, `{"op":"test","path":null,"value":1}`} {
		var op Operation
		require.Error(t, json.Unmarshal([]byte(data), &op))
	}
	encoded, err := json.Marshal(Operation{Op: "copy", From: "", Path: "/x"})
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"from":""`)
	var op Operation
	require.NoError(t, json.Unmarshal([]byte(`{"op":"add","path":"","value":null,"extra":true}`), &op))
	require.Equal(t, json.RawMessage(`null`), op.Value)
}
