package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestPathSpecificityOrder_GetSchema(t *testing.T) {
	def := PathSpecificityOrder{}
	assert.Equal(t, "pathsSpecificityOrder", def.GetSchema().Name)
}

func TestPathSpecificityOrder_RunRule_NoNodes(t *testing.T) {
	def := PathSpecificityOrder{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestPathSpecificityOrder_FlagsStaticPathAfterTemplatedPath(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/api/authentication/v1/permissions/{userId}':
    get:
      summary: Get permissions for a specific user
  '/api/authentication/v1/permissions/all':
    get:
      summary: Get all permissions for the current user`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "pathsSpecificityOrder", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathSpecificityOrder{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "$.paths['/api/authentication/v1/permissions/all']", res[0].Path)
	assert.Contains(t, res[0].Message, "/api/authentication/v1/permissions/all")
	assert.Contains(t, res[0].Message, "/api/authentication/v1/permissions/{userId}")
	assert.Contains(t, res[0].Message, "GET")
}

func TestPathSpecificityOrder_DoesNotFlagAlreadyOrderedPaths(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/api/authentication/v1/permissions/all':
    get:
      summary: Get all permissions for the current user
  '/api/authentication/v1/permissions/{userId}':
    get:
      summary: Get permissions for a specific user`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "pathsSpecificityOrder", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathSpecificityOrder{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestPathSpecificityOrder_DoesNotFlagDifferentMethods(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/cars/{carId}':
    get:
      summary: Get a car by ID
  '/cars/start':
    post:
      summary: Start a car service`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "pathsSpecificityOrder", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathSpecificityOrder{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestPathSpecificityOrder_DoesNotFlagTypedPathParameterMismatch(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/users/{userId}':
    get:
      summary: Get user by ID
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: integer
  '/users/all':
    get:
      summary: Get all users`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "pathsSpecificityOrder", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	doc, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)
	v3Model, modelErrors := doc.BuildV3Model()
	assert.NoError(t, modelErrors)
	ctx.DrDocument = drModel.NewDrDocument(v3Model)

	def := PathSpecificityOrder{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestPathSpecificityOrder_UsesFirstDifferingSegment(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  '/teams/{teamId}/members':
    get:
      summary: Get members for a team
  '/teams/all/{memberId}':
    get:
      summary: Get a member from the all-teams view`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "pathsSpecificityOrder", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := PathSpecificityOrder{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "/teams/all/{memberId}")
	assert.Contains(t, res[0].Message, "/teams/{teamId}/members")
}
