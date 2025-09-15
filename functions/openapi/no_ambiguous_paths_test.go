package openapi

import (
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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

// https://github.com/daveshanley/vacuum/issues/703
func TestAmbiguousPaths_DifferentMethods_NotAmbiguous(t *testing.T) {
	// Test case for GitHub issue #703
	// /cars/{carId} GET and /cars/start POST should NOT be ambiguous
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/cars/{carId}':
    get:
      summary: Get a car by ID
      parameters:
        - name: carId
          in: path
          required: true
          schema:
            type: string
  '/cars/start':
    post:
      summary: Start car service
  '/users/{userId}':
    get:
      summary: Get user by ID
  '/users/admin':
    put:
      summary: Update admin user`

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

	// Create DrDocument for method-aware checking
	doc, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)
	v3Model, modelErrors := doc.BuildV3Model()
	assert.Len(t, modelErrors, 0)
	drDocument := drModel.NewDrDocument(v3Model)
	ctx.DrDocument = drDocument

	def := AmbiguousPaths{}
	res := def.RunRule(nodes, ctx)

	// These paths should NOT be ambiguous because they use different HTTP methods:
	// - /cars/{carId} (GET) vs /cars/start (POST) - different methods
	// - /users/{userId} (GET) vs /users/admin (PUT) - different methods
	assert.Len(t, res, 0, "Paths with different HTTP methods should not be ambiguous")
}

func TestAmbiguousPaths_SameMethodsAmbiguous(t *testing.T) {
	// Test case for same paths with same methods - should be ambiguous
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/cars/{carId}':
    get:
      summary: Get a car by ID
      parameters:
        - name: carId
          in: path
          required: true
          schema:
            type: string
  '/cars/start':
    get:
      summary: Get car start status
  '/api/{version}/users':
    post:
      summary: Create user
  '/api/{v}/users':
    post:
      summary: Create user (alt version)`

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

	// Create DrDocument for method-aware checking
	doc, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)
	v3Model, modelErrors := doc.BuildV3Model()
	assert.Len(t, modelErrors, 0)
	drDocument := drModel.NewDrDocument(v3Model)
	ctx.DrDocument = drDocument

	def := AmbiguousPaths{}
	res := def.RunRule(nodes, ctx)

	// These paths SHOULD be ambiguous because they use the same HTTP methods:
	// - /cars/{carId} (GET) vs /cars/start (GET) - same method, carId could match "start"
	// - /api/{version}/users (POST) vs /api/{v}/users (POST) - same method, same structure
	assert.Len(t, res, 2, "Paths with same HTTP methods and ambiguous structure should be detected")
}

func TestAmbiguousPaths_MultipleMethodsOnSamePath(t *testing.T) {
	// Test case where the same path has multiple methods defined
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/users/{userId}':
    get:
      summary: Get user by ID
    put:
      summary: Update user by ID
    delete:
      summary: Delete user by ID
  '/users/profile':
    get:
      summary: Get current user profile
    post:
      summary: Update current user profile
  '/users/settings':
    put:
      summary: Update user settings`

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

	// Create DrDocument for method-aware checking
	doc, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)
	v3Model, modelErrors := doc.BuildV3Model()
	assert.Len(t, modelErrors, 0)
	drDocument := drModel.NewDrDocument(v3Model)
	ctx.DrDocument = drDocument

	def := AmbiguousPaths{}
	res := def.RunRule(nodes, ctx)

	// Expected ambiguous combinations for same methods:
	// - /users/{userId} (GET) vs /users/profile (GET) - userId could match "profile"
	// - /users/{userId} (PUT) vs /users/settings (PUT) - userId could match "settings"
	// Non-ambiguous combinations due to different methods:
	// - /users/{userId} (PUT) vs /users/profile (POST) - different methods
	// - /users/{userId} (DELETE) vs /users/settings (PUT) - different methods
	assert.Len(t, res, 2, "Only paths with same methods should be ambiguous")

	// Verify both results contain method information
	if len(res) >= 2 {
		// Check that one is GET and one is PUT
		messages := []string{res[0].Message, res[1].Message}
		var hasGet, hasPut bool
		for _, msg := range messages {
			if strings.Contains(msg, "(GET)") {
				hasGet = true
			}
			if strings.Contains(msg, "(PUT)") {
				hasPut = true
			}
		}
		assert.True(t, hasGet, "Should have GET method ambiguity")
		assert.True(t, hasPut, "Should have PUT method ambiguity")
	}
}

func TestAmbiguousPaths_ComplexMethodCombinations(t *testing.T) {
	// Test complex scenario with multiple ambiguous and non-ambiguous combinations
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/api/{version}/data':
    get:
      summary: Get data
    post:
      summary: Create data
  '/api/v1/data':
    get:
      summary: Get v1 data
    delete:
      summary: Delete v1 data
  '/api/{ver}/data':
    put:
      summary: Update data`

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

	// Create DrDocument for method-aware checking
	doc, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)
	v3Model, modelErrors := doc.BuildV3Model()
	assert.Len(t, modelErrors, 0)
	drDocument := drModel.NewDrDocument(v3Model)
	ctx.DrDocument = drDocument

	def := AmbiguousPaths{}
	res := def.RunRule(nodes, ctx)

	// Expected ambiguous combinations:
	// 1. /api/{version}/data (GET) vs /api/v1/data (GET) - version could match "v1"
	// Not ambiguous:
	// - /api/{version}/data (POST) vs /api/v1/data (DELETE) - different methods
	// - /api/{version}/data (GET) vs /api/{ver}/data (PUT) - different methods
	// - /api/{version}/data (POST) vs /api/{ver}/data (PUT) - different methods
	assert.Len(t, res, 1, "Should detect only GET method ambiguity")

	if len(res) > 0 {
		assert.Contains(t, res[0].Message, "(GET)", "Should specify GET method in result")
		assert.Contains(t, res[0].Message, "/api/{version}/data", "Should mention parameterized path")
		assert.Contains(t, res[0].Message, "/api/v1/data", "Should mention literal path")
	}
}
