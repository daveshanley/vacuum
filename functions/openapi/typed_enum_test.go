package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

func TestTypedEnum_RunRule_SuccessCheck(t *testing.T) {
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

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "typed_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := TypedEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestTypedEnum_RunRule_31NullableEnum_SuccessCheck(t *testing.T) {
	yml := `paths:
  /pizza/:
    parameters:
      - in: query
        name: party
        schema:
          type: [string, null]
          enum: [big, small, null]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "typed_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := TypedEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestTypedEnums_RunRule_ThreeValue_WrongType(t *testing.T) {
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
      enum: [yes, true]
    TooManyTypes:
      type: [string, integer, null]
      enum: [hi, 1, null]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "typed_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := TypedEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 5)
}
