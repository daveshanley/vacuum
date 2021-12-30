package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTypedEnum_GetSchema(t *testing.T) {
	def := TypedEnum{}
	assert.Equal(t, "typed_enum", def.GetSchema().Name)
}

func TestTypedEnum_RunRule(t *testing.T) {
	def := TypedEnum{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestPathParameters_RunRule_SuccessCheck(t *testing.T) {

	yml := `paths:
  /pizza/:
    parameters:
      - in: query
        name: party
        schema:
          type: string
          enum: [big, small]
  /cake/:
    parameters:
      - in: query
        name: icecream
        schema:
          type: string
          enum: [lots, little]        
components:
  schemas:
    YesNo:
      type: string
      enum: [yes, no]`

	path := "$..[?(@.enum && @.type)]"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "typed_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := TypedEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestPathParameters_RunRule_ThreeValue_WrongType(t *testing.T) {

	yml := `paths:
  /pizza/:
    parameters:
      - in: query
        name: party
        schema:
          type: string
          enum: [big, 1]
  /cake/:
    parameters:
      - in: query
        name: icecream
        schema:
          type: string
          enum: [0.2, little]        
components:
  schemas:
    YesNo:
      type: string
      enum: [yes, true]`

	path := "$..[?(@.enum && @.type)]"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "typed_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := TypedEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 3)
}
