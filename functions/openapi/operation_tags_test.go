package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestOperationTags_GetSchema(t *testing.T) {
	def := OperationTags{}
	assert.Equal(t, "operation_tags", def.GetSchema().Name)
}

func TestOperationTags_RunRule(t *testing.T) {
	def := OperationTags{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperationTags_RunRule_Success(t *testing.T) {

	yml := `paths:
  /hello:
    post:
      tags:
       - a
       - b
    get:
      tags:
       - a
  /there/yeah:
    post:
      tags:
       - b`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation_tags", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationTags{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
	//assert.Equal(t, "the 'get' operation at path '/ember' contains a duplicate operationId 'littleSong'", res[0].Message)
}

func TestOperationTags_RunRule_NoTags(t *testing.T) {

	yml := `paths:
  /hello:
    post:
      description: hi
    get:
      tags:
       - a
  /there/yeah:
    post:
      tags:
       - b`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation_tags", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationTags{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Tags for `post` operation at path `/hello` are missing", res[0].Message)
	assert.Equal(t, "$.paths./hello.post", res[0].Path)

}

func TestOperationTags_RunRule_EmptyTags(t *testing.T) {

	yml := `paths:
  /hello:
    post:
      tags:
       - a
       - b
    get:
      tags:
  /there/yeah:
    post:
      tags:
       - b`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation_tags", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationTags{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Tags for `get` operation at path `/hello` are empty", res[0].Message)
	assert.Equal(t, "$.paths./hello.get", res[0].Path)

}

func TestOperationTags_RunRule_IgnoreParameters(t *testing.T) {

	yml := `paths:
  /hello:
    post:
      tags:
        - a
        - b
    parameters:
      - in: query`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation_tags", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationTags{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperationTags_RunRule_IgnoreServers(t *testing.T) {
	yml := `paths:
  /hello:
    post:
      tags:
        - a
        - b
    servers:
      - url: https://api.example.com/v1`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation_tags", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationTags{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperationTags_RunRule_IgnoreExtensions(t *testing.T) {

	yml := `paths:
  /hello:
    post:
      tags:
        - a
        - b
    x-parameters:
      - in: query
    parameters:
      - in: path`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation_tags", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationTags{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperationTags_RunRule_HandleNoPaths(t *testing.T) {

	yml := `openapi: 3.0.3`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation_tags", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationTags{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}
