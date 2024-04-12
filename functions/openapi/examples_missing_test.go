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

func TestExamplesMissing_GetSchema(t *testing.T) {
	def := ExamplesMissing{}
	assert.Equal(t, "examples_missing", def.GetSchema().Name)
}

func TestExamplesMissing_RunRule(t *testing.T) {
	def := ExamplesMissing{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestExamplesMissing(t *testing.T) {

	yml := `openapi: 3.1
paths:
  /pizza:
    get:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              description: I need an example`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "schema is missing `examples` or `example`", res[0].Message)
	assert.Contains(t, res[0].Path, "$.paths['/pizza'].get.requestBody.content['application/json']")
}

func TestExamplesMissing_TrainTravel(t *testing.T) {

	yml := `openapi: 3.1
paths:
  /trips:
    get:
      parameters:
        - name: origin
          in: query
          schema:
            type: string
            format: uuid
          example: efdbb9d1-02c2-4bc3-afb7-6788d8782b1ee`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
	//assert.Equal(t, "schema is missing `examples` or `example`", res[0].Message)
	//assert.Contains(t, res[0].Path, "$.paths['/pizza'].get.requestBody.content['application/json']")
}

func TestExamplesMissing_Alt(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Pizza:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 3)
	assert.Equal(t, "schema is missing `examples` or `example`", res[0].Message)
	assert.Contains(t, res[1].Path, "$.components.schemas['Pizza'].properties")
}

func TestExamplesMissing_Header(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /cake:
    get:
      responses:
        '200':
          headers:
            bingo:
              description: I need an example`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "header is missing `examples` or `example`", res[0].Message)
	assert.Equal(t, "$.paths['/cake'].get.responses['200'].headers['bingo']", res[0].Path)

}

func TestExamplesMissing_Header_Alt(t *testing.T) {
	yml := `openapi: 3.1
components:
  headers:
    Cake:
      description: I need an example`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "header is missing `examples` or `example`", res[0].Message)
	assert.Equal(t, "$.components.headers['Cake']", res[0].Path)

}

func TestExamplesMissing_MediaType(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /herbs:
    get:
      requestBody:
        content:
          application/json:
            description: I need an example`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "media type is missing `examples` or `example`", res[0].Message)
	assert.Equal(t, "$.paths['/herbs'].get.requestBody.content['application/json']", res[0].Path)

}

func TestExamplesMissing_MediaType_Alt(t *testing.T) {
	yml := `openapi: 3.1
components:
  requestBodies:
    Herbs:
      content:
        slapsication/json:
          description: I need an example`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "media type is missing `examples` or `example`", res[0].Message)
	assert.Equal(t, "$.components.requestBodies['Herbs'].content['slapsication/json']", res[0].Path)

}

func TestExamplesMissing_MediaType_EmptyArray(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Herby:
      type: object
      examples:
        -
`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema is missing `examples` or `example`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Herby']", res[0].Path)

}
