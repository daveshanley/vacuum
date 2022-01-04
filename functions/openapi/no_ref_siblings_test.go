package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNoRefSiblings_GetSchema(t *testing.T) {
	def := NoRefSiblings{}
	assert.Equal(t, "no_ref_siblings", def.GetSchema().Name)
}

func TestNoRefSiblings_RunRule(t *testing.T) {
	def := NoRefSiblings{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestNoRefSiblings_RunRule_Fail(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    requestBody:
      content:
        application/json:
          schema:
            description: this is the wrong place this this buddy.
            $ref: '#/components/schemas/Rice'
  /hot/{dog}:
    requestBody:
      content:
        application/json:
          schema:
            description: still the wrong place for this.
            $ref: '#/components/schemas/Dog'`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}

func TestNoRefSiblings_RunRule_Success(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Rice'
  /hot/{dog}:
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Dog'`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}

func TestNoRefSiblings_RunRule_Fail_Single(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Rice'
  /hot/{dog}:
    requestBody:
      content:
        application/json:
          schema:
            description: still the wrong place for this.
            $ref: '#/components/schemas/Dog'`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}
