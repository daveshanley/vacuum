package openapi_functions

import (
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

var allOperations = "$.paths[*]['get','put','post','delete','options','head','patch','trace']"

func TestSuccessResponse_RunRule_Success(t *testing.T) {

	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/burgershop.openapi.yaml")

	nodes, _ := utils.FindNodes([]byte(sampleYaml), allOperations)

	rule := buildOpenApiTestRuleAction(allOperations, "xor", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := SuccessResponse{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestSuccessResponse_RunRule_NoNodes(t *testing.T) {

	rule := buildOpenApiTestRuleAction(allOperations, "xor", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := SuccessResponse{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
