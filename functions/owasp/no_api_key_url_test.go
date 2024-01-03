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

func TestNoApiKeyUrl(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    magicHerbs:
      type: apiKey
      in: query
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "no_api_key_in_url", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := NoApiKeyInUrl{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "API keys must not be passed via URL parameters (`query`)", res[0].Message)
	assert.Equal(t, "$.components.securitySchemes['magicHerbs'].in", res[0].Path)
}

func TestNoApiKeyUrl_Path(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    magicHerbs:
      type: apiKey
      in: path
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "no_api_key_in_url", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := NoApiKeyInUrl{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "API keys must not be passed via URL parameters (`path`)", res[0].Message)
	assert.Equal(t, "$.components.securitySchemes['magicHerbs'].in", res[0].Path)
}
