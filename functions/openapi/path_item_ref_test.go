// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
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

func TestPathItemRef_RunRule_RefOnPathItem_Valid(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /pizza/{type}/{topping}:
    $ref: "#/pops"
pops:
  get:
    description: pop`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "pathItemReferences", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathItemReferences{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0, "$ref on a path item is valid and should not be flagged")
}

func TestPathItemRef_RunRule_RefOnOperation_Invalid(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /pizza:
    get:
      $ref: "#/components/operations/GetPizza"
components:
  operations:
    GetPizza:
      description: get a pizza`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "pathItemReferences", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathItemReferences{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "GET")
	assert.Contains(t, res[0].Message, "/pizza")
	assert.Equal(t, "$.paths['/pizza'].get", res[0].Path)
}

func TestPathItemRef_RunRule_NoRef_NoResults(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /pizza:
    get:
      description: get a pizza
    post:
      description: create a pizza`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "pathItemReferences", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathItemReferences{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0, "inline operations should not be flagged")
}

func TestPathItemRef_RunRule_NilDrDocument(t *testing.T) {

	rule := buildOpenApiTestRuleAction("$", "pathItemReferences", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Rule = &rule

	def := PathItemReferences{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0, "nil DrDocument should return no results")
}
