package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperationDescription_GetSchema(t *testing.T) {
	def := OperationDescription{}
	assert.Equal(t, "operation_description", def.GetSchema().Name)
}

func TestOperationDescription_RunRule(t *testing.T) {
	def := OperationDescription{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperationDescription_CheckDescriptionMissing(t *testing.T) {

	yml := `paths:
  /fish/paste:
    get:
      operationId: a
    put:
      operationId: b
    post:
      description: this is a description that is great and 10 words long at least
      operationId: c`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}

func TestOperationDescription_CheckDescriptionTooShort(t *testing.T) {

	yml := `paths:
  /fish/paste:
    post:
      description: this is a thing that does nothing`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)

}

func TestOperationDescription_CheckRequestBodyDescriptionExists(t *testing.T) {

	yml := `paths:
  /fish/paste:
    post:
      description: this is a thing
      requestBody:
        content:
          application/json:
            schema: 
              type: string`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "2"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "Operation requestBody 'post' at path '/fish/paste' is missing a description", res[0].Message)

}

func TestOperationDescription_CheckRequestBodyDescriptionMeetsLength(t *testing.T) {

	yml := `paths:
  /fish/paste:
    post:
      description: this is a thing yeah
      requestBody:
        description: This is another thing
        content:
          application/json:
            schema: 
              type: string`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "5"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "Operation 'post' requestBody description at path '/fish/paste' must be at least 5 words "+
		"long, (4 is not enough)", res[0].Message)

}

func TestOperationDescription_CheckResponsesDescriptionExist(t *testing.T) {

	yml := `paths:
  /fish/paste:
    post:
      description: this is a thing
      requestBody:
        description: this is a thing
      responses:
        '200':
          content:
            application/json:
              schema:
                type: string`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "2"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "Operation 'post' response '200' at path '/fish/paste' is missing a description",
		res[0].Message)

}

func TestOperationDescription_CheckResponsesDescriptionLongEnough(t *testing.T) {

	yml := `paths:
  /fish/paste:
    post:
      description: this is a thing
      requestBody:
        description: this is a thing
      responses:
        '200':
          description: cake
          content:
            application/json:
              schema:
                type: string`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "2"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "Operation 'post' response '200' description at path '/fish/paste' must be at least 2 "+
		"words long, (1 is not enough)", res[0].Message)

}
