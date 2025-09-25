// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, "$.components.schemas['Herbs'].examples[0]", res[1].Path)

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

/*
components:
  schemas:
    Test:
      type: array
      description: Test array with numbers
      items:
        type: number
      example:
        - 0 # <- This gives a warning
        - 0
        - 0
*/

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

// https://github.com/daveshanley/vacuum/issues/615
func TestExamplesSchema_HandleArrays(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas: 
    Test:
      type: array
      description: Test array with numbers
      items:
        type: number
      example:          
        - 0
        - 0
        - 0`

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

// TestExamplesSchema_OpenAPI30_Nullable demonstrates that nullable: true works correctly in OpenAPI 3.0
// See https://github.com/daveshanley/vacuum/issues/710
// See https://github.com/daveshanley/vacuum/issues/603
func TestExamplesSchema_OpenAPI30_Nullable(t *testing.T) {
	yml := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
          nullable: true
      example:
        name: null`

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

	// should pass - nullable: true is valid in OpenAPI 3.0
	assert.Len(t, res, 0)
}

// TestExamplesSchema_OpenAPI31_NullableInvalid demonstrates that nullable: true fails in OpenAPI 3.1
// See https://github.com/daveshanley/vacuum/issues/710
// See https://github.com/daveshanley/vacuum/issues/603
func TestExamplesSchema_OpenAPI31_NullableInvalid(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
          nullable: true
      example:
        name: null`

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

	// should fail - nullable: true is not valid in OpenAPI 3.1
	assert.Greater(t, len(res), 0)
	assert.Contains(t, res[0].Message, "JSON schema compile failed: OpenAPI keyword 'nullable': The `nullable` keyword is not supported in OpenAPI 3.1+. Use `type: ['string', 'null']`")
}

// TestExamplesSchema_OpenAPI31_ProperNullable demonstrates proper nullable syntax in OpenAPI 3.1
// See https://github.com/daveshanley/vacuum/issues/710
// See https://github.com/daveshanley/vacuum/issues/603
func TestExamplesSchema_OpenAPI31_ProperNullable(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: [string, "null"]
      example:
        name: null`

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

	// should pass - type: [string, "null"] is the correct OpenAPI 3.1 syntax
	assert.Len(t, res, 0)
}

// TestExamplesSchema_Issue520_OneOfNonDiscriminant tests the specific issue from #520
// where oneOf validation was not being reported due to non-discriminant alternatives
func TestExamplesSchema_Issue520_OneOfNonDiscriminant(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Issue 520 Test
  version: 1.0.0
components:
  schemas:
    Test:
      type: object
      oneOf:
        - properties:
            pim:
              type: string
        - properties:
            pam:
              type: string
      example:
        pam: nop`

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

	// Should detect oneOf violation - example matches both alternatives
	// The example {pam: "nop"} matches:
	// 1. First alternative (allows objects with "pim" property - no required fields)
	// 2. Second alternative (allows objects with "pam" property - example has "pam")
	// Since oneOf requires exactly one match, this should be invalid
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "'oneOf' failed")
	assert.Contains(t, res[0].Message, "subschemas 0, 1 matched")
	assert.Equal(t, "$.components.schemas['Test'].example", res[0].Path)
}
