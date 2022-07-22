package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestOperationId_GetSchema(t *testing.T) {
	def := OperationId{}
	assert.Equal(t, "operation_id", def.GetSchema().Name)
}

func TestOperationId_RunRule(t *testing.T) {
	def := OperationId{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperationId_RunRule_Fail(t *testing.T) {

	yml := `paths:
  /melody:
    post:
      operationId: littleSong
  /maddox:
    get:
  /ember:
    get:
      operationId: littleMenace`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationId{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the 'get' operation at path '/maddox' does not contain an operationId", res[0].Message)
}

func TestOperationId_RunRule_Success(t *testing.T) {

	yml := `paths:
  /melody:
    post:
      operationId: littleSong
  /maddox:
    get:
      operationId: littleChampion
  /ember:
    get:
      operationId: littleMenace`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Index = index.NewSpecIndex(&rootNode)

	def := OperationId{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)

}
