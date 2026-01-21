// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
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

func TestAdditionalPropertiesConstrained_RunRule(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - object
      additionalProperties:
         type: object
         properties:
           burgers:
             type: string
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "no_additional_properties", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := AdditionalPropertiesConstrained{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema should also define `maxProperties` when `additionalProperties` is an object", res[0].Message)
	assert.Equal(t, "$.components.schemas['thing']", res[0].Path)
}

func TestAdditionalPropertiesConstrained_RunRule_Pass(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - object
      maxProperties: 10
      additionalProperties:
        type: object
        properties:
          burgers:
            type: string
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "no_additional_properties", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := AdditionalPropertiesConstrained{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestAdditionalPropertiesConstrained_RunRule_Pass_Bool(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  schemas:
    thing:
      type:
        - object
      maxProperties: 10
      additionalProperties: true
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "no_additional_properties", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := AdditionalPropertiesConstrained{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
