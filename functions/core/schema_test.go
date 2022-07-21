package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOpenAPISchema_GetSchema(t *testing.T) {
	def := Schema{}
	assert.Equal(t, "oas_schema", def.GetSchema().Name)
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
	opts["schema"] = parser.Schema{
		Type: &utils.ArrayLabel,
		Items: &parser.Schema{
			Type: &utils.StringLabel,
		},
		UniqueItems: true,
	}

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
	assert.Equal(t, "Enum values must not have duplicate entry: array items[0,2] must be unique", res[0].Message)

}

func TestOpenAPISchema_InvalidSchemaInteger(t *testing.T) {

	yml := `smell:
  stink: not a number`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	props := make(map[string]*parser.Schema)
	props["stink"] = &parser.Schema{Type: &utils.IntegerLabel}

	opts := make(map[string]interface{})
	opts["schema"] = parser.Schema{
		Type:       &utils.ObjectLabel,
		Properties: props,
	}

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
	assert.Equal(t, "schema must be valid: Invalid type. Expected: integer, given: string", res[0].Message)

}

func TestOpenAPISchema_InvalidSchemaBoolean(t *testing.T) {

	yml := `smell:
  stank: not a bool`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	props := make(map[string]*parser.Schema)
	props["stank"] = &parser.Schema{Type: &utils.BooleanLabel}

	opts := make(map[string]interface{})
	opts["schema"] = parser.Schema{
		Type:       &utils.ObjectLabel,
		Properties: props,
	}

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
	assert.Equal(t, "schema must be valid: Invalid type. Expected: boolean, given: string", res[0].Message)

}

func TestOpenAPISchema_MissingFieldForceValidation(t *testing.T) {

	yml := `eliminate:
  cyberhacks: not a bool`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	props := make(map[string]*parser.Schema)
	props["stank"] = &parser.Schema{Type: &utils.BooleanLabel}

	opts := make(map[string]interface{})
	opts["schema"] = parser.Schema{
		Type:       &utils.ObjectLabel,
		Properties: props,
	}
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
