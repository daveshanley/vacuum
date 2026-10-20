package rulesets

import (
	"fmt"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestRulesetSchemaCachePreservesValidationAndIsolation(t *testing.T) {
	for i := 0; i < 24; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			valid := []byte(`{"rules":{"check":{"given":"$","then":{"function":"truthy"}}}}`)
			first, err := CreateRuleSetUsingJSON(valid)
			require.NoError(t, err)
			first.Rules["check"].Description = "changed by caller"
			second, err := CreateRuleSetUsingJSON(valid)
			require.NoError(t, err)
			require.Empty(t, second.Rules["check"].Description)
			_, err = CreateRuleSetUsingJSON([]byte(`{"rules":{"check":{"given":42,"then":{"function":"truthy"}}}}`))
			require.Error(t, err)
		})
	}
}

func TestRulesetSchemaCacheUsesCurrentSource(t *testing.T) {
	schema, err := compiledRulesetSchema(`{"type":"string"}`)
	require.NoError(t, err)
	require.NoError(t, schema.Validate("value"))
	require.Error(t, schema.Validate(1))
	schema, err = compiledRulesetSchema(`{"type":"number"}`)
	require.NoError(t, err)
	require.NoError(t, schema.Validate(1))
	require.Error(t, schema.Validate("value"))
	_, err = compiledRulesetSchema(`{`)
	require.Error(t, err)
	schema, err = compiledRulesetSchema(RulesetSchema)
	require.NoError(t, err)
	require.NotNil(t, schema)
}
