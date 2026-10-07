package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestRuleMessageTemplates(t *testing.T) {
	for _, tc := range []struct{ name, spec, given, field, function, template, want string }{
		{"issue 952", `tools: [{name: test, description: ''}]`, "$.tools[*]", "description", "truthy", "Tool '{{property}}' is empty", "Tool 'description' is empty"},
		{"all variables", `tools: [{description: ''}]`, "$.tools[*]", "description", "truthy", "{{description}}|{{property}}|{{value}}|{{path}}|{{error}}", "required field|description||/tools/0/description|required field: `description` must be set"},
		{"concrete selection", `tools: [{description: ''}]`, "$.tools[0]", "description", "truthy", "{{property}}|{{path}}", "description|/tools/0/description"},
		{"missing field", `tools: [{}]`, "$.tools[*]", "description", "truthy", "{{property}}|{{value}}|{{path}}", "description||/tools/0/description"},
		{"nested field", `tools: [{meta: {description: ''}}]`, "$.tools[*]", "meta.description", "truthy", "{{property}}|{{path}}", "description|/tools/0/meta/description"},
		{"numeric", `values: [0]`, "$.values[*]", "", "truthy", "{{property}}={{value}} at {{path}}", "0=0 at /values/0"},
		{"no recursive replacement", `value: '{{property}}'`, "$.value", "", "falsy", "{{value}} {{unknown}}", "{{property}} {{unknown}}"},
		{"escaped pointer", `'a/b~c': false`, "$['a/b~c']", "", "truthy", "{{property}}|{{value}}|{{path}}", "a/b~c|false|/a~1b~0c"},
		{"plain message", `value: true`, "$.value", "", "falsy", "custom message", "custom message"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := &model.Rule{Id: "template", Description: "required field", Message: tc.template, Given: tc.given, Severity: "warn", Then: model.RuleAction{Field: tc.field, Function: tc.function}}
			result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(tc.spec), SkipDocumentCheck: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"template": rule}}})
			defer result.Release()
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 1)
			require.Equal(t, tc.want, result.Results[0].Message)
			require.Equal(t, tc.template, rule.Message)
			require.Same(t, rule, result.Results[0].Rule)
		})
	}
}

type messageTestFunction struct{}

func (messageTestFunction) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "messageTest"}
}
func (messageTestFunction) GetCategory() string { return model.FunctionCategoryCustomJS }
func (messageTestFunction) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		results = append(results, model.RuleFunctionResult{Message: "original error", StartNode: node, EndNode: node})
	}
	return results
}

func TestRuleMessageTemplatesCustomAndBatch(t *testing.T) {
	for _, batch := range []bool{false, true} {
		rule := &model.Rule{Id: "template", Message: "{{property}}={{value}}: {{error}}", Given: "$.values[*]", Severity: "warn", Then: model.RuleAction{Function: "messageTest", FunctionOptions: map[string]any{"batch": batch}}}
		result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte("values: [first, second]\n"), SkipDocumentCheck: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"template": rule}}, CustomFunctions: map[string]model.RuleFunction{"messageTest": messageTestFunction{}}})
		require.Empty(t, result.Errors)
		var messages []string
		for _, r := range result.Results {
			messages = append(messages, r.Message)
		}
		require.ElementsMatch(t, []string{"0=first: original error", "1=second: original error"}, messages)
		result.Release()
	}
}
