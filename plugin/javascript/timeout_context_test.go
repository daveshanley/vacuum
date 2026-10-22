package javascript

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
	"strings"
	"testing"
	"time"
)

func TestJSBatchUsesInvocationTimeout(t *testing.T) {
	fn := NewJSRuleFunction("batch-timeout", strings.ReplaceAll(slowJSScript, `message: "finished"`, `message: "finished", input: input[0]`)).(*JSRuleFunction)
	require.NoError(t, fn.CheckScript())
	fn.SetTimeout(time.Millisecond)
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "input"}
	for _, tc := range []struct {
		timeout time.Duration
		message string
	}{{20 * time.Millisecond, "timed out after 20ms"}, {2 * time.Second, "finished"}} {
		results := fn.RunRule([]*yaml.Node{node}, model.RuleFunctionContext{RuleTimeout: tc.timeout, Options: map[string]any{"batch": true}})
		require.Len(t, results, 1)
		require.Contains(t, results[0].Message, tc.message)
	}
	// A direct caller that supplies no context timeout keeps the legacy setting.
	require.Equal(t, time.Millisecond, fn.getTimeout(model.RuleFunctionContext{}))
	fn.SetTimeout(0)
	require.Equal(t, DefaultRuleTimeout, fn.getTimeout(model.RuleFunctionContext{}))
}
