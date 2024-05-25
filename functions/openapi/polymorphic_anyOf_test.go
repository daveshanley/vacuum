package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestPolymorphicAnyOf_GetSchema(t *testing.T) {
	def := PolymorphicAnyOf{}
	assert.Equal(t, "oasPolymorphicAnyOf", def.GetSchema().Name)
}

func TestPolymorphicAnyOf_RunRule(t *testing.T) {
	def := PolymorphicAnyOf{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestPolymorphicAnyOf_RunRule_Fail(t *testing.T) {

	yml := `components:
  schemas:
    Melody:
      type: object
      properties:
        schema:
          anyOf:
            - $ref: '#/components/schemas/Maddy'
    Maddy:
      type: string`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "polymorphic_anyOf", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PolymorphicAnyOf{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 1)
}

func TestPolymorphicAnyOf_RunRule_Success(t *testing.T) {

	yml := `components:
  schemas:
    Melody:
      type: object
      properties:
        schema:
          $ref: '#/components/schemas/Maddy'
    Maddy:
      type: string`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "polymorphic_anyOf", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PolymorphicAnyOf{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)
}
