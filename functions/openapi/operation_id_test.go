package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperationId_GetSchema(t *testing.T) {
	def := OperationId{}
	assert.Equal(t, "oasOpId", def.GetSchema().Name)
}

func TestOperationId_RunRule(t *testing.T) {
	def := OperationId{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperationId_RunRule_Fail(t *testing.T) {

	yml := `openapi: 3.0.1
paths:
  /melody:
    post:
      operationId: littleSong
  /maddox:
    get:
  /ember:
    get:
      operationId: littleMenace`

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

	def := OperationId{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "the `GET` operation does not contain an `operationId`", res[0].Message)
}

func TestOperationId_RunRule_Success(t *testing.T) {

	yml := `openapi: 3.0.1
paths:
  /melody:
    post:
      operationId: littleSong
  /maddox:
    get:
      operationId: littleChampion
  /ember:
    get:
      operationId: littleMenace`

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

	def := OperationId{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}
