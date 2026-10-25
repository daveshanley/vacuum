package core

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/require"
)

func TestAlphabeticalLocaleAndNumericOrder(t *testing.T) {
	for _, tc := range []struct{ name, input, keyedBy, path string }{
		{"lowercase before uppercase", "[a, B]", "", ""},
		{"unordered mixed case", "[B, a]", "", "$.values[0]"},
		{"accent before next letter", "[é, f]", "", ""},
		{"unordered accent", "[f, é]", "", "$.values[0]"},
		{"integer order", "[2, 10]", "", ""},
		{"unordered integers", "[10, 9]", "", "$.values[0]"},
		{"mixed numeric order", "[2, 10.5]", "", ""},
		{"mixed numeric violation", "[2.5, 2]", "", "$.values[0]"},
		{"exact large integers", "[9007199254740993, 9007199254740992]", "", "$.values[0]"},
		{"first violation only", "[z, y, x]", "", "$.values[0]"},
		{"third keyed element", "[{name: a}, {name: b}, {name: A}]", "name", "$.values"},
		{"ordered keyed elements", "[{name: a}, {name: B}, {name: c}]", "name", ""},
		{"ordered map keys", "{a: 1, B: 2}", "", ""},
		{"unordered map keys", "{B: 1, a: 2}", "", "$.values"},
		{"keyed object values", "{first: {rank: 10}, second: {rank: 2}}", "rank", "$.values"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			nodes, err := utils.FindNodes([]byte("values: "+tc.input), "$.values")
			require.NoError(t, err)
			rule := buildCoreTestRule("$.values", model.SeverityError, "alphabetical", "", map[string]any{"keyedBy": tc.keyedBy})
			ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
			ctx.Given = "$.values"
			actual := (Alphabetical{}).RunRule(nodes, ctx)
			if tc.path == "" {
				require.Empty(t, actual)
				return
			}
			require.Len(t, actual, 1)
			require.Equal(t, tc.path, actual[0].Path)
		})
	}
}

func TestAlphabeticalKeepsEachSelectedContainerPath(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("first: [B, a]\nsecond: [10, 9]\n"), &root))
	nodes, err := utils.FindNodesWithoutDeserializing(&root, "$.*")
	require.NoError(t, err)
	rule := buildCoreTestRule("$.*", model.SeverityError, "alphabetical", "", nil)
	ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
	ctx.Given = "$.*"
	ctx.Index = index.NewSpecIndex(&root)
	actual := (Alphabetical{}).RunRule(nodes, ctx)
	require.Len(t, actual, 2)
	require.Equal(t, "$.first[0]", actual[0].Path)
	require.Equal(t, "$.second[0]", actual[1].Path)
	require.Same(t, root.Content[0].Content[1].Content[0], actual[0].StartNode)
	require.Same(t, root.Content[0].Content[3].Content[0], actual[1].StartNode)
}
