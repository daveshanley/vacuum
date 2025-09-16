package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"strings"
	"testing"
)

func TestUnusedComponent_GetSchema(t *testing.T) {
	def := UnusedComponent{}
	assert.Equal(t, "oasUnusedComponent", def.GetSchema().Name)
}

func TestUnusedComponent_RunRule(t *testing.T) {
	def := UnusedComponent{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestUnusedComponent_RunRule_Success(t *testing.T) {

	yml := `
openapi: 3.0.0
paths:
  /naughty/{puppy}:
    parameters:
      - $ref: '#/components/parameters/Chewy'
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Puppy'
components:
  schemas:
    Puppy:
      type: string
      description: pup
  parameters:
    Chewy:
      description: chewy
      in: query
      name: chewy`

	res := setupUnusedComponentsTestContext(t, yml)
	assert.Len(t, res, 0)
}

func TestUnusedComponent_RunRule_SuccessSwaggerSecurity(t *testing.T) {

	yml := `swagger: 2.0
securityDefinitions:
  basicAuth:
    type: basic
  sessionAuth:
    type: apiKey
    in: header
    name: X-API-Key
paths:
  "/store/inventory":
    get:
      security:
        - basicAuth: []
  "/store/inventory/doSomething":
    get:
      security:
        - sessionAuth: []`

	res := setupUnusedComponentsTestContext(t, yml)

	assert.Len(t, res, 0)
}

func TestUnusedComponent_RunRule_SuccessOpenAPISecurity(t *testing.T) {

	yml := `openapi: 3.0.1
info:
  description: A test spec with a security def that is not a ref!
security:
  - SomeSecurity: []
components:
  securitySchemes:
    SomeSecurity:
      description: A secure way to do things and stuff.`
	res := setupUnusedComponentsTestContext(t, yml)
	assert.Len(t, res, 0)
}

func TestUnusedComponent_RunRule_Success_Fail_TwoMissing_Two_Undefined(t *testing.T) {

	yml := `
openapi: 3.0.1
parameters:
  Chewy:
    description: chewy
    in: query
    name: chewy
paths:
  /naughty/{puppy}:
    parameters:
      - $ref: '#/parameters/Nothing'
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Cupcakes_And_Sugar'
components:
  schemas:
    Puppy:
      type: string
      description: pup
    Kitty:
      $ref: '#/components/schemas/Puppy' `

	res := setupUnusedComponentsTestContext(t, yml)
	assert.Len(t, res, 4)
}

func TestUnusedComponent_RunRule_Success_Fail_Four_Undefined(t *testing.T) {

	yml := `
openapi: 3.0.0
paths:
  /naughty/{puppy}:
    parameters:
      - $ref: '#/components/parameters/Chewy'
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Puppy'
components:
  responses:
    Chappy:
      type: string
      description: Chappy
  schemas:  
    Chippy:
      type: string
      description: chippy
    Puppy:
      type: string
      description: pup
    Kitty:
      $ref: '#/components/schemas/Puppy'
  parameters:
    Minty:
      description: minty
      in: header
      name: minty
    Chewy:
      description: chewy
      in: query
      name: chewy`

	res := setupUnusedComponentsTestContext(t, yml)
	assert.Len(t, res, 4)
}

func TestUnusedComponent_RunRule_Success_PolymorphicCheck(t *testing.T) {

	yml := `
openapi: 3.0.0
paths:
  /naughty/{puppy}:
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              oneOf:
                - $ref: '#/components/schemas/Puppy'
      "404":
        description: The naughty kitty
        content:
          application/json:
            schema:
              anyOf:
                - $ref: '#/components/schemas/Kitty'
      "500":
        description: The naughty bunny
        content:
          application/json:
            schema:
              allOf:
                - $ref: '#/components/schemas/Bunny'
components:
  schemas:
    Puppy:
      type: string
      description: pup
    Kitty:
      type: string
      description: kitty
    Bunny:
      type: string
      description: bunny`

	res := setupUnusedComponentsTestContext(t, yml)
	assert.Len(t, res, 0)
}

func TestUnusedComponent_RunRule_Success_PolymorphicCheckAllOf(t *testing.T) {

	yml := `
openapi: 3.0.0
paths:
  "/naughty/{puppy}":
    get:
      responses:
        "200":
          description: The naughty pup
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Dog"
components:
  schemas:
    Dog:
      type: object
      allOf:
        - $ref: "#/components/schemas/Pet"
      required:
        - breed
      properties:
        breed:
          $ref: "#/components/schemas/Breed"
    Pet:
      type: object
      properties:
        id:
          type: String
          description: Unique identifier of the Pet
      age:
        type: integer
        format: int64
    Breed:
      type: object
      properties:
        name:
          type: String
        category:
          type: String`

	res := setupUnusedComponentsTestContext(t, yml)
	assert.Len(t, res, 0)
}

// TestUnusedComponent_RunRule_PanicWithSpaceInSchemaKey tests the fix for GitHub issue #682
// where a schema key containing a space would cause an "index out of range [-1]" panic.
// See: https://github.com/daveshanley/vacuum/issues/682
func TestUnusedComponent_RunRule_PanicWithSpaceInSchemaKey(t *testing.T) {
	yml := `{
    "openapi": "3.1.0",
    "info": {
      "title": "API",
      "description": "API Specification",
      "termsOfService": "",
      "contact": {
        "email": ""
      },
      "version": "1.0.0"
    },
    "servers": [
      {
        "url": "/v1/"
      }
    ],
    "tags": [

    ],
    "paths": {
    },
    "components": {
      "schemas": {
        "Error Message": {
          "title": "Error Message",
          "type": "object",
          "properties": {
            "error": {
              "type": "string",
              "description": "HTTP error",
              "example": "Bad Request"
            },
            "message": {
              "type": "string",
              "description": "Error message",
              "example": "Something horrific has happened!"
            },
            "path": {
              "type": "string",
              "description": "Request path",
              "example": "/example/resource"
            },
            "status": {
              "type": "integer",
              "description": "HTTP status code",
              "format": "int32",
              "example": 400
            },
            "timestamp": {
              "type": "string",
              "description": "Timestamp. Indicates when the error message was created.",
              "format": "date-time",
              "example": "1970-01-01T12:00:00Z"
            },
            "trace": {
              "type": "string",
              "description": "Traceback (Not used)"
            }
          }
        }
      }
    }
  }`

	// This should not panic and should complete successfully
	res := setupUnusedComponentsTestContext(t, yml)

	// The schema should be used (referenced in the response), so no unused components
	assert.Len(t, res, 1)
}

func setupUnusedComponentsTestContext(t *testing.T, yml string) []model.RuleFunctionResult {
	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	if strings.Contains(yml, "swagger") {
		_, _ = document.BuildV2Model()
	} else {
		_, _ = document.BuildV3Model()
	}

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "unused_component", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	info, _ := datamodel.ExtractSpecInfo([]byte(yml))
	ctx.SpecInfo = info
	ctx.Document = document

	def := UnusedComponent{}

	return def.RunRule(nodes, ctx)
}
