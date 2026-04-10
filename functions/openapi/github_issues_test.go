// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// TestSchemaType_Issue691_ComprehensiveFix tests the complete fix for issue #691
// This ensures that:
// 1. allOf with $ref doesn't falsely report "no properties" when properties exist
// 2. Invalid required fields ARE caught even when using allOf with $ref
func TestSchemaType_Issue691_ComprehensiveFix(t *testing.T) {
	yml := `openapi: 3.0.0
components:
  schemas:
    BaseSchema:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
    ExtendedSchema:
      type: object
      properties:
        extra:
          type: string
    CompleteSchema:
      type: object
      allOf:
        - $ref: '#/components/schemas/BaseSchema'
        - $ref: '#/components/schemas/ExtendedSchema'
      required:
        - id          # Valid - in BaseSchema
        - name        # Valid - in BaseSchema  
        - extra       # Valid - in ExtendedSchema
        - missing     # Invalid - not defined anywhere
    PartialSchema:
      type: object
      allOf:
        - $ref: '#/components/schemas/BaseSchema'
      properties:
        local:
          type: string
      required:
        - id          # Valid - in BaseSchema via allOf
        - local       # Valid - in direct properties
        - nonexistent # Invalid - not defined anywhere`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "schema-type-check", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := SchemaTypeCheck{}
	res := def.RunRule(nil, ctx)

	// Check results
	t.Logf("Total results: %d", len(res))
	for _, r := range res {
		t.Logf("Result: %s at %s", r.Message, r.Path)
	}

	// Should have exactly 2 errors: one for 'missing' and one for 'nonexistent'
	missingFieldErrors := 0
	foundMissingError := false
	foundNonexistentError := false

	for _, r := range res {
		if r.Message == "`required` field `missing` is not defined in `properties`" {
			foundMissingError = true
			missingFieldErrors++
		}
		if r.Message == "`required` field `nonexistent` is not defined in `properties`" {
			foundNonexistentError = true
			missingFieldErrors++
		}
	}

	assert.True(t, foundMissingError, "Should report error for 'missing' field in CompleteSchema")
	assert.True(t, foundNonexistentError, "Should report error for 'nonexistent' field in PartialSchema")
	assert.Equal(t, 2, missingFieldErrors, "Should have exactly 2 missing field errors")

	// Should NOT have any "object contains `required` fields but no `properties`" errors
	hasNoPropertiesError := false
	for _, r := range res {
		if r.Message == "object contains `required` fields but no `properties`" {
			hasNoPropertiesError = true
			t.Errorf("Unexpected error: %s at %s", r.Message, r.Path)
		}
	}
	assert.False(t, hasNoPropertiesError,
		"Should NOT report 'no properties' error when properties exist via allOf")
}

// TestSchemaType_Issue546_AllOfSiblingRequiredProperties ensures required fields declared
// inside a single allOf arm can be satisfied by sibling allOf arms in the composed schema.
func TestSchemaType_Issue546_AllOfSiblingRequiredProperties(t *testing.T) {
	yml := `openapi: 3.0.0
components:
  schemas:
    CloudAppDatasetCreate:
      type: object
      properties:
        status:
          type: string
    CloudAppDataset:
      allOf:
        - $ref: '#/components/schemas/CloudAppDatasetCreate'
        - type: object
          required:
            - datasetId
            - status
          properties:
            datasetId:
              type: string`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "schema-type-check", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := SchemaTypeCheck{}
	res := def.RunRule(nil, ctx)

	for _, r := range res {
		if r.Message == "`required` field `status` is not defined in `properties`" {
			t.Fatalf("unexpected issue 546 false positive at %s", r.Path)
		}
	}
}

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

	// Per OpenAPI spec and issue #749, concrete paths take precedence over templated paths
	// /foo/bar (concrete) vs /foo/{x} (templated) = single var/literal mismatch = NOT ambiguous
	assert.Len(t, res, 0, "Concrete path /foo/bar and templated path /foo/{x} should not be ambiguous per OpenAPI spec")
}

// TestAmbiguousPaths_MoreCases tests additional path cases with concrete vs templated paths
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

	// Per OpenAPI spec and issue #749, all of these have single var/literal mismatches:
	// 1. /users/{id} vs /users/me - NOT ambiguous (concrete takes precedence)
	// 2. /posts/{postId}/comments/{commentId} vs /posts/featured/comments/latest - NOT ambiguous (single mismatch)
	// 3. /items/{name} vs /items/special - NOT ambiguous (concrete takes precedence)
	assert.Len(t, res, 0, "Concrete vs templated paths should not be ambiguous per OpenAPI spec")
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

// TestIssue810_RFC6570PathStyleParameterFalsePositives tests the fix for GitHub issue #810.
// RFC 6570 URI template operators (e.g., {;id}) should not cause false positives in
// no-ambiguous-paths, path-params, or paths-kebab-case rules.
func TestIssue810_RFC6570PathStyleParameterFalsePositives(t *testing.T) {
	yml := `openapi: "3.1.0"
info:
  title: "REST API"
  version: "1.2"
  description: "Description."
paths:
  /users/{id}:
    get:
      description: "Return the user for the given ID."
      operationId: getUser
      responses:
        "200":
          content: {}
          description: "The user."
    parameters:
      - description: "The unique identifier for the user"
        in: path
        name: id
        required: true
        schema:
          type: number
  /users/{;id}:
    get:
      description: "Look up user by legacy identifier"
      operationId: lookupId
      responses:
        "308":
          content: {}
          description: "The user matching the lookup criteria."
    parameters:
      - description: "Identifier from legacy system"
        in: path
        name: id
        required: true
        schema:
          type: number`

	path := "$"

	document, dErr := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, dErr)

	m, mErr := document.BuildV3Model()
	assert.NoError(t, mErr)

	drDocument := drModel.NewDrDocument(m)

	var rootNode yaml.Node
	yErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, yErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&rootNode, config)

	// Test no-ambiguous-paths
	t.Run("no-ambiguous-paths", func(t *testing.T) {
		rule := buildOpenApiTestRuleAction(path, "ambiguousPaths", "", nil)
		ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
		ctx.Rule = &rule
		ctx.Index = idx
		ctx.Document = document
		ctx.DrDocument = drDocument

		ambiguousPaths := AmbiguousPaths{}
		res := ambiguousPaths.RunRule(nodes, ctx)

		assert.Len(t, res, 0, "no-ambiguous-paths: /users/{id} and /users/{;id} are not ambiguous (different RFC 6570 operators)")
	})

	// Test path-params
	t.Run("path-params", func(t *testing.T) {
		rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
		ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
		ctx.Rule = &rule
		ctx.Index = idx
		ctx.Document = document
		ctx.DrDocument = drDocument

		pathParams := PathParameters{}
		res := pathParams.RunRule(nodes, ctx)

		assert.Len(t, res, 0, "path-params: /users/{id} and /users/{;id} are not equivalent (different RFC 6570 operators)")
	})

	// Test paths-kebab-case
	t.Run("paths-kebab-case", func(t *testing.T) {
		rule := buildOpenApiTestRuleAction(path, "pathsKebabCase", "", nil)
		ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
		ctx.Rule = &rule
		ctx.Index = idx
		ctx.Document = document
		ctx.DrDocument = drDocument

		kebabCase := PathsKebabCase{}
		res := kebabCase.RunRule(nodes, ctx)

		assert.Len(t, res, 0, "paths-kebab-case: {;id} is a template variable, not a kebab-case violation")
	})
}
