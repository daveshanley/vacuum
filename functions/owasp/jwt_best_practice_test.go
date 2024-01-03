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

func TestJWTBestPractice_RunRule(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    magicHerbs:
      type: oauth2
      description: "This is a description"
`

	// create a new document from specification bytes
	document, err := libopenapi.NewDocument([]byte(yml))
	// if anything went wrong, an error is thrown
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "jwt_best_practice", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := JWTBestPractice{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "JWTs must explicitly declare support for `RFC8725` in the description", res[0].Message)
}

func TestJWTBestPractice_RunRule_Valid(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    magicHerbs:
      type: oauth2
      description: "This is a description RFC8725"
`

	document, err := libopenapi.NewDocument([]byte(yml))

	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "jwt_best_practice", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := JWTBestPractice{}
	ctx.Document = document
	ctx.DrDocument = drDocument

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestJWTBestPractice_RunRule_ValidJWT(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    magicHerbs:
      bearerFormat: jwt
      description: "This is a description RFC8725"
`

	document, err := libopenapi.NewDocument([]byte(yml))

	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	rule := buildOpenApiTestRuleAction(path, "jwt_best_practice", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := JWTBestPractice{}
	ctx.Document = document
	ctx.DrDocument = drDocument

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
