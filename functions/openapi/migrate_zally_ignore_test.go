package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestMigrateZallyIgnore_GetSchema(t *testing.T) {
	def := MigrateZallyIgnore{}
	assert.Equal(t, "migrateZallyIgnore", def.GetSchema().Name)
}

func TestMigrateZallyIgnore_RunRule_NoNodes(t *testing.T) {
	def := MigrateZallyIgnore{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestMigrateZallyIgnore_RunRule_NoZallyIgnore(t *testing.T) {
	yml := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint
      x-lint-ignore: some-rule`

	path := "$"
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "migrateZallyIgnore", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := MigrateZallyIgnore{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestMigrateZallyIgnore_RunRule_SingleZallyIgnore(t *testing.T) {
	yml := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint
      x-zally-ignore: some-rule`

	path := "$"
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "migrateZallyIgnore", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := MigrateZallyIgnore{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Convert ignore rules to use x-lint-ignore", res[0].Message)
	assert.Equal(t, "$.paths./test.get.x-zally-ignore", res[0].Path)
}

func TestMigrateZallyIgnore_RunRule_MultipleZallyIgnore(t *testing.T) {
	yml := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  x-zally-ignore: info-rule
paths:
  /test:
    get:
      summary: Test endpoint
      x-zally-ignore: operation-rule
    post:
      summary: Another endpoint
      x-zally-ignore: [rule-one, rule-two]
components:
  schemas:
    TestSchema:
      type: object
      x-zally-ignore: schema-rule`

	path := "$"
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "migrateZallyIgnore", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := MigrateZallyIgnore{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 4)
	expectedPaths := []string{
		"$.info.x-zally-ignore",
		"$.paths./test.get.x-zally-ignore",
		"$.paths./test.post.x-zally-ignore",
		"$.components.schemas.TestSchema.x-zally-ignore",
	}
	for i, result := range res {
		assert.Equal(t, "Convert ignore rules to use x-lint-ignore", result.Message)
		assert.Equal(t, expectedPaths[i], result.Path)
	}
}

func TestMigrateZallyIgnore_RunRule_NestedZallyIgnore(t *testing.T) {
	yml := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    TestSchema:
      type: object
      properties:
        field1:
          type: string
          x-zally-ignore: field-rule
        field2:
          type: object
          properties:
            nestedField:
              type: string
              x-zally-ignore: nested-rule`

	path := "$"
	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "migrateZallyIgnore", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := MigrateZallyIgnore{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
	expectedPaths := []string{
		"$.components.schemas.TestSchema.properties.field1.x-zally-ignore",
		"$.components.schemas.TestSchema.properties.field2.properties.nestedField.x-zally-ignore",
	}
	for i, result := range res {
		assert.Equal(t, "Convert ignore rules to use x-lint-ignore", result.Message)
		assert.Equal(t, expectedPaths[i], result.Path)
	}
}
