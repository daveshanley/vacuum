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

func TestNoNumericIds_RunRule(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /hi:
    parameters:
      - name: id
        schema: 
          type: integer
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "no_numeric_ids", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := NoNumericIds{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "don't use numeric IDs, use random IDs that cannot be guessed like UUIDs", res[0].Message)
	assert.Equal(t, "$.paths['/hi'].parameters[0].schema.type", res[0].Path)
}

func TestNoNumericIds_RunRule_Component(t *testing.T) {

	yml := `openapi: "3.1.0"
components:
  parameters:
    chip:
      name: id
      schema:
        type: integer
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "no_numeric_ids", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := NoNumericIds{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "don't use numeric IDs, use random IDs that cannot be guessed like UUIDs", res[0].Message)
	assert.Equal(t, "$.components.parameters['chip'].schema.type", res[0].Path)
}

func TestNoNumericIds_RunRule_Op(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /hi:
    post:
      parameters:
        - name: id
          schema: 
            type: integer
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "no_numeric_ids", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := NoNumericIds{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "don't use numeric IDs, use random IDs that cannot be guessed like UUIDs", res[0].Message)
	assert.Equal(t, "$.paths['/hi'].post.parameters[0].schema.type", res[0].Path)
}

func TestNoNumericIds_RunRule_Op_Alt(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /hi:
    post:
      parameters:
        - name: cake-id
          schema: 
            type: integer
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "no_numeric_ids", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := NoNumericIds{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "don't use numeric IDs, use random IDs that cannot be guessed like UUIDs", res[0].Message)
	assert.Equal(t, "$.paths['/hi'].post.parameters[0].schema.type", res[0].Path)
}
