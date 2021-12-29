package openapi_functions

import (
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperationParameters_GetSchema(t *testing.T) {
	def := OperationParameters{}
	assert.Equal(t, "operation_parameters", def.GetSchema().Name)
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

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

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

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the 'get' operation at path '/users/{id}' contains a parameter with no 'name' value", res[0].Message)
}

func TestOperationParameters_RunRule_DuplicateId(t *testing.T) {

	yml := `paths:
  /users/{id}:
    get:
      parameters:
        - in: path
          name: id
        - in: query
          name: id`

	path := "$.paths"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the 'get' operation at path '/users/{id}' contains a parameter with duplicate name 'id'", res[0].Message)
}

func TestOperationParameters_RunRule_DuplicateId_MultipleVerbs(t *testing.T) {

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

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the 'get' operation at path '/users/{id}' contains a parameter with duplicate name 'id'", res[0].Message)
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

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the 'post' operation at path '/snakes/cakes' contains a duplicate param in:body definition", res[0].Message)
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

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "operation-parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := OperationParameters{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the 'post' operation at path '/snakes/cakes' contains parameters using both in:body and in:formData", res[0].Message)
}
