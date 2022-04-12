package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestOAS2OperationSecurityDefined_GetSchema(t *testing.T) {
	def := OAS2OperationSecurityDefined{}
	assert.Equal(t, "oas2_operation_security_defined", def.GetSchema().Name)
}

func TestOAS2OperationSecurityDefined_RunRule(t *testing.T) {
	def := OAS2OperationSecurityDefined{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOAS2OperationSecurityDefined_RunRule_Success(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /melody:
    post:
      security:
        - littleChampion: []
    get:
      security:
        - littleSong: []		
securityDefinitions:
  littleChampion:
    type: basic
  littleSong:
    type: apiKey
    in: header
    name: X-API-KEY`

	path := "$"

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(yml), &rootNode)

	rule := buildOpenApiTestRuleAction(path, "oas2_operation_security_defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = model.NewSpecIndex(&rootNode)

	def := OAS2OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)
}

func TestOAS2OperationSecurityDefined_RunRule_Fail(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /melody:
    post:
      security:
        - littleMenace: []
    get:
      security:
        - littleSong: []		
securityDefinitions:
  littleChampion:
    type: basic
  littleSong:
    type: apiKey
    in: header
    name: X-API-KEY`

	path := "$"

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(yml), &rootNode)

	rule := buildOpenApiTestRuleAction(path, "oas2_operation_security_defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = model.NewSpecIndex(&rootNode)

	def := OAS2OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 1)
}
