package openapi_functions

import (
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestSuccessResponse_RunRule_Success(t *testing.T) {

	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/burgershop.openapi.yaml")

	nodes, _ := utils.FindNodes([]byte(sampleYaml), GetAllOperationsJSONPath())

	rule := buildOpenApiTestRuleAction(GetAllOperationsJSONPath(), "xor", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := SuccessResponse{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestSuccessResponse_RunRule_NoNodes(t *testing.T) {

	rule := buildOpenApiTestRuleAction(GetAllOperationsJSONPath(), "xor", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := SuccessResponse{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
