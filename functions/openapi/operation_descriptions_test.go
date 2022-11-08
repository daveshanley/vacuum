package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)

}

func TestOperationDescription_SummaryButNoDescription(t *testing.T) {

	yml := `paths:
  /fish/paste:
    post:
      summary: this is a thing that does nothing`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "1"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "2"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "Field `requestBody` for operation `post` at path `/fish/paste` is missing a description and a summary", res[0].Message)

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
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "5"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "Field `requestBody` for operation `post` description at path `/fish/paste` must be at least 5 words "+
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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "2"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "Operation `post` response `200` at path `/fish/paste` is missing a description and a summary",
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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "2"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "Operation `post` response `200` description at path `/fish/paste` must be at least 2 "+
		"words long, (1 is not enough)", res[0].Message)

}

func TestOperationDescription_CheckParametersIgnored(t *testing.T) {

	yml := `paths:
  /fish/paste:
    parameters:
      - in: query
    post:
      description: this is a description that is great and 10 words long at least
      operationId: c`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperationDescription_CheckServersIgnored(t *testing.T) {
	yml := `paths:
  /fish/paste:
    servers:
      - url: https://api.example.com/v1
    post:
      description: this is a description that is great and 10 words long at least
      operationId: c`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperationDescription_CheckExtensionsIgnored(t *testing.T) {

	yml := `paths:
  /fish/paste:
    x-parameters:
      - in: query
    post:
      description: this is a description that is great and 10 words long at least
      operationId: c`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}

func TestOperationDescription_CheckForNoPaths(t *testing.T) {

	yml := `openapi: 3.0.3`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}
