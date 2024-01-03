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

func TestStringLimit_RunRule(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    thing:
      type:
        - string
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "array_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := StringLimit{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema of type `string` must specify `maxLength`, `const` or `enum`", res[0].Message)
	assert.Equal(t, "$.components.schemas['thing']", res[0].Path)
}

func TestStringLimit_RunRule_Valid(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    thing:
      type:
        - string
      maxLength: 10
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "array_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := StringLimit{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestStringLimit_RunRule_ValidConst(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  schemas:
    thing:
      type:
        - string
      const: "hello"
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "array_limit", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := StringLimit{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
