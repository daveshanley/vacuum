package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestDuplicatedEnum_GetSchema(t *testing.T) {
	def := DuplicatedEnum{}
	assert.Equal(t, "duplicated_enum", def.GetSchema().Name)
}

func TestDuplicatedEnum_RunRule(t *testing.T) {
	def := DuplicatedEnum{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestDuplicatedEnum_RunRule_SuccessCheck(t *testing.T) {

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

	rule := buildOpenApiTestRuleAction(path, "duplicated_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := DuplicatedEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestDuplicatedEnum_RunRule_DuplicationFail(t *testing.T) {

	yml := `paths:
  /pizza/:
    parameters:
      - in: query
        name: party
        schema:
          type: string
          enum: [big, small, big, huge, small]
  /cake/:
    parameters:
      - in: query
        name: icecream
        schema:
          type: string
          enum: [little, little]        
components:
  schemas:
    YesNo:
      type: string
      enum: [yes, no, yes, no]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "duplicated_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := DuplicatedEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 5)
}
