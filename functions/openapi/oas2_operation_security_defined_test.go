package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
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
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "oas2_operation_security_defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OAS2OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)
}

func TestOAS2OperationSecurityDefined_RunRule_Success_Root_Security(t *testing.T) {

	yml := `swagger: 2.0
security:
  - littleChampion: []
paths:
  /melody:
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
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "oas2_operation_security_defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OAS2OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)
}

func TestOAS2OperationSecurityDefined_RunRule_Success_Root_SecurityAllRoot(t *testing.T) {

	yml := `swagger: 2.0
security:
  - littleChampion: []
  - littleSong: []		
paths:
  /melody:
    get:      
securityDefinitions:
  littleChampion:
    type: basic
  littleSong:
    type: apiKey
    in: header
    name: X-API-KEY`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "oas2_operation_security_defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OAS2OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)
}

func TestOAS2OperationSecurityDefined_RunRule_Fail_Root_SecurityAllRoot(t *testing.T) {

	yml := `swagger: 2.0
security:
  - littleScreamer: []
  - littleTantrum: []		
paths:
  /melody:
    get:      
securityDefinitions:
  littleChampion:
    type: basic
  littleSong:
    type: apiKey
    in: header
    name: X-API-KEY`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "oas2_operation_security_defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OAS2OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 2)
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
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "oas2_operation_security_defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OAS2OperationSecurityDefined{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 1)
}
