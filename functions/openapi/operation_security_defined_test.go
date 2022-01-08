package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperationSecurityDefined_GetSchema(t *testing.T) {
	def := OperationSecurityDefined{}
	assert.Equal(t, "operation_security_defined", def.GetSchema().Name)
}

func TestOperationSecurityDefined_RunRule(t *testing.T) {
	def := OperationSecurityDefined{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperationSecurityDefined_Success(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    post: 
      security:
        - BasicAuth: [admin]
  /hot/{dog}:
    get:
      security:
        - BasicAuth: [admin]
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	ops := make(map[string]string)
	ops["schemesPath"] = "$.components.securitySchemes"

	rule := buildOpenApiTestRuleAction(path, "operation_security", "", ops)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), ops)

	def := OperationSecurityDefined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestOperationSecurityDefined_Fail_One(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    post: 
      security:
        - BingoDingo: [admin]
  /hot/{dog}:
    get:
      security:
        - BasicAuth: [admin]
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	ops := make(map[string]string)
	ops["schemesPath"] = "$.components.securitySchemes"

	rule := buildOpenApiTestRuleAction(path, "operation_security", "", ops)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), ops)

	def := OperationSecurityDefined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}

func TestOperationSecurityDefined_Fail_Two(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    post: 
      security:
        - BingoDingo: [admin]
  /hot/{dog}:
    get:
      security:
        - JingoJango: [admin]
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	ops := make(map[string]string)
	ops["schemesPath"] = "$.components.securitySchemes"

	rule := buildOpenApiTestRuleAction(path, "operation_security", "", ops)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), ops)

	def := OperationSecurityDefined{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}
