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

func TestPathItemRef_RunRule_AllParamsInTop(t *testing.T) {

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

	rule := buildOpenApiTestRuleAction(path, "path_parameters", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathItemReferences{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "path `/pizza/{type}/{topping}` item uses a $ref, it's technically allowed, but not a great idea", res[0].Message)
	assert.Equal(t, "$.paths['/pizza/{type}/{topping}']", res[0].Path)

}
