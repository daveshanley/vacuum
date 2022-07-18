package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

	path := "$"

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(yml), &rootNode)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := UniqueOperationId{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 1)
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

	path := "$"

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(yml), &rootNode)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := UniqueOperationId{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 1)
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

	path := "$"
	var rootNode yaml.Node
	yaml.Unmarshal([]byte(yml), &rootNode)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := UniqueOperationId{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)

}
