package openapi_functions

import (
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUniqueOperationId_GetSchema(t *testing.T) {
	def := UniqueOperationId{}
	assert.Equal(t, "unique_operation_id", def.GetSchema().Name)
}

func TestUniqueOperationId_RunRule(t *testing.T) {
	def := UniqueOperationId{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestUniqueOperationId_RunRule_DuplicateId(t *testing.T) {

	yml := `paths:
  /melody:
    post:
      operationId: littleSong
  /maddox:
    get:
      operationId: littleChampion
  /ember:
    get:
      operationId: littleSong`

	path := "$.paths"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := UniqueOperationId{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the 'get' operation at path '/ember' contains a duplicate operationId 'littleSong'", res[0].Message)
}

func TestUniqueOperationId_RunRule_MissingId_AndDuplicate(t *testing.T) {

	yml := `paths:
  /melody:
    post:
      operationId: littleSong
  /maddox:
    get:
  /ember:
    get:
      operationId: littleSong`

	path := "$.paths"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := UniqueOperationId{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}

func TestUniqueOperationId_RunRule_MissingId(t *testing.T) {

	yml := `paths:
  /melody:
    post:
      operationId: littleSong
  /maddox:
    get:
  /ember:
    get:
      operationId: littleMenace`

	path := "$.paths"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := UniqueOperationId{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the 'get' operation at path '/maddox' does not contain an operationId", res[0].Message)
}

func TestUniqueOperationId_RunRule_Success(t *testing.T) {

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

	path := "$.paths"

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := UniqueOperationId{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}
