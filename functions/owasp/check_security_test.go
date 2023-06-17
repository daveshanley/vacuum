package owasp

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
)

func TestCheckSecurity_GetSchema(t *testing.T) {
	def := CheckSecurity{}
	assert.Equal(t, "check_security", def.GetSchema().Name)
}

func TestCheckSecurity_RunRule(t *testing.T) {
	def := CheckSecurity{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestCheckSecurity_SecurityMissing(t *testing.T) {

	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
paths:
  /security-gloabl-ok-put:
    put:
      responses: {}
  /security-ok-put:
    put:
      responses: {}
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "check_security", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), map[string]interface{}{
		"methods": []string{"put"},
	})

	def := CheckSecurity{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}
