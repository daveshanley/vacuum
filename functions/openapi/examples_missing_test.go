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
              description: I need an example
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

	assert.Len(t, res, 4)
	assert.Equal(t, "schema is missing examples", res[0].Message)
	assert.Equal(t, "$.paths['/pizza'].get.requestBody.content['application/json'].schema", res[0].Path)
	assert.Equal(t, "$.components.schemas['Pizza']", res[1].Path)
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
              description: I need an example
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

	assert.Len(t, res, 2)
	assert.Equal(t, "header is missing examples", res[0].Message)
	assert.Equal(t, "$.paths['/cake'].get.responses['200'].headers['bingo']", res[0].Path)
	assert.Equal(t, "$.components.headers['Cake']", res[1].Path)

}

func TestExamplesMissing_MediaType(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /herbs:
    get:
      requestBody:
        content:
          application/json:
            description: I need an example
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

	assert.Len(t, res, 2)
	assert.Equal(t, "media type is missing examples", res[0].Message)
	assert.Equal(t, "$.paths['/herbs'].get.requestBody.content['application/json']", res[0].Path)
	assert.Equal(t, "$.components.requestBodies['Herbs'].content['slapsication/json']", res[1].Path)

}
