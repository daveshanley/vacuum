package motor

import (
	"fmt"
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
		{"apostrophe", `"a'b": false`, `$["a'b"]`, "", "truthy", "{{property}}|{{value}}|{{path}}", "a'b|false|/a'b"},
		{"empty key", `"": false`, `$[""]`, "", "truthy", "{{property}}|{{value}}|{{path}}", "|false|/"},
		{"plain message", `value: true`, "$.value", "", "falsy", "custom message", "custom message: `value` must be falsy"},
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

type messageTestFunction struct{ child, children, keys bool }

func (messageTestFunction) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "messageTest"}
}
func (messageTestFunction) GetCategory() string { return model.FunctionCategoryCustomJS }
func (f messageTestFunction) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		if f.children {
			for i := 0; i+1 < len(node.Content); i += 2 {
				target := node.Content[i+1]
				if f.keys {
					target = node.Content[i]
				}
				results = append(results, model.RuleFunctionResult{Message: "original error", StartNode: target, EndNode: target})
			}
			continue
		}
		if f.child {
			node = node.Content[1]
		}
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

func TestRuleMessageTemplatesPreserveCustomResultTarget(t *testing.T) {
	rule := &model.Rule{Id: "template", Message: "{{property}}|{{value}}|{{path}}|{{error}}", Given: "$.object", Severity: "warn", Then: model.RuleAction{Function: "messageTest"}}
	result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte("object: {child: bad}\n"), SkipDocumentCheck: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"template": rule}}, CustomFunctions: map[string]model.RuleFunction{"messageTest": messageTestFunction{child: true}}})
	defer result.Release()
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.Equal(t, "child|bad|/object/child|original error", result.Results[0].Message)
}

func TestRuleMessageTemplatesUseEachAction(t *testing.T) {
	rule := &model.Rule{Id: "template", Message: "{{property}} at {{path}}", Given: "$.object", Severity: "warn", Then: []model.RuleAction{{Function: "truthy", Field: "first"}, {Function: "truthy", Field: "second"}}}
	result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte("object: {first: '', second: ''}\n"), SkipDocumentCheck: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"template": rule}}})
	defer result.Release()
	require.Empty(t, result.Errors)
	var messages []string
	for _, r := range result.Results {
		messages = append(messages, r.Message)
	}
	require.ElementsMatch(t, []string{"first at /object/first", "second at /object/second"}, messages)
}

func TestRuleMessageTemplatesMatchDoctorResults(t *testing.T) {
	for _, function := range []string{"truthy", "falsy"} {
		for _, message := range []string{"custom error", "custom {{property}}: {{error}}"} {
			t.Run(function+"/"+message, func(t *testing.T) {
				field := "description"
				if function == "falsy" {
					field = "title"
				}
				rule := &model.Rule{Id: "template", Given: "$.info", Message: message, Description: "title rule", Severity: "error", Resolved: true, Then: model.RuleAction{Function: function, Field: field}}
				result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"template": rule}}})
				defer result.Release()
				require.Empty(t, result.Errors)
				require.Len(t, result.Results, 1)
				doc := result.RuleSetExecution.DrDocument
				require.NotNil(t, doc)
				stored := doc.V3Document.GetRuleFunctionResults()
				require.Len(t, stored, 1)
				require.Equal(t, result.Results[0].Message, stored[0].Message)
				require.Equal(t, message, stored[0].Rule.Message)
			})
		}
	}
}

func TestRuleMessageTemplatesConcurrentRules(t *testing.T) {
	rules := make(map[string]*model.Rule)
	for i := 0; i < 32; i++ {
		id := fmt.Sprintf("rule-%d", i)
		rules[id] = &model.Rule{Id: id, Given: "$.values[*]", Message: "{{property}}={{value}} at {{path}}", Severity: "warn", Then: model.RuleAction{Function: "truthy"}}
	}
	result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte("values: [0, false]\n"), SkipDocumentCheck: true, RuleSet: &rulesets.RuleSet{Rules: rules}})
	defer result.Release()
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 64)
	for _, r := range result.Results {
		require.Contains(t, []string{"0=0 at /values/0", "1=false at /values/1"}, r.Message)
	}
}

func TestRuleMessageTemplatesConcurrentCustomTargets(t *testing.T) {
	for _, keys := range []bool{false, true} {
		rules := make(map[string]*model.Rule)
		for i := 0; i < 32; i++ {
			id := fmt.Sprintf("rule-%d", i)
			rules[id] = &model.Rule{Id: id, Given: "$.object", Message: "{{property}}={{value}} at {{path}}", Severity: "warn", Then: model.RuleAction{Function: "messageTest"}}
		}
		result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte("object: {first: one, second: two}\n"), SkipDocumentCheck: true, RuleSet: &rulesets.RuleSet{Rules: rules}, CustomFunctions: map[string]model.RuleFunction{"messageTest": messageTestFunction{children: true, keys: keys}}})
		require.Empty(t, result.Errors)
		require.Len(t, result.Results, 64)
		for _, r := range result.Results {
			require.Contains(t, []string{"first=one at /object/first", "second=two at /object/second"}, r.Message)
		}
		result.Release()
	}
}
