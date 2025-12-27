// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package javascript

import (
	"fmt"
	"github.com/daveshanley/vacuum/functions/core"
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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
				fmt.Printf("✗ Core function '%s' had a panic attack via JavaScript: %s\n", r, "truthy")
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

// ============================================================================
// Async/Await Tests - Testing the new event loop functionality
// ============================================================================

func Test_JSPlugin_Async_Promise_Resolve(t *testing.T) {
	script := `
function runRule(input) {
	return Promise.resolve([{ message: "async result: " + input }]);
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("hello async"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 1)
	assert.Equal(t, "async result: hello async", results[0].Message)
}

func Test_JSPlugin_Async_AsyncAwait_Basic(t *testing.T) {
	script := `
async function runRule(input) {
	const value = await Promise.resolve("processed");
	return [{ message: value + ": " + input }];
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("test input"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 1)
	assert.Equal(t, "processed: test input", results[0].Message)
}

func Test_JSPlugin_Async_MultipleAwaits(t *testing.T) {
	script := `
async function runRule(input) {
	const a = await Promise.resolve("first");
	const b = await Promise.resolve("second");
	const c = await Promise.resolve("third");
	return [{ message: a + "-" + b + "-" + c }];
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("input"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 1)
	assert.Equal(t, "first-second-third", results[0].Message)
}

func Test_JSPlugin_Async_WithSetTimeout(t *testing.T) {
	script := `
function runRule(input) {
	return new Promise(function(resolve) {
		setTimeout(function() {
			resolve([{ message: "delayed result" }]);
		}, 10);
	});
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("input"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 1)
	assert.Equal(t, "delayed result", results[0].Message)
}

func Test_JSPlugin_Async_EmptyResult(t *testing.T) {
	script := `
async function runRule(input) {
	await Promise.resolve();
	// Validation passed, return empty array
	return [];
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("valid input"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Empty(t, results)
}

func Test_JSPlugin_Async_ConditionalValidation(t *testing.T) {
	script := `
async function runRule(input) {
	const isValid = await Promise.resolve(input.valid);
	if (!isValid) {
		return [{ message: "validation failed for " + input.name }];
	}
	return [];
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	// Test with invalid input
	var y yaml.Node
	_ = yaml.Unmarshal([]byte("name: test\nvalid: false"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 1)
	assert.Equal(t, "validation failed for test", results[0].Message)

	// Test with valid input
	_ = yaml.Unmarshal([]byte("name: test2\nvalid: true"), &y)

	results = f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Empty(t, results)
}

func Test_JSPlugin_Async_PromiseAll(t *testing.T) {
	script := `
async function runRule(input) {
	const results = await Promise.all([
		Promise.resolve("a"),
		Promise.resolve("b"),
		Promise.resolve("c")
	]);
	return [{ message: results.join("-") }];
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("input"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 1)
	assert.Equal(t, "a-b-c", results[0].Message)
}

func Test_JSPlugin_Async_PromiseReject(t *testing.T) {
	script := `
async function runRule(input) {
	throw new Error("async validation error");
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("input"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 1)
	// Error objects from JS are exported as maps, so check for promise rejection
	assert.Contains(t, results[0].Message, "promise rejected")
}

func Test_JSPlugin_Async_PromiseRejectString(t *testing.T) {
	// Test with string rejection which preserves the message
	script := `
async function runRule(input) {
	return Promise.reject("string validation error");
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("input"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "string validation error")
}

func Test_JSPlugin_Async_NestedAsyncFunctions(t *testing.T) {
	script := `
async function helper(value) {
	return await Promise.resolve(value.toUpperCase());
}

async function runRule(input) {
	const processed = await helper(input);
	return [{ message: "Result: " + processed }];
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("hello world"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 1)
	assert.Equal(t, "Result: HELLO WORLD", results[0].Message)
}

func Test_JSPlugin_Async_MixedSyncAsync(t *testing.T) {
	// Test that synchronous code still works with the event loop
	script := `
function runRule(input) {
	// Purely synchronous - no async/await
	if (input.length < 5) {
		return [{ message: "Input too short" }];
	}
	return [];
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("hi"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 1)
	assert.Equal(t, "Input too short", results[0].Message)
}

func Test_JSPlugin_Async_WithContext(t *testing.T) {
	script := `
async function runRule(input) {
	await Promise.resolve();
	const fieldName = context.ruleAction.field;
	if (!input[fieldName]) {
		return [{ message: "Field '" + fieldName + "' is required" }];
	}
	return [];
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("name: test"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
		RuleAction: &model.RuleAction{
			Field: "description",
		},
	})
	assert.Len(t, results, 1)
	assert.Equal(t, "Field 'description' is required", results[0].Message)
}

func Test_JSPlugin_Async_MultipleResults(t *testing.T) {
	script := `
async function runRule(input) {
	await Promise.resolve();
	var errors = [];
	if (!input.name) {
		errors.push({ message: "name is required" });
	}
	if (!input.description) {
		errors.push({ message: "description is required" });
	}
	return errors;
}
`
	f := NewJSRuleFunction("test", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("version: 1.0"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "test.path",
	})
	assert.Len(t, results, 2)
	assert.Equal(t, "name is required", results[0].Message)
	assert.Equal(t, "description is required", results[1].Message)
}

// Integration test for the async_validation.js sample - tests a real-world async pattern
func Test_JSPlugin_Async_SampleValidation(t *testing.T) {
	// This is a simplified version of the async_validation.js sample
	script := `
function getSchema() {
    return {
        "name": "asyncValidation",
        "description": "Demonstrates async/await in vacuum custom functions"
    };
}

async function simulateAsyncCheck(value) {
    return new Promise(function(resolve) {
        setTimeout(function() {
            var isDeprecated = value && value.toLowerCase().indexOf("deprecated") !== -1;
            resolve({
                valid: !isDeprecated,
                reason: isDeprecated ? "Contains deprecated indicator" : null
            });
        }, 5);
    });
}

async function runRule(input) {
    var results = [];

    if (input.description) {
        var checkResult = await simulateAsyncCheck(input.description);
        if (!checkResult.valid) {
            results.push({
                message: "Description issue: " + checkResult.reason
            });
        }
    }

    return results;
}
`
	f := NewJSRuleFunction("asyncValidation", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	// Test with deprecated content - should return an error
	var y yaml.Node
	_ = yaml.Unmarshal([]byte("description: This is deprecated content"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "$.info",
	})
	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "deprecated")

	// Test with valid content - should return no errors
	_ = yaml.Unmarshal([]byte("description: This is a valid API description"), &y)

	results = f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "$.info",
	})
	assert.Empty(t, results)
}

// Test parallel async operations with Promise.all
func Test_JSPlugin_Async_ParallelOperations(t *testing.T) {
	script := `
async function checkItem(item) {
    return new Promise(function(resolve) {
        setTimeout(function() {
            resolve({
                name: item,
                valid: item.length > 2
            });
        }, 5);
    });
}

async function runRule(input) {
    if (!input.items || !Array.isArray(input.items)) {
        return [];
    }

    var checks = input.items.map(function(item) {
        return checkItem(item);
    });

    var results = await Promise.all(checks);

    return results
        .filter(function(r) { return !r.valid; })
        .map(function(r) {
            return { message: "Item '" + r.name + "' is too short" };
        });
}
`
	f := NewJSRuleFunction("parallelCheck", script)
	err := f.CheckScript()
	assert.NoError(t, err)

	var y yaml.Node
	_ = yaml.Unmarshal([]byte("items:\n  - ab\n  - abc\n  - x\n  - valid"), &y)

	results := f.RunRule([]*yaml.Node{y.Content[0]}, model.RuleFunctionContext{
		Given: "$.data",
	})
	// "ab" and "x" are too short (length <= 2)
	assert.Len(t, results, 2)
	assert.Contains(t, results[0].Message, "ab")
	assert.Contains(t, results[1].Message, "x")
}
