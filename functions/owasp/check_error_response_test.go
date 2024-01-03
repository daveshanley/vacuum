// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckErrorResponse_GetSchema(t *testing.T) {
	def := DefineErrorDefinition{}
	assert.Equal(t, "define_error_definition", def.GetSchema().Name)
}

func TestCheckErrorResponse_RunRule(t *testing.T) {
	def := DefineErrorDefinition{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestCheckErrorResponse_ErrorDefinitionMissing(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "422":
          description: "classic validation fail"
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	opts := make(map[string]interface{})
	opts["code"] = "401"

	rule := buildOpenApiTestRuleAction(path, "check_error_response", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := CheckErrorResponse{}
	ctx.Document = document

	drDocument := drModel.NewDrDocument(m)

	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "missing response code `401` for `GET`", res[0].Message)
	assert.Equal(t, "$.paths['/'].get.responses", res[0].Path)
}

func TestCheckErrorResponse_ErrorDefinitionMissing_Pass(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "401":
          description: "classic validation fail"
          content:
            application/json:
              schema:
                type: string`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	opts := make(map[string]interface{})
	opts["code"] = "401"

	rule := buildOpenApiTestRuleAction(path, "check_error_response", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := CheckErrorResponse{}
	ctx.Document = document

	drDocument := drModel.NewDrDocument(m)

	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestCheckErrorResponse_ErrorDefinitionMissing_MissingSchema(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "401":
          description: "classic validation fail"`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	opts := make(map[string]interface{})
	opts["code"] = "401"

	rule := buildOpenApiTestRuleAction(path, "check_error_response", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := CheckErrorResponse{}
	ctx.Document = document
	drDocument := drModel.NewDrDocument(m)

	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "missing schema for `401` response on `GET`", res[0].Message)
	assert.Equal(t, "$.paths['/'].get.responses['401']", res[0].Path)
}
