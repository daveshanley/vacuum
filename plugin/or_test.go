package plugin

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin/javascript"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestJavaScriptOr(t *testing.T) {
	function := javascript.NewJSRuleFunction("testOr", `function runRule(input) { return vacuum_or(input, context); }`)
	require.NoError(t, function.CheckScript())
	RegisterCoreFunctions(function)
	for _, tc := range []struct {
		input string
		want  int
	}{{"title: null", 0}, {"{}", 1}} {
		var doc yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(tc.input), &doc))
		results := function.RunRule(doc.Content, model.RuleFunctionContext{
			Rule:    &model.Rule{},
			Options: map[string]any{"properties": []any{"title", "description"}},
		})
		require.Len(t, results, tc.want)
		if tc.want != 0 {
			assert.Equal(t, `At least one of "title" or "description" must be defined`, results[0].Message)
		}
	}
}
