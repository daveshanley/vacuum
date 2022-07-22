package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)
	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}

func TestNoRefSiblings_RunRule_Components(t *testing.T) {

	yml := `components:
  schemas:
    Beer:
      description: nice
      $ref: '#/components/Yum'
    Bottle:
      type: string
    Cake:
      $ref: '#/components/Sugar'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestNoRefSiblings_RunRule_Parameters(t *testing.T) {

	yml := `parameters:
  testParam:
    $ref: '#/parameters/oldParam'
  oldParam:
    in: query
    description: old
  wrongParam:
    description: I should not be here
    $ref: '#/parameters/oldParam'  `

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestNoRefSiblings_RunRule_Definitions(t *testing.T) {

	yml := `definitions:
  test:
    $ref: '#/definitions/old'
  old:
    type: object
    description: old
  wrong:
    description: I should not be here
    $ref: '#/definitions/oldParam'  `

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}
