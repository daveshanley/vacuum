package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"testing"
)

func TestOperationParameters_GetSchema(t *testing.T) {
	def := OperationParameters{}
	assert.Equal(t, "oasOpParams", def.GetSchema().Name)
}

func TestOperationParameters_RunRule(t *testing.T) {
	def := OperationParameters{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperationParameters_RunRule_Success(t *testing.T) {

	yml := `paths:
  /users/{id}:
    get:
      parameters:
        - in: path
          name: id
        - in: query
          name: chicken
        - in: cookie
          name: pizza
        - in: header
          name: minty
        - in: body
          name: limes`

	path := "$.paths"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperationParameters_RunRule_MissingName(t *testing.T) {

	yml := `paths:
  /users/{id}:
    get:
      parameters:
        - in: path
          name: id
        - in: query`

	path := "$.paths"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the `GET` operation parameter at path `/users/{id}`, index 1 has no `name` value", res[0].Message)
}

func TestOperationParameters_RunRule_DuplicateIdButDifferentInType(t *testing.T) {

	yml := `paths:
  /users/{id}:
    get:
      parameters:
        - in: path
          name: id
        - in: query
          name: id`

	path := "$.paths"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperationParameters_RunRule_DuplicateIdSameInType(t *testing.T) {

	yml := `paths:
  /users/{id}:
    get:
      parameters:
        - in: path
          name: id
        - in: path
          name: id`

	path := "$.paths"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the `GET` operation parameter at path `/users/{id}`, index 1 has a duplicate name `id` and `in` type", res[0].Message)
}

func TestOperationParameters_RunRule_DuplicateId_MultipleVerbsDifferentInTypes(t *testing.T) {

	yml := `paths:
  /users/{id}:
    get:
      parameters:
        - in: path
          name: id
        - in: query
          name: id
    post:
      parameters:
        - in: path
          name: id
        - in: query
          name: winter`

	path := "$.paths"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperationParameters_RunRule_DuplicateId_MultipleVerbs(t *testing.T) {

	yml := `paths:
  /users/{id}:
    get:
      parameters:
        - in: path
          name: id
        - in: path
          name: id
    post:
      parameters:
        - in: path
          name: id
        - in: query
          name: winter`

	path := "$.paths"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the `GET` operation parameter at path `/users/{id}`, index 1 has a duplicate name `id` and `in` type", res[0].Message)
}

func TestOperationParameters_RunRule_DuplicateInBody(t *testing.T) {

	yml := `paths:
  /snakes/cakes:
    post:
      parameters:
        - in: body
          name: snakes
        - in: body
          name: cake`

	path := "$.paths"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the `POST` operation at path `/snakes/cakes` contains a duplicate param in:body definition", res[0].Message)
}

func TestOperationParameters_RunRule_FormDataAndBody(t *testing.T) {

	yml := `paths:
  /snakes/cakes:
    post:
      parameters:
        - in: body
          name: snakes
        - in: formData
          name: cake`

	path := "$.paths"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the `POST` operation at path `/snakes/cakes` contains parameters using both in:body and in:formData", res[0].Message)
}
