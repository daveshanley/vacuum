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

func TestIntegerLimit_RunRule(t *testing.T) {

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

	drDocument := drModel.NewDrDocument(m.Index, m.Index.GetRolodex())
	drDocument.WalkV3(&m.Model)

	def := IntegerLimit{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema of type `string` must specify `minimum` and `maximum` or `exclusiveMinimum` "+
		"and `exclusiveMaximum`", res[0].Message)
	assert.Equal(t, "$.components.schemas['thing']", res[0].Path)
}

func TestIntegerLimit_RunRule_Min_Fail(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - integer
      minimum: 10
`
	document, _ := libopenapi.NewDocument([]byte(yml))
	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "integer_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m.Index, m.Index.GetRolodex())
	drDocument.WalkV3(&m.Model)

	def := IntegerLimit{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema of type `string` must specify `minimum` and `maximum` or `exclusiveMinimum` "+
		"and `exclusiveMaximum`", res[0].Message)
	assert.Equal(t, "$.components.schemas['thing']", res[0].Path)
}

func TestIntegerLimit_RunRule_MinMax(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - integer
      minimum: 10
      maximum: 20
`
	document, _ := libopenapi.NewDocument([]byte(yml))
	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "integer_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m.Index, m.Index.GetRolodex())
	drDocument.WalkV3(&m.Model)

	def := IntegerLimit{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0)
}

func TestIntegerLimit_RunRule_Max_Fail(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - integer
      maximum: 10
`
	document, _ := libopenapi.NewDocument([]byte(yml))
	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "integer_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m.Index, m.Index.GetRolodex())
	drDocument.WalkV3(&m.Model)

	def := IntegerLimit{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema of type `string` must specify `minimum` and `maximum` or `exclusiveMinimum` "+
		"and `exclusiveMaximum`", res[0].Message)
	assert.Equal(t, "$.components.schemas['thing']", res[0].Path)
}

func TestIntegerLimit_RunRule_ExlMin_Fail(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - integer
      minimum: 5
      exclusiveMinimum: 10
`
	document, _ := libopenapi.NewDocument([]byte(yml))
	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "integer_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m.Index, m.Index.GetRolodex())
	drDocument.WalkV3(&m.Model)

	def := IntegerLimit{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema of type `string` must specify `minimum` and `maximum` or `exclusiveMinimum` "+
		"and `exclusiveMaximum`", res[0].Message)
	assert.Equal(t, "$.components.schemas['thing']", res[0].Path)
}

func TestIntegerLimit_RunRule_ExlMax_Fail(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - integer
      minimum: 5
      exclusiveMaximum: 10
`
	document, _ := libopenapi.NewDocument([]byte(yml))
	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "integer_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m.Index, m.Index.GetRolodex())
	drDocument.WalkV3(&m.Model)

	def := IntegerLimit{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestIntegerLimit_RunRule_ExlMin_Pass(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - integer
      exclusiveMinimum: 10
      exclusiveMaximum: 20
`
	document, _ := libopenapi.NewDocument([]byte(yml))
	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "integer_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m.Index, m.Index.GetRolodex())
	drDocument.WalkV3(&m.Model)

	def := IntegerLimit{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
