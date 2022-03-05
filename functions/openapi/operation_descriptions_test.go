package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperationDescription_GetSchema(t *testing.T) {
	def := OperationDescription{}
	assert.Equal(t, "operation_description", def.GetSchema().Name)
}

func TestOperationDescription_RunRule(t *testing.T) {
	def := OperationDescription{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperationDescription_CheckDescriptionMissing(t *testing.T) {

	yml := `paths:
  /fish/paste:
    get:
      responses:
        '200':
          description: hi
    put:
      responses:
        '200':
          description: bye            
    post:
      description: this is a description that is great and 10 words long at least
      responses:
        '200':
          description: bye      `

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}

func TestOperationDescription_CheckDescriptionTooShort(t *testing.T) {

	yml := `paths:
  /fish/paste:
    post:
      description: this is a thing that does nothing
      responses:
        '200':
          description: bye`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)

}
