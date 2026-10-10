package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"testing"
)

func TestUniqueOperationId_GetSchema(t *testing.T) {
	def := UniqueOperationId{}
	assert.Equal(t, "oasOpIdUnique", def.GetSchema().Name)
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
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

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
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

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
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := UniqueOperationId{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)

}

func TestUniqueOperationId_RunRule_DuplicateId_SortedPaths(t *testing.T) {

	yml := `paths:
  /z-last:
    get:
      operationId: sharedId
  /m-middle:
    delete:
      operationId: sharedId
  /a-first:
    post:
      operationId: sharedId`

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction("$", "unique_operation_id", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	want := []string{
		"$.paths['/m-middle'].delete",
		"$.paths['/z-last'].get",
	}

	def := UniqueOperationId{}
	var first []string
	for i := 0; i < 30; i++ {
		res := def.RunRule(rootNode.Content, ctx)
		assert.Len(t, res, 2)
		got := []string{res[0].Path, res[1].Path}
		if i == 0 {
			assert.Equal(t, want, got)
			assert.Equal(t, "the 'delete' operation at path '/m-middle' contains a duplicate operationId 'sharedId'", res[0].Message)
			assert.Equal(t, "the 'get' operation at path '/z-last' contains a duplicate operationId 'sharedId'", res[1].Message)
			first = append([]string(nil), got...)
			continue
		}
		assert.Equal(t, first, got, "iteration %d", i)
	}
}
