package owasp

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
)

func TestHeaderDefinition_GetSchema(t *testing.T) {
	def := HeaderDefinition{}
	assert.Equal(t, "header_definition", def.GetSchema().Name)
}

func TestHeaderDefinition_RunRule(t *testing.T) {
	def := HeaderDefinition{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestHeaderDefinition_HeaderDefinitionMissing(t *testing.T) {

	yml := `paths:
  /pizza/:
    responses:
      400:
        error
      200:
        error
      299:
        error
      499:
        "Accept":
          error
      461:
        headers:
          "Content-Type":
            schema:
              type: string
      450:
        headers:
          "Accept":
            schema:
              type: string
          "Cache-Control":
            schema:
              type: string
`

	path := "$.paths..responses"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "header_definition", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), map[string]interface{}{
		"headers": [][]string{{"Accept", "Cache-Control"}, {"Content-Type"}},
	})

	def := HeaderDefinition{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 4)

}
