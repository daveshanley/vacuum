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

func TestSchemaType_Invalid(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: gummy`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "unknown schema type: `gummy`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].type", res[0].Path)
}

func TestSchemaType_InvalidMinLength(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: string
     minLength: -10`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`minLength` should be a non-negative number", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].minLength", res[0].Path)
}

func TestSchemaType_InvalidMaxLength(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: string
     maxLength: -10`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`maxLength` should be a non-negative number", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].maxLength", res[0].Path)
}

func TestSchemaType_InvalidMaxLength_Oversize(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: string
     maxLength:  5
     minLength: 10`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`maxLength` should be greater than or equal to `minLength`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].maxLength", res[0].Path)
}

func TestSchemaType_InvalidFormat(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: string
     format: (*&@(*&@(*&@#*&@`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "schema `format` should be a ECMA-262 regular expression dialect", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].format", res[0].Path)
}

func TestSchemaType_ValidFormat(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: string
     format: hello`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 0)
}

func TestSchemaType_MultipleOf(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     multipleOf: -2`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`multipleOf` should be a number greater than `0`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].multipleOf", res[0].Path)
}

func TestSchemaType_Minimum(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     minimum: -2`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`minimum` should be a number greater than or equal to `0`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].minimum", res[0].Path)
}

func TestSchemaType_Minimum_Zero(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     minimum: 0`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 0)
}

func TestSchemaType_Maximum_Zero(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     maximum: 0`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 0)
}

func TestSchemaType_Maximum(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     maximum: 0`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 0)
}

func TestSchemaType_Maximum_Negative(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     maximum: -50`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`maximum` should be a number greater than or equal to `0`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].maximum", res[0].Path)
}

func TestSchemaType_MinMaximum(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     maximum: 5
     minimum: 10`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`maximum` should be a number greater than or equal to `minimum`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].maximum", res[0].Path)
}

func TestSchemaType_ExclusiveMinimum(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     exclusiveMinimum: -5`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`exclusiveMinimum` should be a number greater than or equal to `0`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].exclusiveMinimum", res[0].Path)
}

func TestSchemaType_ExclusiveMinimum_Zero(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     exclusiveMinimum: 0`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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
	assert.Len(t, res, 0)

}

func TestSchemaType_ExclusiveMaximum(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     exclusiveMaximum: -5`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`exclusiveMaximum` should be a number greater than or equal to `0`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].exclusiveMaximum", res[0].Path)
}

func TestSchemaType_ExclusiveMaximum_Zero(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     exclusiveMaximum: 0`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 0)
}

func TestSchemaType_ExclusiveMinMaximum(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: number
     exclusiveMaximum: 4
     exclusiveMinimum: 10`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`exclusiveMaximum` should be greater than or equal to `exclusiveMinimum`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].exclusiveMaximum", res[0].Path)
}

func TestSchemaType_MinItems(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: array
     minItems: -5`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`minItems` should be a non-negative number", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].minItems", res[0].Path)
}

func TestSchemaType_MaxItems(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: array
     maxItems: -5`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`maxItems` should be a non-negative number", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].maxItems", res[0].Path)
}

func TestSchemaType_MinMaxItems(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: array
     maxItems: 4
     minItems: 7`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`maxItems` should be greater than or equal to `minItems`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].maxItems", res[0].Path)
}

func TestSchemaType_MinContains(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: array
     minContains: -5`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`minContains` should be a non-negative number", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].minContains", res[0].Path)
}

func TestSchemaType_MaxContains(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: array
     maxContains: -5`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`maxContains` should be a non-negative number", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].maxContains", res[0].Path)
}

func TestSchemaType_MinMaxContains(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: array
     maxContains: 6
     minContains: 10`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`maxContains` should be greater than or equal to `minContains`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].maxContains", res[0].Path)
}

func TestSchemaType_MinProperties(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: object
     minProperties: -5`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`minProperties` should be a non-negative number", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].minProperties", res[0].Path)
}

func TestSchemaType_MaxProperties(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: object
     maxProperties: -5`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`maxProperties` should be a non-negative number", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].maxProperties", res[0].Path)
}

func TestSchemaType_MinMaxProperties(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: object
     minProperties: 3
     maxProperties: 2`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`maxProperties` should be greater than or equal to `minProperties`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].maxProperties", res[0].Path)
}

func TestSchemaType_RequiredProperties(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: object
     required:
       - hello
     properties:
       goodbye:
         type: string`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

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

	assert.Len(t, res, 1)
	assert.Equal(t, "`required` field `hello` is not defined in `properties`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].required[0]", res[0].Path)
}
