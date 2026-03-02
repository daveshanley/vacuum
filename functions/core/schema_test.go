package core

import (
	"context"
	"github.com/daveshanley/vacuum/model"
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"testing"
)

func TestOpenAPISchema_GetSchema(t *testing.T) {
	def := Schema{}
	assert.Equal(t, "schema", def.GetSchema().Name)
}

func TestOpenAPISchema_RunRule(t *testing.T) {
	def := Schema{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOpenAPISchema_DuplicateEntryInEnum(t *testing.T) {

	yml := `components:
  schemas:
    Color:
      type: string
      enum:
        - black
        - white
        - black`

	path := "$..[?(@.enum)]"

	nodes, _ := utils.FindNodes([]byte(yml), path)
	opts := make(map[string]interface{})

	validate := `type: array
items:
  type: string
uniqueItems: true`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts["schema"] = schema

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "enum",
			Function:        "enum",
			FunctionOptions: opts,
		},
		Description: "Enum values must not have duplicate entry",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Enum values must not have duplicate entry: items at 0 and 2 are equal", res[0].Message)

}

func TestOpenAPISchema_InvalidSchemaInteger(t *testing.T) {

	yml := `smell: not a number`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	validate := `type: integer`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts := make(map[string]interface{})
	opts["schema"] = schema

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "smell",
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "schema must be valid",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema must be valid: got string, want integer", res[0].Message)

}

func testGenerateJSONSchema(node *yaml.Node) *highBase.Schema {
	sch := lowBase.Schema{}
	_ = low.BuildModel(node, &sch)
	_ = sch.Build(context.Background(), node, nil)
	highSch := highBase.NewSchema(&sch)
	return highSch
}

func TestOpenAPISchema_InvalidSchemaBoolean(t *testing.T) {

	yml := `smell:
  stank: not a bool`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	props := make(map[string]*highBase.Schema)
	props["stank"] = &highBase.Schema{
		Type: []string{utils.BooleanLabel},
	}

	opts := make(map[string]interface{})

	validate := `type: object
properties:
  stank:
    type: boolean`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts["schema"] = schema

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "smell",
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "schema must be valid",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema must be valid: got string, want boolean", res[0].Message)

}

func TestOpenAPISchema_MissingFieldForceValidation(t *testing.T) {

	yml := `eliminate:
  cyberhacks: not a bool`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	props := make(map[string]*highBase.Schema)
	props["stank"] = &highBase.Schema{
		Type: []string{utils.BooleanLabel},
	}

	opts := make(map[string]interface{})

	validate := `type: object
properties:
  stank:
    type: boolean`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts["schema"] = schema
	opts["forceValidation"] = true

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "lolly",
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "schema must be valid",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema must be valid: `lolly`, is missing and is required", res[0].Message)

}

// Test for Issue #490 - Auto-enable forceValidationOnCurrentNode for root validation
func TestSchema_AutoEnableRootValidation_AdditionalProperties(t *testing.T) {
	// Test case where additional properties should be rejected
	yml := `consumers:
  - name: Consumer 1
    id: consumer-1
services:
  - name: Service 1
    id: service-1`

	path := "$"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	validate := `type: object
properties:
  consumers:
    type: array
    items:
      type: object
      properties:
        name:
          type: string
        id:
          type: string
additionalProperties: false`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts := make(map[string]interface{})
	opts["schema"] = schema

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			// No field specified - should auto-enable root validation
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Ensure only consumer entities allowed",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "additional properties 'services' not allowed")
}

func TestSchema_AutoEnableRootValidation_ValidDocument(t *testing.T) {
	// Test case where document is valid
	yml := `consumers:
  - name: Consumer 1
    id: consumer-1
  - name: Consumer 2
    id: consumer-2`

	path := "$"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	validate := `type: object
properties:
  consumers:
    type: array
    items:
      type: object
      properties:
        name:
          type: string
        id:
          type: string
additionalProperties: false`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts := make(map[string]interface{})
	opts["schema"] = schema

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			// No field specified - should auto-enable root validation
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Ensure only consumer entities allowed",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0) // Should pass validation
}

func TestSchema_AutoEnableRootValidation_WithField(t *testing.T) {
	// Test that auto-enable doesn't happen when a field is specified
	yml := `consumers:
  - name: Consumer 1
    id: consumer-1
extra: "should not matter"`

	path := "$"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	validate := `type: array
items:
  type: object
  properties:
    name:
      type: string
    id:
      type: string`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts := make(map[string]interface{})
	opts["schema"] = schema

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "consumers", // Field is specified
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Validate consumers field",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0) // Should pass - only validates consumers field
}

func TestSchema_AutoEnableRootValidation_NonRootPath(t *testing.T) {
	// Test that auto-enable doesn't happen for non-root paths
	yml := `root:
  nested:
    value: test
    extra: field`

	path := "$.root.nested"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	validate := `type: object
properties:
  value:
    type: string
additionalProperties: false`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts := make(map[string]interface{})
	opts["schema"] = schema
	// Need to explicitly enable forceValidationOnCurrentNode for non-root paths
	opts["forceValidationOnCurrentNode"] = true

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			// No field, but path is not root - auto-enable won't trigger
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Validate nested object",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	// Should fail because it validates the nested object
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "additional properties 'extra' not allowed")
}

func TestSchema_AutoEnableRootValidation_NonRootPath_NoAutoEnable(t *testing.T) {
	// Test that auto-enable truly doesn't happen for non-root paths without explicit force
	yml := `root:
  nested:
    someField:
      value: test`

	path := "$.root"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	validate := `type: object
properties:
  value:
    type: string
required: [value]`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts := make(map[string]interface{})
	opts["schema"] = schema

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "nested", // Looking for a field
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Validate nested field",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	// Should fail - nested.someField doesn't have required 'value' property
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "missing property 'value'")
}

func TestSchema_ExplicitForceValidationOnCurrentNode(t *testing.T) {
	// Test that explicit forceValidationOnCurrentNode still works
	yml := `consumers:
  - name: Consumer 1
    id: consumer-1
services:
  - name: Service 1`

	path := "$"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	validate := `type: object
properties:
  consumers:
    type: array
additionalProperties: false`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts := make(map[string]interface{})
	opts["schema"] = schema
	opts["forceValidationOnCurrentNode"] = true // Explicitly set

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "someField", // Even with field, explicit force should work
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Force validation on current node",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "additional properties 'services' not allowed")
}

// Test for Issue #799 - schema function with contains and nested object properties
// triggers "circular reference detected during inline rendering" error
func TestSchema_Issue799_ContainsWithNestedProperties(t *testing.T) {
	// Simulating an OpenAPI operation with parameters array
	yml := `parameters:
  - name: X-Other-Header
    in: header
    required: true
  - name: X-Required-Version
    in: header
    required: true`

	path := "$"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	// Schema using contains with nested object properties - this triggers the bug
	validate := `type: array
minItems: 1
contains:
  type: object
  properties:
    name:
      const: X-Required-Version
    in:
      const: header
  required:
    - name
    - in`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts := make(map[string]interface{})
	opts["schema"] = schema

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "parameters",
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Each operation must include X-Required-Version header param",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	// This should pass because the parameters array contains the required header
	// Bug: Currently fails with "circular reference detected during inline rendering"
	assert.Len(t, res, 0, "Should pass - parameters contains X-Required-Version header")
}

// Test for Issue #799 - schema function with items and nested object properties
func TestSchema_Issue799_ItemsWithNestedProperties(t *testing.T) {
	// Simulating an OpenAPI operation with parameters array
	yml := `parameters:
  - name: X-Other-Header
    in: header
    required: true`

	path := "$"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	// Schema using items with nested object properties - also triggers the bug
	validate := `type: array
items:
  type: object
  properties:
    name:
      type: string
  required:
    - name`

	var n yaml.Node
	_ = yaml.Unmarshal([]byte(validate), &n)

	schema := testGenerateJSONSchema(n.Content[0])

	opts := make(map[string]interface{})
	opts["schema"] = schema

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "parameters",
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Each operation parameter must have a name",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	// This should pass because all parameters have a name
	// Bug: Currently fails with "circular reference detected during inline rendering"
	assert.Len(t, res, 0, "Should pass - all parameters have required name field")
}

// Test for Issue #799 - schema passed as raw map (simulates ruleset YAML parsing)
// This is the actual code path used during linting - schema is NOT pre-built
func TestSchema_Issue799_SchemaFromRawMap(t *testing.T) {
	// Simulating an OpenAPI operation with parameters array
	yml := `parameters:
  - name: X-Other-Header
    in: header
    required: true
  - name: X-Required-Version
    in: header
    required: true`

	path := "$"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	// Schema passed as raw map (like from YAML ruleset parsing)
	// This is how it comes from the ruleset, NOT as a pre-built *highBase.Schema
	schemaMap := map[string]interface{}{
		"type":     "array",
		"minItems": 1,
		"contains": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"const": "X-Required-Version",
				},
				"in": map[string]interface{}{
					"const": "header",
				},
			},
			"required": []interface{}{"name", "in"},
		},
	}

	opts := make(map[string]interface{})
	opts["schema"] = schemaMap // Pass as raw map, NOT as *highBase.Schema

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "parameters",
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Each operation must include X-Required-Version header param",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
		Index:      nil, // No index - still fails!
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	// This should pass because the parameters array contains the required header
	// Bug: This is the actual failing path - schema is built from raw map
	assert.Len(t, res, 0, "Should pass - parameters contains X-Required-Version header")
}

// Test for Issue #799 - simpler schema from raw map that DOES work
func TestSchema_Issue799_SimpleSchemaFromRawMap(t *testing.T) {
	yml := `parameters:
  - name: X-Other-Header
    in: header`

	path := "$"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	// Simple schema without nested properties - should work
	schemaMap := map[string]interface{}{
		"type":     "array",
		"minItems": 1,
	}

	opts := make(map[string]interface{})
	opts["schema"] = schemaMap

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "parameters",
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Parameters must exist",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	// Simple schema works fine
	assert.Len(t, res, 0, "Should pass - simple schema works")
}

// Test for Issue #799 - items with properties from raw map
func TestSchema_Issue799_ItemsPropertiesFromRawMap(t *testing.T) {
	yml := `parameters:
  - name: X-Other-Header
    in: header`

	path := "$"
	nodes, _ := utils.FindNodes([]byte(yml), path)

	// Schema with items.properties - also triggers the bug
	schemaMap := map[string]interface{}{
		"type": "array",
		"items": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []interface{}{"name"},
		},
	}

	opts := make(map[string]interface{})
	opts["schema"] = schemaMap

	rule := model.Rule{
		Given: path,
		Then: &model.RuleAction{
			Field:           "parameters",
			Function:        "schema",
			FunctionOptions: opts,
		},
		Description: "Each parameter must have a name",
	}

	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := Schema{}
	res := def.RunRule(nodes, ctx)

	// Bug: This also fails with circular reference
	assert.Len(t, res, 0, "Should pass - parameter has name")
}
