package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDuplicatedEnum_GetSchema(t *testing.T) {
	def := DuplicatedEnum{}
	assert.Equal(t, "duplicated_enum", def.GetSchema().Name)
}

func TestDuplicatedEnum_RunRule(t *testing.T) {
	def := DuplicatedEnum{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestDuplicatedEnum_RunRule_SuccessCheck(t *testing.T) {

	yml := `openapi: 3.0
paths:
  /pizza/:
    parameters:
      - in: query
        name: party
        schema:
          type: string
          enum: [big, small]
  /cake/:
    parameters:
      - in: query
        name: icecream
        schema:
          type: string
          enum: [lots, little]        
components:
  schemas:
    YesNo:
      type: string
      enum: [yes, no]`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "duplicated_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := DuplicatedEnum{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestDuplicatedEnum_RunRule_DuplicationFail(t *testing.T) {

	yml := `openapi: 3.0
paths:
  /pizza/:
    parameters:
      - in: query
        name: party
        schema:
          type: string
          enum: [big, small, big, huge, small]
  /cake/:
    parameters:
      - in: query
        name: icecream
        schema:
          type: string
          enum: [little, little]        
components:
  schemas:
    YesNo:
      type: string
      enum: [yes, no, yes, no]`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "duplicated_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := DuplicatedEnum{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 5)
}
