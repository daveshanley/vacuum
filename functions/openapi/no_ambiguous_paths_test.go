package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestNoAmbiguousPaths_GetSchema(t *testing.T) {
	def := AmbiguousPaths{}
	assert.Equal(t, "ambiguousPaths", def.GetSchema().Name)
}

func TestNoAmbiguousPaths_RunRule(t *testing.T) {
	def := AmbiguousPaths{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestAmbiguousPaths_RunRule_SuccessCheck(t *testing.T) {

	yml := `openapi: 3.0.0
paths:
  '/good/{id}':
    get:
      summary: List all pets
  '/good/last':
    get:
      summary: List all pets
  '/good/{id}/{pet}':
    get:
      summary: List all pets
  '/good/last/{id}':
    get:
      summary: List all pets
  '/{id}/ambiguous':
    get:
      summary: List all pets
  '/ambiguous/{id}':
    get:
      summary: List all pets
  '/pet/last':
    get:
      summary: List all pets
  '/pet/first':
    get:
      summary: List all pets
  '/{entity}/{id}/last':
    get:
      summary: List all pets
  '/pet/first/{id}':
    get:
      summary: List all pets`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "ambiguousPaths", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	ctx.Index = index.NewSpecIndex(&rootNode)

	def := AmbiguousPaths{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 3)
}
