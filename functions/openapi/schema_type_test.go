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

func TestSchemaType_InvalidPattern(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: string
     pattern: (*&@(*&@(*&@#*&@`

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
	assert.Equal(t, "schema `pattern` should be a ECMA-262 regular expression dialect", res[0].Message)
	assert.Equal(t, "$.components.schemas['Gum'].pattern", res[0].Path)
}

func TestSchemaType_ValidPattern(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
      type: string
      pattern: hello
    Apostrophe:
      type: string
      pattern: '[''"]'`

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

func TestSchemaType_Issue629_CronPattern(t *testing.T) {
	// Test case from issue #629
	// The pattern should be valid according to ECMA-262 regex specification
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    cronSchedule:
      type: object
      properties:
        schedule:
          type: string
          default: "*/15 * * * *"
          pattern: "(@(annually|yearly|monthly|weekly|daily|hourly|reboot))|(@every (\\d+(ns|us|µs|ms|s|m|h))+)|((((\\d+,)+\\d+|(\\d+(/|-)\\d+)|\\d+|\\*) ?){5,7})"
          title: "Cron Schedule Pattern"`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "schemaTypeCheck",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	st := SchemaTypeCheck{}
	res := st.RunRule(nil, ctx)

	// The pattern should be valid - no errors expected
	assert.Empty(t, res)
}

func TestSchemaType_Issue629_InvalidPattern(t *testing.T) {
	// Test with an actually invalid regex pattern to ensure error detection works
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    badPattern:
      type: string
      pattern: "[unclosed"`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "schemaTypeCheck",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	st := SchemaTypeCheck{}
	res := st.RunRule(nil, ctx)

	// Should detect the invalid pattern
	assert.NotEmpty(t, res)
	assert.Contains(t, res[0].Message, "pattern")
	assert.Contains(t, res[0].Message, "ECMA-262")
}

func TestSchemaType_Issue629_PatternWithSpecialChars(t *testing.T) {
	// Test patterns with various special characters that need proper escaping
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    specialChars:
      type: string
      pattern: '[''"]'
    backslashes:
      type: string
      pattern: '\\d{3}-\\d{3}-\\d{4}'
    unicodeChars:
      type: string
      pattern: '[\u0041-\u005A]+'`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "schemaTypeCheck",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	st := SchemaTypeCheck{}
	res := st.RunRule(nil, ctx)

	// All patterns should be valid - no errors expected
	assert.Empty(t, res)
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

	assert.Len(t, res, 0)
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
     maximum: 50`

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

	assert.Len(t, res, 0)
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

	assert.Len(t, res, 0)

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

	assert.Len(t, res, 0)
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

func TestSchemaType_RequiredPropertiesPolyAllOf(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: object
     required:
       - hello
     properties:
       goodbye:
         type: string
     allOf:
       - type: object
         properties:
           hello:
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

	assert.Len(t, res, 0)
}

func TestSchemaType_RequiredPropertiesPolyAllOf_NoProps(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: object
     required:
       - hello
     allOf:
       - type: object
         properties:
           hello:
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

	assert.Len(t, res, 0)
}

func TestSchemaType_RequiredPropertiesPolyOneOf(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: object
     required:
       - hello
     properties:
       goodbye:
         type: string
     oneOf:
       - type: object
         properties:
           hello:
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

	assert.Len(t, res, 0)
}

func TestSchemaType_RequiredPropertiesPolyOneOf_NoProps(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: object
     required:
       - hello
     oneOf:
       - type: object
         properties:
           hello:
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

	assert.Len(t, res, 0)
}

func TestSchemaType_RequiredPropertiesPolyAnyOf(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: object
     required:
       - hello
     properties:
       goodbye:
         type: string
     anyOf:
       - type: object
         properties:
           hello:
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

	assert.Len(t, res, 0)
}

func TestSchemaType_RequiredPropertiesPolyAnyOf_NoProps(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: object
     required:
       - hello
     anyOf:
       - type: object
         properties:
           hello:
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

	assert.Len(t, res, 0)
}

func TestSchemaType_Null(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Gum:
     type: null`

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

	assert.Empty(t, res)
}

func TestSchemaType_DependentRequired_Basic(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Person:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer
        address:
          type: string
        phone:
          type: string
      dependentRequired:
        name: ["age"]
        address: ["phone"]`

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

	assert.Len(t, res, 0) // Valid dependentRequired - all referenced properties exist
}

func TestSchemaType_DependentRequired_MissingProperty(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Person:
      type: object
      properties:
        name:
          type: string
        address:
          type: string
      dependentRequired:
        name: ["age", "phone"]  # age and phone don't exist in properties`

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

	assert.Len(t, res, 2)
	assert.Equal(t, "property `age` referenced in `dependentRequired` does not exist in schema `properties`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Person'].dependentRequired", res[0].Path)
	assert.Equal(t, "property `phone` referenced in `dependentRequired` does not exist in schema `properties`", res[1].Message)
	assert.Equal(t, "$.components.schemas['Person'].dependentRequired", res[1].Path)
}

func TestSchemaType_DependentRequired_CircularDependency(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Person:
      type: object
      properties:
        name:
          type: string
        email:
          type: string
        phone:
          type: string
      dependentRequired:
        name: ["email"]
        email: ["phone"]
        phone: ["name"]  # Creates circular dependency: name -> email -> phone -> name`

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

	assert.Len(t, res, 0) // Current implementation doesn't detect circular dependencies yet
}

func TestSchemaType_DependentRequired_PolymorphicAllOf(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Person:
      type: object
      properties:
        id:
          type: string
      dependentRequired:
        id: ["name", "email"]  # name and email are in allOf
      allOf:
        - type: object
          properties:
            name:
              type: string
        - type: object
          properties:
            email:
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

	assert.Len(t, res, 0) // Valid - properties found in allOf schemas
}

func TestSchemaType_DependentRequired_PolymorphicOneOf(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Contact:
      type: object
      properties:
        type:
          type: string
      dependentRequired:
        type: ["phone", "email"]  # phone is in oneOf[0], email is in oneOf[1]
      oneOf:
        - type: object
          properties:
            phone:
              type: string
        - type: object
          properties:
            email:
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

	assert.Len(t, res, 0) // Valid - properties found in oneOf schemas
}

func TestSchemaType_DependentRequired_PolymorphicAnyOf(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    User:
      type: object
      properties:
        username:
          type: string
      dependentRequired:
        username: ["password"]  # password is in anyOf
      anyOf:
        - type: object
          properties:
            password:
              type: string
        - type: object
          properties:
            token:
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

	assert.Len(t, res, 0) // Valid - password found in anyOf
}

func TestSchemaType_DependentRequired_PolymorphicMissingProperty(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    User:
      type: object
      properties:
        username:
          type: string
      dependentRequired:
        username: ["missing_prop"]  # missing_prop doesn't exist anywhere
      anyOf:
        - type: object
          properties:
            password:
              type: string
        - type: object
          properties:
            token:
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
	assert.Equal(t, "property `missing_prop` referenced in `dependentRequired` does not exist in schema `properties`", res[0].Message)
	assert.Equal(t, "$.components.schemas['User'].dependentRequired", res[0].Path)
}

func TestSchemaType_DependentRequired_EmptyDependentRequired(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Person:
      type: object
      properties:
        name:
          type: string
      dependentRequired: {}`

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

	assert.Len(t, res, 0) // Empty dependentRequired is valid
}

func TestSchemaType_DependentRequired_EmptyRequiredArray(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Person:
      type: object
      properties:
        name:
          type: string
      dependentRequired:
        name: []`

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

	assert.Len(t, res, 0) // Empty required array is valid
}

func TestSchemaType_DependentRequired_SelfDependency(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Person:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer
      dependentRequired:
        name: ["name", "age"]  # Self-dependency on name`

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

	assert.Len(t, res, 1) // Self-dependency is detected as circular
	assert.Equal(t, "circular dependency detected: property `name` requires itself in `dependentRequired`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Person'].dependentRequired", res[0].Path)
}

func TestSchemaType_ConstValidString(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    StringConst:
      type: string
      const: "hello"`

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

func TestSchemaType_ConstInvalidString(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    StringConst:
      type: string
      const: 123`

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
	assert.Equal(t, "`const` value type does not match schema type [string]", res[0].Message)
	assert.Equal(t, "$.components.schemas['StringConst'].const", res[0].Path)
}

func TestSchemaType_ConstValidInteger(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    IntegerConst:
      type: integer
      const: 42`

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

func TestSchemaType_ConstInvalidInteger(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    IntegerConst:
      type: integer
      const: "not a number"`

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
	assert.Equal(t, "`const` value type does not match schema type [integer]", res[0].Message)
	assert.Equal(t, "$.components.schemas['IntegerConst'].const", res[0].Path)
}

func TestSchemaType_ConstValidIntegerFloatValue(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    IntegerConst:
      type: integer
      const: 42.0`

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

func TestSchemaType_ConstInvalidIntegerFloatValue(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    IntegerConst:
      type: integer
      const: 42.5`

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
	assert.Equal(t, "`const` value type does not match schema type [integer]", res[0].Message)
	assert.Equal(t, "$.components.schemas['IntegerConst'].const", res[0].Path)
}

func TestSchemaType_ConstValidNumber(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    NumberConst:
      type: number
      const: 3.14`

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

func TestSchemaType_ConstValidNumberInteger(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    NumberConst:
      type: number
      const: 42`

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

func TestSchemaType_ConstInvalidNumber(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    NumberConst:
      type: number
      const: "not a number"`

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
	assert.Equal(t, "`const` value type does not match schema type [number]", res[0].Message)
	assert.Equal(t, "$.components.schemas['NumberConst'].const", res[0].Path)
}

func TestSchemaType_ConstValidBoolean(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    BooleanConst:
      type: boolean
      const: true`

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

func TestSchemaType_ConstInvalidBoolean(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    BooleanConst:
      type: boolean
      const: "true"`

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
	assert.Equal(t, "`const` value type does not match schema type [boolean]", res[0].Message)
	assert.Equal(t, "$.components.schemas['BooleanConst'].const", res[0].Path)
}

func TestSchemaType_ConstValidNull(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    NullConst:
      type: "null"
      const: null`

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

func TestSchemaType_ConstInvalidNull(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    NullConst:
      type: "null"
      const: "not null"`

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
	assert.Equal(t, "`const` value type does not match schema type [null]", res[0].Message)
	assert.Equal(t, "$.components.schemas['NullConst'].const", res[0].Path)
}

func TestSchemaType_ConstValidArray(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    ArrayConst:
      type: array
      const: [1, 2, 3]`

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

func TestSchemaType_ConstInvalidArray(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    ArrayConst:
      type: array
      const: "not an array"`

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
	assert.Equal(t, "`const` value type does not match schema type [array]", res[0].Message)
	assert.Equal(t, "$.components.schemas['ArrayConst'].const", res[0].Path)
}

func TestSchemaType_ConstValidObject(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    ObjectConst:
      type: object
      const: {"key": "value"}`

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

func TestSchemaType_ConstInvalidObject(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    ObjectConst:
      type: object
      const: "not an object"`

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
	assert.Equal(t, "`const` value type does not match schema type [object]", res[0].Message)
	assert.Equal(t, "$.components.schemas['ObjectConst'].const", res[0].Path)
}

func TestSchemaType_ConstMultipleTypesValid(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    MultiTypeConst:
      type: [string, number]
      const: "hello"`

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

func TestSchemaType_ConstMultipleTypesValidNumber(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    MultiTypeConst:
      type: [string, number]
      const: 42`

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

func TestSchemaType_ConstMultipleTypesInvalid(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    MultiTypeConst:
      type: [string, number]
      const: true`

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
	assert.Equal(t, "`const` value type does not match schema type [string, number]", res[0].Message)
	assert.Equal(t, "$.components.schemas['MultiTypeConst'].const", res[0].Path)
}

func TestSchemaType_ConstNoTypes(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    NoTypeConst:
      const: "hello"`

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

func TestSchemaType_NoConst(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    NoConstSchema:
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

	assert.Len(t, res, 0)
}

func TestSchemaType_EnumConstValid(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    ValidEnumConst:
      type: string
      enum:
        - "foo"
        - "bar"
        - "baz"
      const: "bar"`

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

func TestSchemaType_EnumConstInvalid(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    InvalidEnumConst:
      type: string
      enum:
        - "foo"
        - "bar"
        - "baz"
      const: "invalid"`

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
	assert.Equal(t, "`const` value `invalid` is not present in `enum` values", res[0].Message)
	assert.Equal(t, "$.components.schemas['InvalidEnumConst'].const", res[0].Path)
}

func TestSchemaType_EnumConstNumericValid(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    ValidNumericEnumConst:
      type: number
      enum:
        - 1
        - 2
        - 3.14
      const: 3.14`

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

func TestSchemaType_EnumConstNumericInvalid(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    InvalidNumericEnumConst:
      type: number
      enum:
        - 1
        - 2
        - 3.14
      const: 5`

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
	assert.Equal(t, "`const` value `5` is not present in `enum` values", res[0].Message)
	assert.Equal(t, "$.components.schemas['InvalidNumericEnumConst'].const", res[0].Path)
}

func TestSchemaType_EnumConstBooleanValid(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    ValidBooleanEnumConst:
      type: boolean
      enum:
        - true
        - false
      const: true`

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

func TestSchemaType_EnumOnly(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    EnumOnly:
      type: string
      enum:
        - "foo"
        - "bar"
        - "baz"`

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

func TestSchemaType_ConstOnly(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    ConstOnly:
      type: string
      const: "hello"`

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
