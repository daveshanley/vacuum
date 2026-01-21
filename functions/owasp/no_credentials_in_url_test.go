// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

func TestNoCredentialsUrl(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  parameters:
    spicyChicken:
      name: token
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

	rule := buildOpenApiTestRuleAction(path, "no_credentials_in_url", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := NoCredentialsInUrl{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	comp, _ := regexp.Compile(`(?i)^.*(client_?secret|token|access_?token|refresh_?token|id_?token|password|secret|api-?key).*$`)
	rule.PrecompiledPattern = comp
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "URL parameters must not contain credentials, passwords, or secrets (`token`)", res[0].Message)
	assert.Equal(t, "$.components.parameters['spicyChicken'].name", res[0].Path)
}

func TestNoCredentialsUrl_Alt(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  parameters:
    spicyChicken:
      name: api-key
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

	rule := buildOpenApiTestRuleAction(path, "no_credentials_in_url", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	drDocument := drModel.NewDrDocument(m)

	def := NoCredentialsInUrl{}
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	comp, _ := regexp.Compile(`(?i)^.*(client_?secret|token|access_?token|refresh_?token|id_?token|password|secret|api-?key).*$`)
	rule.PrecompiledPattern = comp
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "URL parameters must not contain credentials, passwords, or secrets (`api-key`)", res[0].Message)
	assert.Equal(t, "$.components.parameters['spicyChicken'].name", res[0].Path)
}
