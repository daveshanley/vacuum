package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestPolymorphicOneOf_GetSchema(t *testing.T) {
	def := PolymorphicOneOf{}
	assert.Equal(t, "polymorphic_oneOf", def.GetSchema().Name)
}

func TestPolymorphicOneOf_RunRule(t *testing.T) {
	def := PolymorphicOneOf{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestPolymorphicOneOf_RunRule_Fail(t *testing.T) {

	yml := `components:
  schemas:
    Melody:
      type: object
      properties:
        schema:
          oneOf:
            - $ref: '#/components/schemas/Maddy'
    Maddy:
      type: string`

	path := "$"

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(yml), &rootNode)

	rule := buildOpenApiTestRuleAction(path, "polymorphic_oneOf", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	ctx.Index = model.NewSpecIndex(&rootNode)

	def := PolymorphicOneOf{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 1)
}

func TestPolymorphicOneOf_RunRule_Success(t *testing.T) {

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
	yaml.Unmarshal([]byte(yml), &rootNode)

	rule := buildOpenApiTestRuleAction(path, "polymorphic_oneOf", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	ctx.Index = model.NewSpecIndex(&rootNode)

	def := PolymorphicOneOf{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 0)
}
