package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestPathsKebabCase_GetSchema(t *testing.T) {
	def := PathsKebabCase{}
	assert.Equal(t, "pathsKebabCase", def.GetSchema().Name)
}

func TestPathsKebabCase_RunRule(t *testing.T) {
	def := PathsKebabCase{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestPathsKebabCase_Success(t *testing.T) {

	yml := `openapi: 3.0.0
paths:
  '/woah/slow-down/you-move-too-fast':
    get:
      summary: not bad
  '/youHave/got/to/make/the_morning last':
    get:
      summary: bad path
  '/just-kicking/down/the/cobble-stones':
    get:
      summary: nice
  '/looking~1/{forFun}/AND/feeling_groovy':
    get:
      summary: this is also doomed
  '/ok//ok':
    get:
      summary: should we complain? nah`

	path := "$"

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yml), &rootNode)

	assert.NoError(t, err)
	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "pathsKebabCase", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathsKebabCase{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}

func TestPathsKebabCase_WithExtension(t *testing.T) {

	yml := `openapi: 3.0.0
paths:
  '/woah/slow-down/you-move-too-fast.pdf':
    get:
      summary: not bad
  '/you-have/got/to/make/{theMorning}.last':
    get:
      summary: still good
  '/just-kicking/down/the/cobble-stones.csv':
    get:
      summary: nice
  '/looking/{for-fun}/and/feeling-groovy.json':
    get:
      summary: this is fine
  '/ok//ok':
    get:
      summary: should we complain? nah`

	path := "$"

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yml), &rootNode)

	assert.NoError(t, err)
	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "pathsKebabCase", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathsKebabCase{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}
