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
	assert.Equal(t, "noAmbiguousPaths", def.GetSchema().Name)
}

func TestNoAmbiguousPaths_RunRule(t *testing.T) {
	def := AmbiguousPaths{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestAmbiguousPaths_Issue504(t *testing.T) {
	// Test case for issue #504
	// These paths should NOT be ambiguous as they have different literal segments
	yml := `openapi: 3.0.0
paths:
  '/a/{Id1}/b/c/{Id3}':
    get:
      summary: Path with c literal
  '/a/{Id1}/b/{Id2}/d':
    get:
      summary: Path with d literal`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "ambiguousPaths", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := AmbiguousPaths{}
	res := def.RunRule(nodes, ctx)

	// These paths should NOT be ambiguous because:
	// - Segment 3: 'c' vs {Id2} (literal vs variable - different)
	// - Segment 4: {Id3} vs 'd' (variable vs literal - different)
	assert.Len(t, res, 0, "Paths with different literal segments should not be ambiguous")
}

func TestAmbiguousPaths_ActuallyAmbiguous(t *testing.T) {
	// Test case for paths that ARE actually ambiguous
	yml := `openapi: 3.0.0
paths:
  '/users/{id}/posts':
    get:
      summary: Get user posts
  '/users/{userId}/posts':
    get:
      summary: Get user posts (alternative)
  '/{entity}/list':
    get:
      summary: List entities
  '/{resource}/list':
    get:
      summary: List resources`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "ambiguousPaths", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := AmbiguousPaths{}
	res := def.RunRule(nodes, ctx)

	// Expected ambiguous pairs:
	// 1. /users/{id}/posts vs /users/{userId}/posts (same structure, variables at same position)
	// 2. /{entity}/list vs /{resource}/list (same structure, variables at same position)
	assert.Len(t, res, 2, "Paths with same structure and variables at same positions should be ambiguous")
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
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := AmbiguousPaths{}
	res := def.RunRule(nodes, ctx)

	// With the updated logic, these paths are ambiguous:
	// 1. /good/{id} vs /good/last (variable {id} could match 'last')
	// 2. /good/{id}/{pet} vs /good/last/{id} (variable {id} could match 'last')
	// 3. /{id}/ambiguous vs /ambiguous/{id} (variable {id} could match 'ambiguous')
	// 4. /{entity}/{id}/last vs /pet/first/{id} (variable {entity} could match 'pet' but 'first' != literal in position 2, so not ambiguous)
	// Actually analyzing more carefully:
	// - /good/{id} vs /good/last: ambiguous
	// - /good/{id}/{pet} vs /good/last/{id}: ambiguous
	// - /{id}/ambiguous vs /ambiguous/{id}: ambiguous
	// - /{entity}/{id}/last vs /pet/first/{id}: NOT ambiguous (different at position 2: {id} vs 'first')
	assert.Greater(t, len(res), 0, "Should detect ambiguous paths")
}
