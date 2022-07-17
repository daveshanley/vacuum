package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"testing"
)

func TestOperation4xResponse_GetSchema(t *testing.T) {
	def := Operation4xResponse{}
	assert.Equal(t, "operation_4xx_response", def.GetSchema().Name)
}

func TestOperation4xResponse_RunRule(t *testing.T) {
	def := Operation4xResponse{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperation4xResponse_RunRule_Success(t *testing.T) {

	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/burgershop.openapi.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(sampleYaml, &rootNode)
	nodes, _ := utils.FindNodes(sampleYaml, "$")
	rule := buildOpenApiTestRuleAction(GetAllOperationsJSONPath(), "xor", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = model.NewSpecIndex(&rootNode)

	def := Operation4xResponse{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperation4xResponse_RunRule_ExitEarly(t *testing.T) {

	sampleYaml := []byte("hi: there")

	var rootNode yaml.Node
	yaml.Unmarshal(sampleYaml, &rootNode)
	nodes, _ := utils.FindNodes(sampleYaml, "$")
	rule := buildOpenApiTestRuleAction(GetAllOperationsJSONPath(), "xor", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = model.NewSpecIndex(&rootNode)

	def := Operation4xResponse{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperation4xResponse_RunRule_Fail(t *testing.T) {

	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/stripe.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(sampleYaml, &rootNode)
	nodes, _ := utils.FindNodes(sampleYaml, "$")
	rule := buildOpenApiTestRuleAction(GetAllOperationsJSONPath(), "xor", "responses", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = model.NewSpecIndex(&rootNode)

	def := Operation4xResponse{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 402)
}
