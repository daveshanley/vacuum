package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOpenAPISchema_GetSchema(t *testing.T) {
	def := OpenAPISchema{}
	assert.Equal(t, "oas_schema", def.GetSchema().Name)
}

func TestOpenAPISchema_RunRule(t *testing.T) {
	def := OpenAPISchema{}
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

	rule := buildOpenApiTestRuleAction(path, "oas_schema", "enum", opts)
	rule.Description = "Enum values must not have duplicate entry"
	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Rule:       &rule,
		Options:    opts,
		Given:      rule.Given,
	}

	def := OpenAPISchema{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Enum values must not have duplicate entry: array items[0,2] must be unique", res[0].Message)

}

// TODO: add boolean and integer check on values
