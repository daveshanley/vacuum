package openapi

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// Commented out OAS2 tests - focusing on OpenAPI 3+ only
// func TestOAS2ParameterDescription_GetSchema(t *testing.T) {
// 	def := ParameterDescription{}
// 	assert.Equal(t, "oasParamDescriptions", def.GetSchema().Name)
// }

func TestParameterDescription_GetSchema(t *testing.T) {
	def := ParameterDescription{}
	assert.Equal(t, "oasParamDescriptions", def.GetSchema().Name)
}

func TestParameterDescription_GetCategory(t *testing.T) {
	def := ParameterDescription{}
	assert.Equal(t, model.FunctionCategoryOpenAPI, def.GetCategory())
}

func TestParameterDescription_RunRule(t *testing.T) {
	def := ParameterDescription{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

// func TestOAS2ParameterDescription_RunRule(t *testing.T) {
// 	def := ParameterDescription{}
// 	res := def.RunRule(nil, model.RuleFunctionContext{})
// 	assert.Len(t, res, 0)
// }

// func TestOAS2ParameterDescription_RunRule_Success(t *testing.T) {

// 	yml := `swagger: 2.0
// paths:
//   /melody:
//     post:
//       parameters:
//         - in: header
//           name: blue-eyes
//           description: beautiful girl
// parameters:
//   Maddy:
//    in: header
//    name: little champion
//    description: beautiful boy`

// 	path := "$"

// 	var rootNode yaml.Node
// 	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
// 	assert.NoError(t, mErr)

// 	rule := buildOpenApiTestRuleAction(path, "oas2_parameter_description", "", nil)
// 	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
// 	config := index.CreateOpenAPIIndexConfig()
// 	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

// 	def := ParameterDescription{}
// 	res := def.RunRule(rootNode.Content, ctx)

// 	assert.Len(t, res, 0)
// }

// func TestOAS2ParameterDescription_RunRule_Fail(t *testing.T) {

// 	yml := `swagger: 2.0
// paths:
//   /melody:
//     post:
//       parameters:
//         - in: header
//           name: blue-eyes
// parameters:
//   Maddy:
//    in: header
//    name: little champion`

// 	path := "$"

// 	var rootNode yaml.Node
// 	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
// 	assert.NoError(t, mErr)

// 	rule := buildOpenApiTestRuleAction(path, "oas2_parameter_description", "", nil)
// 	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
// 	config := index.CreateOpenAPIIndexConfig()
// 	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

// 	def := ParameterDescription{}
// 	res := def.RunRule(rootNode.Content, ctx)

// 	assert.Len(t, res, 2)
// }

// func TestOAS2ParameterDescription_RunRule_Success_NoIn(t *testing.T) {

// 	yml := `swagger: 2.0
// paths:
//   /melody:
//     post:
//       parameters:
//         - name: blue-eyes
// parameters:
//   Maddy:
//    name: little champion`

// 	path := "$"

// 	var rootNode yaml.Node
// 	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
// 	assert.NoError(t, mErr)

// 	rule := buildOpenApiTestRuleAction(path, "oas2_parameter_description", "", nil)
// 	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
// 	config := index.CreateOpenAPIIndexConfig()
// 	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

// 	def := ParameterDescription{}
// 	res := def.RunRule(rootNode.Content, ctx)

// 	assert.Len(t, res, 0)
// }

// func TestOAS2ParameterDescription_RunRule_Fail_EmptyDescription(t *testing.T) {

// 	yml := `swagger: 2.0
// paths:
//   /melody:
//     post:
//       parameters:
//         - in: header
//           name: blue-eyes
//           description:
// parameters:
//   Maddy:
//    in: header
//    name: little champion
//    description:`

// 	path := "$"

// 	var rootNode yaml.Node
// 	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
// 	assert.NoError(t, mErr)

// 	rule := buildOpenApiTestRuleAction(path, "oas2_parameter_description", "", nil)
// 	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
// 	config := index.CreateOpenAPIIndexConfig()
// 	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

// 	def := ParameterDescription{}
// 	res := def.RunRule(rootNode.Content, ctx)

// 	assert.Len(t, res, 2)
// }

// Test with DrDocument - Component Parameters
func TestParameterDescription_ComponentParams_MissingDescription(t *testing.T) {
	yml := `openapi: 3.1
components:
  parameters:
    UserID:
      name: userId
      in: path
      required: true
    AuthToken:
      name: token
      in: header
      description: Authentication token for API access
    PageSize:
      name: pageSize
      in: query
      description: ""`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "parameter-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ParameterDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "the parameter `UserID` does not contain a description", res[0].Message)
	assert.Equal(t, "the parameter `PageSize` does not contain a description", res[1].Message)
	assert.Equal(t, "$.components.parameters['UserID']", res[0].Path)
	assert.Equal(t, "$.components.parameters['PageSize']", res[1].Path)
}

// Test Path-level Parameters
func TestParameterDescription_PathParams_MissingDescription(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /users/{userId}:
    parameters:
      - name: userId
        in: path
        required: true
      - name: includeDeleted
        in: query
        description: Include deleted users in response
    get:
      summary: Get user by ID
  /products:
    parameters:
      - name: category
        in: query
        description: ""
      - name: sortBy
        in: query`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "parameter-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ParameterDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 3)
	// Check that we found the right parameters
	messages := []string{res[0].Message, res[1].Message, res[2].Message}
	assert.Contains(t, messages, "the parameter `userId` does not contain a description")
	assert.Contains(t, messages, "the parameter `category` does not contain a description")
	assert.Contains(t, messages, "the parameter `sortBy` does not contain a description")
}

// Test Operation Parameters
func TestParameterDescription_OperationParams_MissingDescription(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /users:
    get:
      parameters:
        - name: limit
          in: query
          description: Maximum number of users to return
        - name: offset
          in: query
    post:
      parameters:
        - name: dryRun
          in: query
          description: ""
        - name: validate
          in: query
  /products/{id}:
    put:
      parameters:
        - name: id
          in: path
          required: true
        - name: force
          in: query
          description: Force update even if conflicts exist`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "parameter-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ParameterDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 4)
	// Check for specific operation parameters
	var foundOffset, foundDryRun, foundValidate, foundId bool
	for _, r := range res {
		if r.Message == "the parameter `offset` does not contain a description" {
			foundOffset = true
			assert.Equal(t, "$.paths['/users'].get.parameters[1]", r.Path)
		}
		if r.Message == "the parameter `dryRun` does not contain a description" {
			foundDryRun = true
			assert.Equal(t, "$.paths['/users'].post.parameters[0]", r.Path)
		}
		if r.Message == "the parameter `validate` does not contain a description" {
			foundValidate = true
			assert.Equal(t, "$.paths['/users'].post.parameters[1]", r.Path)
		}
		if r.Message == "the parameter `id` does not contain a description" {
			foundId = true
			assert.Equal(t, "$.paths['/products/{id}'].put.parameters[0]", r.Path)
		}
	}
	assert.True(t, foundOffset)
	assert.True(t, foundDryRun)
	assert.True(t, foundValidate)
	assert.True(t, foundId)
}

// Test Mixed Parameters (component, path, and operation)
func TestParameterDescription_MixedParams(t *testing.T) {
	yml := `openapi: 3.1
components:
  parameters:
    GlobalAuth:
      name: Authorization
      in: header
    GlobalLimit:
      name: limit
      in: query
      description: Global limit parameter
paths:
  /items:
    parameters:
      - name: filter
        in: query
    get:
      parameters:
        - name: sort
          in: query
          description: Sort order for results
        - name: fields
          in: query
    post:
      parameters:
        - name: validate
          in: query
          description: ""`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "parameter-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ParameterDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 4)
	// Verify we caught parameters at all levels
	var foundComponent, foundPath, foundOperation, foundEmptyDesc bool
	for _, r := range res {
		if r.Path == "$.components.parameters['GlobalAuth']" {
			foundComponent = true
		}
		if r.Path == "$.paths['/items'].parameters[0]" {
			foundPath = true
		}
		if r.Path == "$.paths['/items'].get.parameters[1]" {
			foundOperation = true
		}
		if r.Message == "the parameter `validate` does not contain a description" {
			foundEmptyDesc = true
		}
	}
	assert.True(t, foundComponent)
	assert.True(t, foundPath)
	assert.True(t, foundOperation)
	assert.True(t, foundEmptyDesc)
}

// Test all operations have parameters
func TestParameterDescription_AllOperations(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /resource:
    get:
      parameters:
        - name: getParam
          in: query
    post:
      parameters:
        - name: postParam
          in: body
    put:
      parameters:
        - name: putParam
          in: query
    delete:
      parameters:
        - name: deleteParam
          in: query
    patch:
      parameters:
        - name: patchParam
          in: query
    head:
      parameters:
        - name: headParam
          in: header
    options:
      parameters:
        - name: optionsParam
          in: query
    trace:
      parameters:
        - name: traceParam
          in: query`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "parameter-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ParameterDescription{}
	res := def.RunRule(nil, ctx)

	// All 8 HTTP methods should have parameters without descriptions
	assert.Len(t, res, 8)

	// Verify we have one for each operation
	operations := []string{"get", "post", "put", "delete", "patch", "head", "options", "trace"}
	for _, op := range operations {
		found := false
		for _, r := range res {
			if r.Path == fmt.Sprintf("$.paths['/resource'].%s.parameters[0]", op) {
				found = true
				break
			}
		}
		assert.True(t, found, "Should have found parameter for %s operation", op)
	}
}

// Test parameters with references
func TestParameterDescription_WithReferences(t *testing.T) {
	yml := `openapi: 3.1
components:
  parameters:
    SharedParam:
      name: shared
      in: query
      description: A shared parameter
    MissingDescParam:
      name: missing
      in: header
paths:
  /users:
    get:
      parameters:
        - $ref: '#/components/parameters/SharedParam'
        - $ref: '#/components/parameters/MissingDescParam'
        - name: inline
          in: query`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "parameter-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ParameterDescription{}
	res := def.RunRule(nil, ctx)

	// Should find MissingDescParam in components, the resolved reference in operation, and inline parameter
	assert.Len(t, res, 3)

	foundComponentMissing := false
	foundResolvedMissing := false
	foundInline := false
	for _, r := range res {
		if r.Message == "the parameter `MissingDescParam` does not contain a description" &&
			r.Path == "$.components.parameters['MissingDescParam']" {
			foundComponentMissing = true
		}
		if r.Message == "the parameter `missing` does not contain a description" &&
			r.Path == "$.paths['/users'].get.parameters[1]" {
			// This is the resolved reference
			foundResolvedMissing = true
		}
		if r.Message == "the parameter `inline` does not contain a description" {
			foundInline = true
			// This is an inline parameter in the operation
			assert.Equal(t, "$.paths['/users'].get.parameters[2]", r.Path)
		}
	}
	assert.True(t, foundComponentMissing, "Should find MissingDescParam in components")
	assert.True(t, foundResolvedMissing, "Should find resolved reference 'missing' in operation")
	assert.True(t, foundInline, "Should find inline parameter")
}

// Test parameter without name (edge case)
func TestParameterDescription_NoName(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /test:
    get:
      parameters:
        - in: query
          description: Has description but no name
        - in: header`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "parameter-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ParameterDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the parameter `parameter[1]` does not contain a description", res[0].Message)
	assert.Equal(t, "$.paths['/test'].get.parameters[1]", res[0].Path)
}

// Test with empty document
func TestParameterDescription_EmptyComponents(t *testing.T) {
	yml := `openapi: 3.1
info:
  title: Empty API
  version: 1.0.0
paths: {}`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "parameter-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ParameterDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

// Test valid parameters (all have descriptions)
func TestParameterDescription_AllValid(t *testing.T) {
	yml := `openapi: 3.1
components:
  parameters:
    ValidParam:
      name: valid
      in: query
      description: This parameter has a description
paths:
  /users:
    parameters:
      - name: pathParam
        in: query
        description: Path level parameter with description
    get:
      parameters:
        - name: operationParam
          in: query
          description: Operation level parameter with description`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "parameter-description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ParameterDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

// Legacy test - kept for backward compatibility with Index
func TestParameterDescription_RunRule_Fail_EmptyDescription_OpenAPI3(t *testing.T) {

	yml := `openapi: 3.0
paths:
  /melody:
    post:
      parameters:
        - in: header
          name: blue-eyes
          description:  
components:
  parameters:
    Maddy:
      in: header
      name: little champion
      description:`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "oas3_parameter_description", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := ParameterDescription{}
	res := def.RunRule(rootNode.Content, ctx)

	assert.Len(t, res, 2)
}
