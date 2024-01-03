// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFormatLimit_RunRule(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - integer
`

	document, _ := libopenapi.NewDocument([]byte(yml))

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "integer_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := IntegerFormat{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema of type `integer` must specify a format of `int32` or `int64`", res[0].Message)
	assert.Equal(t, "$.components.schemas['thing']", res[0].Path)
}

func TestFormatLimit_RunRule_WrongFormat(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - integer
      format: int16
`

	document, _ := libopenapi.NewDocument([]byte(yml))

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "integer_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := IntegerFormat{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema of type `integer` must specify a format of `int32` or `int64`", res[0].Message)
	assert.Equal(t, "$.components.schemas['thing']", res[0].Path)
}

func TestFormatLimit_RunRule_Int32(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - integer
      format: int32
`

	document, _ := libopenapi.NewDocument([]byte(yml))

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "integer_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := IntegerFormat{}
	ctx.Document = document
	ctx.DrDocument = drDocument

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestFormatLimit_RunRule_Int64(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - integer
      format: int64
`

	document, _ := libopenapi.NewDocument([]byte(yml))

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "integer_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := IntegerFormat{}
	ctx.Document = document
	ctx.DrDocument = drDocument

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
