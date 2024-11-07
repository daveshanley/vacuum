package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestOASNoRefSiblings_GetSchema(t *testing.T) {
	def := OASNoRefSiblings{}
	assert.Equal(t, "oasRefSiblings", def.GetSchema().Name)
}

func TestOASNoRefSiblings_RunRule(t *testing.T) {
	def := OASNoRefSiblings{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOASNoRefSiblings_RunRule_Description(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    requestBody:
      content:
        application/json:
          schema:
            description: this is a good place to do this
            $ref: '#/components/schemas/Rice'
  /hot/{dog}:
    requestBody:
      content:
        application/json:
          schema:
            description: Still a good place to do this
            $ref: '#/components/schemas/Dog'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)
	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "oas3_no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOASNoRefSiblings_RunRule_Deprecated_Example(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    requestBody:
      content:
        application/json:
          schema:
            description: the deprecated flag should not be here
            deprecated: true
            $ref: '#/components/schemas/Rice'
  /hot/{dog}:
    requestBody:
      content:
        application/json:
          schema:
            example: this should also not be here
            $ref: '#/components/schemas/Dog'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)
	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "oas3_no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}

func TestOASNoRefSiblings_RunRule_Components(t *testing.T) {

	yml := `components:
  schemas:
    Beer:
      description: perfect
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

	rule := buildOpenApiTestRuleAction(path, "oas3_no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}

func TestOASNoRefSiblings_RunRule_Parameters(t *testing.T) {

	yml := `parameters:
  testParam:
    $ref: '#/parameters/oldParam'
  oldParam:
    in: query
    description: old
  wrongParam:
    description: I am allowed to be here
    $ref: '#/parameters/oldParam'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "oas3_no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}

func TestOASNoRefSiblings_RunRule_Definitions(t *testing.T) {

	yml := `definitions:
  test:
    $ref: '#/definitions/old'
  old:
    type: object
    description: old
  wrong:
    description: I am allowed to be here
    $ref: '#/definitions/oldParam'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "oas3_no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}

func TestOASNoRefSiblings_RunRule_Success(t *testing.T) {

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

	rule := buildOpenApiTestRuleAction(path, "oas3_no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}

func TestOASNoRefSiblings_RunRule_Fail_Single(t *testing.T) {

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
            type: integer
            $ref: '#/components/schemas/Dog'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "oas3_no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}
