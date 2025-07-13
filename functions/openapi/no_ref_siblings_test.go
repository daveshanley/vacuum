package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"net/url"
	"testing"
)

func TestNoRefSiblings_GetSchema(t *testing.T) {
	def := NoRefSiblings{}
	assert.Equal(t, "refSiblings", def.GetSchema().Name)
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
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

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
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

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
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

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
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

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
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

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
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestNoRefSiblings_RunRule_MultiFile(t *testing.T) {
	rootYML := `
openapi: 3.0.0
info: {title: api, version: 1.0.0}
paths:
  /item:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: './child.yaml#/components/schemas/External'
components:
  schemas:
    RootBad:
      description: wrong
      $ref: '#/components/schemas/Ok'
    Ok: {type: object}`

	childYML := `
components:
  schemas:
    ExternalBad:
      description: wrong
      $ref: '#/components/schemas/Ok2'
    External: {type: object}
    Ok2: {type: object}`

	var rootNode, childNode yaml.Node
	_ = yaml.Unmarshal([]byte(rootYML), &rootNode)
	_ = yaml.Unmarshal([]byte(childYML), &childNode)

	childCfg := index.CreateOpenAPIIndexConfig()
	childCfg.BaseURL, _ = url.Parse("child.yaml")
	childCfg.AllowFileLookup = false
	childCfg.AllowRemoteLookup = false
	childIdx := index.NewSpecIndexWithConfig(&childNode, childCfg)

	rolodex := index.NewRolodex(childCfg)
	rolodex.AddIndex(childIdx)
	
	rootCfg := index.CreateOpenAPIIndexConfig()
	rootCfg.Rolodex = rolodex
	rootCfg.AllowFileLookup = false
	rootCfg.AllowRemoteLookup = false
	rootIdx := index.NewSpecIndexWithConfig(&rootNode, rootCfg)

	rule := buildOpenApiTestRuleAction("$", "no_ref_siblings_multi_file", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = rootIdx

	nodes, _ := utils.FindNodes([]byte(rootYML), "$")

	var def NoRefSiblings
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}
