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

func TestRatelimitRetry429_RunRule(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "429":
          headers:
            "Retry-After": "something"`

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

	rule := buildOpenApiTestRuleAction(path, "ratelimit_retry_429", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := RatelimitRetry429{}
	ctx.Document = document
	drDocument := drModel.NewDrDocument(m)
	
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestRatelimitRetry429_RunRule_Fail(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
paths:
  /:
    get:
      responses:
        "429":
          headers:
            "Yummy-Cakes": "something"`

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

	rule := buildOpenApiTestRuleAction(path, "ratelimit_retry_249", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	def := RatelimitRetry429{}
	ctx.Document = document
	drDocument := drModel.NewDrDocument(m)
	
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "missing 'Retry-After' header for 429 error response", res[0].Message)
	assert.Equal(t, "$.paths./.get.responses.429", res[0].Path)
}
