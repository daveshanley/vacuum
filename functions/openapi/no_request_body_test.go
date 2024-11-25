package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNoRequestBody_GetSchema(t *testing.T) {
	def := NoRequestBody{}
	assert.Equal(t, "noRequestBody", def.GetSchema().Name)
}

func TestNoRequestBody_RunRule(t *testing.T) {
	def := NoRequestBody{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestNoRequestBody_RunRule_Fail(t *testing.T) {

	yml := `openapi: 3.0.1
paths:
  /melody:
    post:
      requestBody:
        description: "the body of the request"
        content:
          application/json:
            schema:
              properties:
                id:
                  type: string
  /maddox:
    get:
      requestBody:
        description: "the body of the request"
        content:
          application/json:
            schema:
              properties:
                id:
                  type: string
    delete:
      requestBody:
        description: "the body of the request"
        content:
          application/json:
            schema:
              properties:
                id:
                  type: string
  /ember:
    get:
      requestBody:
        description: "the body of the request"
        content:
          application/json:
            schema:
              properties:
                id:
                  type: string
`
	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "responses", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := NoRequestBody{}
	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 3)
}

func TestNoRequestBody_RunRule_Success(t *testing.T) {

	yml := `openapi: 3.0.1
paths:
  /melody:
    post:
      requestBody:
        description: "the body of the request"
        content:
          application/json:
            schema:
              properties:
                id:
                  type: string
  /maddox:
    post:
      requestBody:
        description: "the body of the request"
        content:
          application/json:
            schema:
              properties:
                id:
                  type: string
  /ember:
    patch:
      requestBody:
        description: "the body of the request"
        content:
          application/json:
            schema:
              properties:
                id:
                  type: string
`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "responses", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := NoRequestBody{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}
