package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestSuccessResponse_GetSchema(t *testing.T) {
	def := SuccessResponse{}
	assert.Equal(t, "success_response", def.GetSchema().Name)
}

func TestSuccessResponse_RunRule(t *testing.T) {
	def := SuccessResponse{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestSuccessResponse_RunRule_Success(t *testing.T) {

	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/burgershop.openapi.yaml")

	nodes, _ := utils.FindNodes(sampleYaml, "$")

	rule := buildOpenApiTestRuleAction(GetAllOperationsJSONPath(), "xor", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := SuccessResponse{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestSuccessResponse_RunRule_NoNodes(t *testing.T) {

	rule := buildOpenApiTestRuleAction("$", "xor", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := SuccessResponse{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
