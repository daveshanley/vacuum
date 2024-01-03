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

func TestExamplesExternalVal(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /herbs:
    get:
      requestBody:
        content:
          application/json:
            description: I need an example
            schema:
              type: object
              properties:
                id:
                  type: string
            examples:
              herbs:
                value:
                  id: 1
                externalValue: https://pb33f.io/herbs.json
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

	def := ExamplesExternalCheck{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "media type example contains both `externalValue` and `value`", res[0].Message)
	assert.Equal(t, "$.paths['/herbs'].get.requestBody.content['application/json'].examples['herbs']", res[0].Path)

}

func TestExamplesExternalVal_Header(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /herbs:
    get:
      responses:
        "200":
          headers: 
            "minty":
              description: I need an example
              schema:
                type: object
                properties:
                  id:
                    type: string
              examples:
                herbs:
                  value:
                    id: 1
                  externalValue: https://pb33f.io/herbs.json
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

	def := ExamplesExternalCheck{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "header example contains both `externalValue` and `value`", res[0].Message)
	assert.Equal(t, "$.paths['/herbs'].get.responses['200'].headers['minty'].examples['herbs']", res[0].Path)

}

func TestExamplesExternalVal_Param(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /herbs:
    get:
      parameters:
        - in: header
          description: I need an example
          schema:
            type: object
            properties:
              id:
                type: string
          examples:
            herbs:
              value:
                id: 1
              externalValue: https://pb33f.io/herbs.json
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

	def := ExamplesExternalCheck{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "parameter example contains both `externalValue` and `value`", res[0].Message)
	assert.Equal(t, "$.paths['/herbs'].get.parameters[0].examples['herbs']", res[0].Path)

}

func TestExamplesExternalVal_Valid(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /herbs:
    get:
      parameters:
        - in: header
          description: I need an example
          schema:
            type: object
            properties:
              id:
                type: string
          examples:
            herbs:
              value:
                id: 1
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

	def := ExamplesExternalCheck{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
