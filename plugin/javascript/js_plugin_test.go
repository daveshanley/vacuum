// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package javascript

import (
	"fmt"
	"github.com/daveshanley/vacuum/functions/core"
	"github.com/daveshanley/vacuum/model"
	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func Test_JSPlugin_Basic_Fail(t *testing.T) {

	script := `
function runRule(input) {
  if (input !== "hello") {
    return [
      {
        message: 'Value must equal "hello" and not: ' + input + ' and context is: ' + context.options['keyedBy'],
      },
    ];
  }
};`

	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("hello sally"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "pizza",
		Options: map[string]string{
			"keyedBy": "name",
		},
	})
	assert.Equal(t, "Value must equal \"hello\" and not: hello sally and context is: name", results[0].Message)
}

func Test_JSPlugin_Schema_Success(t *testing.T) {

	script := `function getSchema() {
    return {
        "name": "a nice test",
        "properties": [
            {
                "name": "mickey",
                "description": "a mouse"
            }
        ],
    };
}`

	f := NewJSRuleFunction("test", script)
	schema := f.GetSchema()
	assert.NotNil(t, schema)
	assert.Equal(t, "a nice test", schema.Name)
	assert.Equal(t, "mickey", schema.Properties[0].Name)
	assert.Equal(t, "a mouse", schema.Properties[0].Description)
}

func Test_JSPlugin_Bad_Rule_CodeError(t *testing.T) {

	script := `function runRule() {
   throw new Error("oops");
}`

	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("beep"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{})
	assert.Equal(t, "Unable to execute JavaScript function: 'test': Error: oops", results[0].Message)
}

func Test_JSPlugin_Bad_Rule_GoException(t *testing.T) {

	script := `function runRule() {
	vacuum_truthy("oops");
}
`

	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	f.RegisterCoreFunction("truthy", func(input any, context model.RuleFunctionContext) []model.RuleFunctionResult {
		defer func() {
			if r := recover(); r != nil {
				pterm.Error.Printf("Core function '%s' had a panic attack via JavaScript: %s\n", r, "truthy")
			}
		}()
		panic("our go function failed! " + fmt.Sprint(input))
	})

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("beep"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{})
	assert.Empty(t, results)
}

func Test_JSPlugin_Core_Function_OK(t *testing.T) {

	script := `function runRule(input) {
	return vacuum_truthy(input, context);
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	f.RegisterCoreFunction("truthy", func(input interface{}, context model.RuleFunctionContext) []model.RuleFunctionResult {
		truthy := core.Truthy{}
		// re-encode input back into yaml
		var y yaml.Node
		_ = y.Encode(input)

		var results []model.RuleFunctionResult
		if y.Kind == yaml.DocumentNode {
			results = truthy.RunRule([]*yaml.Node{y.Content[0]}, context)
		} else {
			results = truthy.RunRule([]*yaml.Node{&y}, context)
		}

		return results
	})

	checkModel := `apples: true`

	var y yaml.Node
	_ = yaml.Unmarshal([]byte(checkModel), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Rule: &model.Rule{
			Description: "apples must be truthy",
		},
		RuleAction: &model.RuleAction{
			Field: "apples",
		},
	})
	assert.Empty(t, results)
}

func Test_JSPlugin_Core_Function_Not_OK(t *testing.T) {

	script := `function runRule(input) {
	return vacuum_truthy(input, context);
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	f.RegisterCoreFunction("truthy", func(input interface{}, context model.RuleFunctionContext) []model.RuleFunctionResult {
		truthy := core.Truthy{}
		// re-encode input back into yaml
		var y yaml.Node
		_ = yaml.Unmarshal([]byte(fmt.Sprintf("%v", input)), &y)

		results := truthy.RunRule([]*yaml.Node{y.Content[0]}, context)
		return results
	})

	checkModel := `oranges: true`

	var y yaml.Node
	_ = yaml.Unmarshal([]byte(checkModel), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Rule: &model.Rule{
			Description: "apples must be truthy!",
		},
		RuleAction: &model.RuleAction{
			Field: "apples",
		},
	})
	assert.NotEmpty(t, results)
	assert.Equal(t, "apples must be truthy!: `apples` must be set", results[0].Message)
}

func Test_JSPlugin_Bad_Rule_CodeException(t *testing.T) {

	script := `function runRule() {
   throw new Error("oops");
}`

	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("beep"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{})
	assert.Equal(t, "Unable to execute JavaScript function: 'test': Error: oops", results[0].Message)
}

func Test_JSPlugin_Schema_BadData(t *testing.T) {

	script := `function getSchema() {
    return "cakes";
}`

	f := NewJSRuleFunction("test", script)
	schema := f.GetSchema()
	assert.NotNil(t, schema)
	assert.Equal(t, "test", schema.Name)
}

func Test_JSPlugin_Fail_NoRunRuleFunc(t *testing.T) {

	script := `const notAFunc = "hello";`

	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.Error(t, err)
	assert.Equal(t, "runRule function not found", err.Error())

}
