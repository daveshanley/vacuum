package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"os"
	"testing"
)

func TestSuccessResponse_GetSchema(t *testing.T) {
	def := SuccessResponse{}
	assert.Equal(t, "oasOpSuccessResponse", def.GetSchema().Name)
}

func TestSuccessResponse_RunRule(t *testing.T) {
	def := SuccessResponse{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestSuccessResponse_RunRule_Success(t *testing.T) {

	sampleYaml, _ := os.ReadFile("../../model/test_files/burgershop.openapi.yaml")

	info, _ := datamodel.ExtractSpecInfo([]byte(sampleYaml))

	nodes, _ := utils.FindNodes(sampleYaml, "$")

	rule := buildOpenApiTestRuleAction(GetAllOperationsJSONPath(), "xor", "responses", nil)

	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := SuccessResponse{}
	ctx.SpecInfo = info
	res := def.RunRule(nodes, ctx)
	assert.Len(t, res, 0)
}

func TestSuccessResponse_TriggerFailure(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /melody:
    post:
      operationId: fresh
      responses:
        "500":
          description: hello`

	path := "$"

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	rule := buildOpenApiTestRuleAction(path, "success_response", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(info.RootNode, config)
	ctx.SpecInfo = info
	def := SuccessResponse{}
	res := def.RunRule(info.RootNode.Content, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "operation `fresh` must define at least a single `2xx` or `3xx` response", res[0].Message)

}

func TestSuccessResponse_NoPaths(t *testing.T) {

	yml := `swagger: 2.0
definitions:
  something:
    description: hello`

	path := "$"

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	rule := buildOpenApiTestRuleAction(path, "success_response", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(info.RootNode, config)
	ctx.SpecInfo = info
	def := SuccessResponse{}
	res := def.RunRule(info.RootNode.Content, ctx)

	assert.Len(t, res, 0)
}

func TestSuccessResponse_TriggerFailure_NoId(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /melody:
    post:
      responses:
        "500":
          description: hello`

	path := "$"

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	rule := buildOpenApiTestRuleAction(path, "success_response", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(info.RootNode, config)
	ctx.SpecInfo = info

	def := SuccessResponse{}
	res := def.RunRule(info.RootNode.Content, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "operation `undefined operation (no operationId)` must define at least a"+
		" single `2xx` or `3xx` response", res[0].Message)

}

func TestSuccessResponse_TriggerFailure_IntVsString(t *testing.T) {

	yml := `openapi: 3.1
paths:
  /melody:
    post:
      responses:
        500:
          description: hello`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	info, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "success_response", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(info.RootNode, config)
	ctx.SpecInfo = info

	def := SuccessResponse{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "operation `undefined operation (no operationId)` uses an `integer` instead of a "+
		"`string` for response code `500`", res[0].Message)

}

func TestSuccessResponse_RunRule_NoNodes(t *testing.T) {

	rule := buildOpenApiTestRuleAction("$", "xor", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := SuccessResponse{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
