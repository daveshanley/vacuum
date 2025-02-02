// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExamplesSchema(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Herbs:
      type: object
      properties:
        id:
          type: string
      examples:
        - id: smoked`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}

func TestExamplesSchema_TrainTravel(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Station:
      type: object
      properties:
        id:
          type: string
          format: uuid
          examples:
            - efdbb9d1-02c2-4bc3-afb7-6788d8782b1e
            - b2e783e1-c824-4d63-b37a-d8d698862f1d`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}

func TestExamplesSchema_Invalid(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Herbs:
      type: object
      properties:
        id:
          type: string
      additionalProperties: false
      examples:
        - id: smoked
          name: illegal`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "additional properties 'name' not allowed", res[0].Message)
	assert.Equal(t, "$.components.schemas['Herbs'].examples[0]", res[0].Path)

}

func TestExamplesSchema_Valid_OneOf(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Herbs:
      type: object
      properties:
        id:
          oneOf:
            - type: string
              const: smoked
            - type: integer
              const: 1
      examples:
        - id: smoked`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}

func TestExamplesSchema_Valid_OneOf_Int(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Herbs:
      type: object
      properties:
        id:
          oneOf:
            - type: string
              const: smoked
            - type: integer
              const: 1
      examples:
        - id: 1`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}

func TestExamplesSchema_Invalid_OneOf(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Herbs:
      type: object
      properties:
        id:
          oneOf:
            - type: string
              const: smoked
            - type: integer
              const: 1
      examples:
        - id: eaten`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "value must be 'smoked'", res[0].Message)
	assert.Equal(t, "$.components.schemas['Herbs'].examples[0]", res[0].Path)
	assert.Equal(t, "got string, want integer", res[1].Message)
	assert.Equal(t, "$.components.schemas['Herbs'].examples[0]", res[01].Path)

}

func TestExamplesSchema_ExampleProp(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Herbs:
      type: object
      properties:
        id:
          oneOf:
            - type: string
              const: smoked
            - type: integer
              const: 1
      example:
        id: smoked`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}

func TestExamplesSchema_ExampleProp_Failed(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Herbs:
      type: object
      properties:
        id:
          oneOf:
            - type: string
              const: smoked
            - type: integer
              const: 1
      example:
        id: baked`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "value must be 'smoked'", res[0].Message)
	assert.Equal(t, "$.components.schemas['Herbs'].example", res[0].Path)
	assert.Equal(t, "got string, want integer", res[1].Message)
	assert.Equal(t, "$.components.schemas['Herbs'].example", res[1].Path)

}

func TestExamplesSchema_Param_Valid(t *testing.T) {
	yml := `openapi: 3.1
components:
  parameters:
    Herbs:
      in: header
      name: herbs
      schema:
        type: object
        properties:
          id:
            type: string
            const: spicy
      examples:
        - id: spicy`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}

func TestExamplesSchema_Param_Invalid(t *testing.T) {
	yml := `openapi: 3.1
components:
  parameters:
    Herbs:
      in: header
      name: herbs
      schema:
        type: object
        properties:
          id:
            type: string
            const: spicy
      examples:
        sammich:
          value:
            id: crispy`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "value must be 'spicy'", res[0].Message)
	assert.Equal(t, "$.components.parameters['Herbs'].examples['sammich']", res[0].Path)

}

func TestExamplesSchema_Header_Invalid(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /herbs:
    get:
      responses:
        "200":
          headers:
            "Herbs":
              schema:
                type: string
                const: tasty
              examples:
                sammich:
                  value: crispy
                  
      `

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "value must be 'tasty'", res[0].Message)
	assert.Equal(t, "$.paths['/herbs'].get.responses['200'].headers['Herbs'].examples['sammich']", res[0].Path)

}

func TestExamplesSchema_MT_Invalid(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /herbs:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                type: string
                const: tasty
              examples:
                sammich:
                  value: crispy
                  
      `

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "value must be 'tasty'", res[0].Message)
	assert.Equal(t, "$.paths['/herbs'].get.responses['200'].content['application/json'].examples['sammich']", res[0].Path)

}

func TestExamplesSchema_HandleJSONTime(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    badDate:
      type: string
      description: a bad time.
      format: date-time
      example: 2022-08-07T12:12:00Z`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesSchema{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
