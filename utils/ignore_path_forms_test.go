package utils

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestIgnoreMatcher_PathForms(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`components:
  schemas:
    Metadata:
      propertyNames: {maxLength: 32}
    Meta-data:
      propertyNames: {maxLength: 32}
"a.b": [value]
a:
  b: [other]
"a'b": value
`), &root))
	for _, tc := range []struct {
		name, ignore, path string
		want               bool
	}{
		{"wildcard bracket", "$.components.schemas[*].propertyNames", "$.components.schemas['Metadata'].propertyNames", true},
		{"wildcard dot", "$.components.schemas[*].propertyNames", "$.components.schemas.Metadata.propertyNames", true},
		{"hyphen key", "$.components.schemas[*].propertyNames", "$.components.schemas['Meta-data'].propertyNames", true},
		{"recursive", "$..propertyNames", "$['components']['schemas']['Metadata']['propertyNames']", true},
		{"literal equivalent", "$.components.schemas.Metadata.propertyNames", "$.components.schemas['Metadata'].propertyNames", true},
		{"quoted dot and array", "$['a.b'][*]", "$[\"a.b\"][0]", true},
		{"nested key is distinct", "$['a.b'][*]", "$.a.b[0]", false},
		{"different index", "$['a.b'][*]", "$['a.b'][1]", false},
		{"escaped quote", "$[\"a'b\"]", "$['a\\'b']", true},
		{"rule path not subtree", "$.components.schemas[*]", "$.components.schemas.Metadata.propertyNames", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			matcher := NewIgnoreMatcher(model.IgnoredItems{"rule": {tc.ignore}}, IgnoreMatcherOptions{RootNode: &root})
			require.Equal(t, tc.want, matcher.Matches(&model.RuleFunctionResult{RuleId: "rule", Path: tc.path}))
			require.Equal(t, tc.want, matcher.Matches(&model.RuleFunctionResult{RuleId: "rule", Path: "$.unrelated", Paths: []string{tc.path}}))
			require.False(t, matcher.Matches(&model.RuleFunctionResult{RuleId: "other", Path: tc.path}))
		})
	}
}
