package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	ops := make(map[string]string)
	ops["schemesPath"] = "$.components.securitySchemes"

	rule := buildOpenApiTestRuleAction(path, "operation_security", "", ops)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	ops := make(map[string]string)
	ops["schemesPath"] = "$.components.securitySchemes"

	rule := buildOpenApiTestRuleAction(path, "operation_security", "", ops)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 1)
}

func TestOperationSecurityDefined_Fail_One_Root(t *testing.T) {

	yml := `openapi: 3.0
security:
  - BingoDingo: [admin]
paths:
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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "operation_security", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 1)
}

func TestOperationSecurityDefined_Fail_Two_Root(t *testing.T) {

	yml := `openapi: 3.0
security:
  - BingoDingo: [admin]
  - JingoLingo: [admin]
paths:
  /hot/{dog}:
    get:
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "operation_security", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 2)
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

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	ops := make(map[string]string)
	ops["schemesPath"] = "$.components.securitySchemes"

	rule := buildOpenApiTestRuleAction(path, "operation_security", "", ops)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), ops)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 2)
}
