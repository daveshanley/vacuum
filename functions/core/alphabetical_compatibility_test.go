package core

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/require"
)

func TestAlphabeticalLexicalAndNumericOrder(t *testing.T) {
	for _, tc := range []struct{ name, input, keyedBy, path string }{
		{"lowercase after uppercase", "[a, B]", "", "$.values[0]"},
		{"ordered mixed case", "[B, a]", "", ""},
		{"accent after ASCII", "[é, f]", "", "$.values[0]"},
		{"ordered accent", "[f, é]", "", ""},
		{"integer order", "[2, 10]", "", ""},
		{"unordered integers", "[10, 9]", "", "$.values[0]"},
		{"mixed numeric order", "[2, 10.5]", "", ""},
		{"mixed numeric violation", "[2.5, 2]", "", "$.values[0]"},
		{"exact large integers", "[9007199254740993, 9007199254740992]", "", "$.values[0]"},
		{"third keyed element", "[{name: a}, {name: b}, {name: A}]", "name", "$.values"},
		{"missing key after inversion", "[{name: z}, {name: a}, {}]", "name", "$.values"},
		{"missing key before inversion", "[{}, {name: z}, {name: a}]", "name", "$.values"},
		{"missing key preserves comparison", "[{name: z}, {}, {name: a}]", "name", "$.values"},
		{"nonobject after inversion", "[{name: z}, {name: a}, false]", "name", "$.values"},
		{"unsupported key after inversion", "[{name: z}, {name: a}, {name: false}]", "name", "$.values"},
		{"missing key across null", "[{name: z}, {}, {name: a}]", "name", "$.values"},
		{"unsupported entry after inversion", "[z, a, false]", "", "$.values[0]"},
		{"unsupported entry before inversion", "[false, z, a]", "", "$.values[1]"},
		{"unsupported entry preserves comparison", "[z, null, a]", "", "$.values[0]"},
		{"ordered keyed elements", "[{name: B}, {name: a}, {name: c}]", "name", ""},
		{"ordered map keys", "{B: 1, a: 2}", "", ""},
		{"unordered map keys", "{a: 1, B: 2}", "", "$.values"},
		{"keyed object values", "{first: {rank: 2}, second: {rank: 10}}", "rank", "$.values"},
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
	require.NoError(t, yaml.Unmarshal([]byte("first: [a, B]\nsecond: [10, 9]\n"), &root))
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

func TestAlphabeticalPreservesAllArrayFindings(t *testing.T) {
	for _, tc := range []struct{ name, input, keyedBy string }{
		{"strings", "[z, y, x]", ""},
		{"numbers", "[3, 2, 1]", ""},
		{"floats", "[3.5, 2.5, 1.5]", ""},
		{"keyed array", "[{name: z}, {name: y}, {name: x}]", "name"},
		{"keyed map", "{first: {name: z}, second: {name: y}, third: {name: x}}", "name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := utils.FindNodes([]byte("values: "+tc.input), "$.values")
			require.NoError(t, err)
			rule := buildCoreTestRule("$.values", model.SeverityError, "alphabetical", "", map[string]any{"keyedBy": tc.keyedBy})
			ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
			ctx.Given = "$.values"
			actual := (Alphabetical{}).RunRule(nodes, ctx)
			require.Len(t, actual, 2)
			if tc.keyedBy == "" {
				require.Equal(t, "$.values[0]", actual[0].Path)
				require.Equal(t, "$.values[1]", actual[1].Path)
			}
		})
	}
}
