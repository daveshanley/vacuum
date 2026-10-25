package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/require"
	"testing"
)

// Compatibility work must not silently remove existing validation checks.
func TestPreservedValueFindings(t *testing.T) {
	for _, tc := range []struct {
		name, document, function string
		implementation           model.RuleFunction
		options                  map[string]any
		count                    int
	}{
		{"empty array", "value: []", "truthy", &Truthy{}, nil, 1},
		{"empty object", "value: {}", "truthy", &Truthy{}, nil, 1},
		{"nonempty array", "value: [1]", "truthy", &Truthy{}, nil, 0},
		{"null", "value: null", "truthy", &Truthy{}, nil, 1},
		{"numeric pattern", "value: 42", "pattern", &Pattern{}, map[string]any{"match": "^[a-z]+$"}, 1},
		{"matching numeric pattern", "value: 42", "pattern", &Pattern{}, map[string]any{"match": "^[0-9]+$"}, 0},
		{"empty casing", "value: ''", "casing", &Casing{}, map[string]any{"type": "camel"}, 1},
		{"byte maximum", "value: é", "length", &Length{}, map[string]any{"max": 1}, 1},
		{"scalar xor", "value: text", "xor", &Xor{}, map[string]any{"properties": []string{"a", "b"}}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := utils.FindNodes([]byte(tc.document), "$.value")
			require.NoError(t, err)
			rule := buildCoreTestRule("$.value", model.SeverityError, tc.function, "", tc.options)
			ctx := buildCoreTestContextFromRule(model.CastToRuleAction(rule.Then), rule)
			ctx.Given = "$.value"
			require.Len(t, tc.implementation.RunRule(nodes, ctx), tc.count)
		})
	}
}
