package owasp

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
)

func TestDefineErrorDefinition_GetSchema(t *testing.T) {
	def := DefineErrorDefinition{}
	assert.Equal(t, "define_error_definition", def.GetSchema().Name)
}

func TestDefineErrorDefinition_RunRule(t *testing.T) {
	def := DefineErrorDefinition{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestDefineErrorDefinition_ErrorDefinitionMissing(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "422":
          description: "classic validation fail"
`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "define_error_definition", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := DefineErrorDefinition{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}
