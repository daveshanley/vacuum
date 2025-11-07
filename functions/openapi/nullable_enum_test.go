package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestNullableEnum_GetSchema(t *testing.T) {
	def := NullableEnum{}
	assert.Equal(t, "nullableEnum", def.GetSchema().Name)
}

func TestNullableEnum_RunRule(t *testing.T) {
	def := NullableEnum{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestNullableEnum_RunRule_OAS3_Success(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    GoodNullableEnum:
      type: string
      nullable: true
      enum: [active, inactive, null]
    RegularEnum:
      type: string
      enum: [yes, no]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "nullable_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NullableEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestNullableEnum_RunRule_OAS3_MissingNull(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    BadNullableEnum:
      type: string
      nullable: true
      enum: [active, inactive]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "nullable_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NullableEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "nullable but does not contain a `null` value")
}

func TestNullableEnum_RunRule_OAS3_StringNull(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    BadNullableEnum:
      type: string
      nullable: true
      enum: [active, inactive, "null"]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "nullable_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NullableEnum{}
	res := def.RunRule(nodes, ctx)

	// String "null" is not the same as actual null
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "not the string \"null\"")
}

func TestNullableEnum_RunRule_OAS31_Success(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    GoodNullableEnum:
      type: [string, "null"]
      enum: [active, inactive, null]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "nullable_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NullableEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestNullableEnum_RunRule_OAS31_MissingNull(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    BadNullableEnum:
      type: [string, "null"]
      enum: [active, inactive]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "nullable_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NullableEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "nullable but does not contain a `null` value")
}

func TestNullableEnum_RunRule_MultipleViolations(t *testing.T) {
	yml := `openapi: 3.0.3
paths:
  /test:
    get:
      parameters:
        - name: status
          in: query
          schema:
            type: string
            nullable: true
            enum: [active, inactive]
components:
  schemas:
    Status:
      type: string
      nullable: true
      enum: [pending, complete]
    Priority:
      type: [string, "null"]
      enum: [low, medium, high]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "nullable_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NullableEnum{}
	res := def.RunRule(nodes, ctx)

	// Should find 3 violations: query param, Status schema, and Priority schema
	assert.Len(t, res, 3)
}

func TestNullableEnum_RunRule_NonNullable_NoViolation(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    RegularEnum:
      type: string
      enum: [active, inactive]
    ExplicitNotNullable:
      type: string
      nullable: false
      enum: [yes, no]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "nullable_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NullableEnum{}
	res := def.RunRule(nodes, ctx)

	// Non-nullable enums should not trigger violations
	assert.Len(t, res, 0)
}

func TestNullableEnum_RunRule_NoEnum_NoViolation(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    NullableString:
      type: string
      nullable: true
    NullableInteger:
      type: integer
      nullable: true`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "nullable_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NullableEnum{}
	res := def.RunRule(nodes, ctx)

	// Nullable schemas without enums should not trigger violations
	assert.Len(t, res, 0)
}
