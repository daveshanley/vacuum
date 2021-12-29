package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPostResponseSuccess_GetSchema(t *testing.T) {
	def := PostResponseSuccess{}
	assert.Equal(t, "operation_response_success", def.GetSchema().Name)
}

func TestPostResponseSuccess_RunRule(t *testing.T) {
	def := PostResponseSuccess{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestPostResponseSuccess_RunRule_Success(t *testing.T) {

	yml := `paths:
  /fish/cake:
    post:
      responses:
        '200':
          description: yeah
        '201':
          description: uh
        '202':
          description: no`

	path := "$.paths.*.post.responses"

	nodes, _ := utils.FindNodes([]byte(yml), path)
	opts := make(map[string][]string)
	opts["properties"] = []string{"200", "201", "202"}

	rule := buildOpenApiTestRuleAction(path, "post_response_success", "", opts)
	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Options:    opts,
	}

	def := PostResponseSuccess{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestPostResponseSuccess_RunRule_Fail(t *testing.T) {

	yml := `paths:
  /fish/cake:
    post:
      responses:
        '302':
          description: wat
        '500':
          description: b0rked
        '404':
          description: gone`

	path := "$.paths.*.post.responses"

	nodes, _ := utils.FindNodes([]byte(yml), path)
	opts := make(map[string][]string)
	opts["properties"] = []string{"200", "201", "202"}

	rule := buildOpenApiTestRuleAction(path, "post_response_success", "", opts)
	ctx := model.RuleFunctionContext{
		RuleAction: model.CastToRuleAction(rule.Then),
		Options:    opts,
	}

	def := PostResponseSuccess{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "operations must define a success response with one of the "+
		"following codes: '200, 201, 202'", res[0].Message)
}
