package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestPathParameters_GetSchema(t *testing.T) {
	def := PathParameters{}
	assert.Equal(t, "oasPathParam", def.GetSchema().Name)
}

func TestPathParameters_RunRule(t *testing.T) {
	def := PathParameters{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestPathParameters_RunRule_AllParamsInTop(t *testing.T) {

	yml := `paths:
  /pizza/{type}/{topping}:
    parameters:
      - name: type
        in: path
    get:
      operationId: get_pizza`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "`GET` must define parameter `topping` as expected by path `/pizza/{type}/{topping}`", res[0].Message)
}

func TestPathParameters_RunRule_VerbsWithDifferentParams(t *testing.T) {

	yml := `paths:
  /pizza/{type}/{topping}:
    parameters:
      - name: type
        in: path
    get:
      parameters:
        - name: topping
          in: path
      operationId: get_pizza
    post:
      operationId: make_pizza
`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "`POST` must define parameter `topping` as expected by path `/pizza/{type}/{topping}`", res[0].Message)
}

func TestPathParameters_RunRule_DuplicatePathCheck(t *testing.T) {

	yml := `paths:
  /pizza/{cake}/{limes}:
    parameters:
      - in: path
        name: cake
    get:
      parameters:
        - in: path
          name: limes
  /pizza/{minty}/{tape}:
    parameters:
      - in: path
        name: minty
    get:
      parameters:
        - in: path
          name: tape`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestPathParameters_RunRule_DuplicatePathParamCheck_MissingParam(t *testing.T) {

	yml := `openapi: 3.0.1
info:
title: pizza-cake
paths:
  /pizza/{cake}/{cake}:
    parameters:
      - in: path
        name: cake
    get:
      parameters:
        - in: path
          name: limes
  /pizza/{minty}:
    parameters:
      - in: path
        name: minty
    get:
      parameters:          `

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}

func TestPathParameters_RunRule_MissingParam_PeriodInParam(t *testing.T) {

	yml := `openapi: 3.0.1
info:
title: pizza-cake
paths:
  /pizza/{cake}/{cake.id}:
    parameters:
      - in: path
        name: cake
    get:
      parameters:
        - in: path
          name: cake.id
          required: true`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}

func TestPathParameters_RunRule_TopParameterCheck_MissingRequired(t *testing.T) {

	yml := `paths:
 /musical/{melody}/{pizza}:
   parameters:
       - in: path
         name: melody
         required: fresh
   get:
     parameters:
       - in: path
         name: pizza
         required: true`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "/musical/{melody}/{pizza} must have `required` parameter that is set to `true`", res[0].Message)
}

func TestPathParameters_RunRule_TopParameterCheck_RequiredShouldBeTrue(t *testing.T) {

	yml := `paths:
 /musical/{melody}/{pizza}:
   parameters:
       - in: path
         name: melody
         required: false
   get:
     parameters:
       - in: path
         name: pizza
         required: true`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "/musical/{melody}/{pizza} must have `required` parameter that is set to `true`", res[0].Message)
}

func TestPathParameters_RunRule_TopParameterCheck_MultipleDefinitionsOfParam(t *testing.T) {

	yml := `paths:
 /musical/{melody}/{pizza}:
   parameters:
       - in: path
         name: melody
       - in: path
         name: melody
   get:
     parameters:
       - in: path
         name: pizza
       - in: path
         name: pizza`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}

func TestPathParameters_RunRule_TopParameterCheck(t *testing.T) {

	yml := `paths:
 /musical/{melody}/{pizza}:
   parameters:
       - in: path
         name: melody
   get:
     parameters:
       - in: path
         name: pizza`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestPathParameters_RunRule_TopParameterCheck_MissingParamDefInOp(t *testing.T) {

	yml := `paths:
 /musical/{melody}/{pizza}/{cake}:
   parameters:
       - in: path
         name: melody
         required: true
   get:
     parameters:
       - in: path
         name: pizza
         required: true`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "`GET` must define parameter `cake` as expected by path `/musical/{melody}/{pizza}/{cake}`",
		res[0].Message)
}

func TestPathParameters_RunRule_MultiplePaths_TopAndVerbParams(t *testing.T) {

	yml := `components: 
  parameters:
    chicken:
      in: path
      required: true
      name: chicken
paths:
 /musical/{melody}/{pizza}/{cake}:
   parameters:
     - in: path
       name: melody
       required: true
   get:
     parameters:
       - in: path
         name: pizza
         required: true
 /dogs/{chicken}/{ember}:
   get:
     parameters:
       - in: path
         name: ember
         required: true
       - $ref: '#/components/parameters/chicken'
   post:
     parameters:
       - in: path
         name: ember
       - $ref: '#/components/parameters/chicken'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}

	// we need to resolve this
	r := index.NewResolver(ctx.Index)
	r.Resolve()
	res := def.RunRule([]*yaml.Node{&rootNode}, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "`GET` must define parameter `cake` as expected by path `/musical/{melody}/{pizza}/{cake}`",
		res[0].Message)
}

func TestPathParameters_RunRule_NoParamsDefined(t *testing.T) {

	yml := `
paths:
  /update/{somethings}:
    post:
      operationId: postSomething
      summary: Post something
      tags:
        - tag1
      responses:
        '200':
          description: Post OK
    get:
      operationId: getSomething
      summary: Get something
      tags:
        - tag1
      responses:
        '200':
          description: Get OK
components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: basic`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "`POST` must define parameter `somethings` as expected by path `/update/{somethings}`",
		res[0].Message)
	assert.Equal(t, "`GET` must define parameter `somethings` as expected by path `/update/{somethings}`",
		res[1].Message)
}

func TestPathParameters_RunRule_NoParamsDefined_TopExists(t *testing.T) {

	yml := `paths:
  /update/{somethings}:
    parameters:
      - in: path
        name: somethings
        schema:
          type: string
          example: something nice
          description: this is something nice.
        required: true
    post:
      operationId: postSomething
      summary: Post something
      tags:
        - tag1
      responses:
        '200':
          description: Post OK
    get:
      operationId: getSomething
      summary: Get something
      tags:
        - tag1
      responses:
        '200':
          description: Get OK
components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: basic`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}
