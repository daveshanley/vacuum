package core

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/require"
)

func TestUnreferencedReusableObject(t *testing.T) {
	for _, tc := range []struct {
		name, document string
		want           []string
	}{
		{"reference", `use: {$ref: '#/defs/Used'}
defs: {Used: {}, Unused: {}}`, []string{"$.defs.Unused"}},
		{"none", `defs: {A: {}, B: {}}`, []string{"$.defs.A", "$.defs.B"}},
		{"polymorphic", `use: {allOf: [{$ref: '#/defs/A'}]}
defs: {A: {}}`, nil},
		{"escaped keys", `use: {$ref: '#/defs/a~1b~0c'}
defs: {'a/b~c': {}, Missing: {}}`, []string{"$.defs.Missing"}},
		{"percent encoding", `use: {$ref: '#/defs/a%20b'}
defs: {'a b': {}, Missing: {}}`, []string{"$.defs.Missing"}},
		{"self reference", `defs: {A: {$ref: '#/defs/A'}}`, nil},
		{"not object", `defs: [a, b]`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.document), &root))
			idx := index.NewSpecIndexWithConfig(&root, index.CreateClosedAPIIndexConfig())
			nodes, err := utils.FindNodes([]byte(tc.document), "$.defs")
			require.NoError(t, err)
			ctx := model.RuleFunctionContext{Index: idx, Rule: &model.Rule{Id: "unused"}, Given: "$.defs", Options: map[string]any{"reusableObjectsLocation": "#/defs"}}
			results := (UnreferencedReusableObject{}).RunRule(nodes, ctx)
			var paths []string
			for _, result := range results {
				paths = append(paths, result.Path)
				require.Equal(t, "unused", result.Rule.Id)
				require.Greater(t, result.StartNode.Line, 0)
			}
			require.Equal(t, tc.want, paths)
		})
	}
}

func TestUnreferencedReusableObjectOptionsAndMissingIndex(t *testing.T) {
	function := UnreferencedReusableObject{}
	for _, option := range []any{nil, map[string]any{}, map[string]any{"reusableObjectsLocation": 3}, map[string]any{"reusableObjectsLocation": "remote.yaml#/defs"}, map[string]any{"reusableObjectsLocation": "#/bad~2"}, map[string]any{"reusableObjectsLocation": "#/%zz"}} {
		require.Error(t, function.ValidateOptions(option))
	}
	require.NoError(t, function.ValidateOptions(map[string]any{"reusableObjectsLocation": "#/a~1b"}))
	require.NoError(t, function.ValidateOptions(map[string]any{"reusableObjectsLocation": "#%2Fdefs"}))
	var failure error
	results := function.RunRule([]*yaml.Node{{Kind: yaml.MappingNode}}, model.RuleFunctionContext{Rule: &model.Rule{Id: "unused"}, Options: map[string]any{"reusableObjectsLocation": "#/defs"}, ReportError: func(err error) { failure = err }})
	require.ErrorContains(t, failure, "requires a document index")
	require.Len(t, results, 1)
}

func TestUnreferencedReusableObjectRoot(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("Used: {}\nUnused: {}\nuse: {$ref: '#/Used'}"), &root))
	idx := index.NewSpecIndexWithConfig(&root, index.CreateClosedAPIIndexConfig())
	results := (UnreferencedReusableObject{}).RunRule(root.Content, model.RuleFunctionContext{Index: idx, Rule: &model.Rule{Id: "unused"}, Given: "$", Options: map[string]any{"reusableObjectsLocation": "#"}})
	require.Len(t, results, 2)
	require.Equal(t, "$.Unused", results[0].Path)
	require.Equal(t, "$.use", results[1].Path)
}
