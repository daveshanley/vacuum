package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// TestAmbiguousPaths_Issue644 tests the case from issue #644
// where /foo/{x} and /foo/bar should be flagged as ambiguous
func TestAmbiguousPaths_Issue644(t *testing.T) {
	yml := `openapi: 3.0.0
paths:
  '/foo/{x}':
    get:
      summary: Path with variable
  '/foo/bar':
    get:
      summary: Path with literal`

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

	// This should flag an ambiguity
	assert.Len(t, res, 1, "Should detect ambiguity between /foo/{x} and /foo/bar")
	if len(res) > 0 {
		assert.Contains(t, res[0].Message, "ambiguous")
		assert.Contains(t, res[0].Message, "/foo/{x}")
		assert.Contains(t, res[0].Message, "/foo/bar")
	}
}

// TestAmbiguousPaths_MoreCases tests additional ambiguous path cases
func TestAmbiguousPaths_MoreCases(t *testing.T) {
	yml := `openapi: 3.0.0
paths:
  '/users/{id}':
    get:
      summary: Get user by ID
  '/users/me':
    get:
      summary: Get current user
  '/posts/{postId}/comments/{commentId}':
    get:
      summary: Get specific comment
  '/posts/featured/comments/latest':
    get:
      summary: Get latest comment on featured post
  '/items/{name}':
    get:
      summary: Get item by name
  '/items/special':
    get:
      summary: Get special item`

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

	// Should detect multiple ambiguities:
	// 1. /users/{id} vs /users/me
	// 2. /posts/{postId}/comments/{commentId} vs /posts/featured/comments/latest
	// 3. /items/{name} vs /items/special
	assert.GreaterOrEqual(t, len(res), 3, "Should detect at least 3 ambiguous path pairs")
}

// TestAmbiguousPaths_NoFalsePositives tests that we don't flag non-ambiguous paths
func TestAmbiguousPaths_NoFalsePositives(t *testing.T) {
	yml := `openapi: 3.0.0
paths:
  '/users/{id}/profile':
    get:
      summary: Get user profile
  '/users/{id}/settings':
    get:
      summary: Get user settings
  '/products/electronics':
    get:
      summary: Get electronics
  '/products/books':
    get:
      summary: Get books`

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

	// These paths are NOT ambiguous - they have different literal segments
	assert.Len(t, res, 0, "Should not detect any ambiguous paths")
}