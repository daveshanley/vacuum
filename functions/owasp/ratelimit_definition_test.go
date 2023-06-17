package owasp

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitDefinition_GetSchema(t *testing.T) {
	def := RateLimitDefinition{}
	assert.Equal(t, "ratelimit_definition", def.GetSchema().Name)
}

func TestRateLimitDefinition_RunRule(t *testing.T) {
	def := RateLimitDefinition{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestRateLimitDefinition_CheckDescriptionMissing(t *testing.T) {

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
        error
      450:
        headers:
          "X-RateLimit-Limit"`

	path := "$.paths..responses"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "ratelimit_definition", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := RateLimitDefinition{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 4)

}
