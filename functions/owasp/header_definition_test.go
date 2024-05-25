package owasp

import (
	"fmt"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
)

func TestHeaderDefinition_GetSchema(t *testing.T) {
	def := HeaderDefinition{}
	assert.Equal(t, "owaspHeaderDefinition", def.GetSchema().Name)
}

func TestHeaderDefinition_RunRule(t *testing.T) {
	def := HeaderDefinition{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestHeaderDefinition_HeaderDefinitionMissing(t *testing.T) {

	yml := `openapi: 3.1'
paths:
  /pizza/:
    get:
      responses:
        400:
          error
        200:
          error
        299:
          error
        499:
          headers:
            "Accept":
              error
        461:
          headers:
            "Content-Type":
              schema:
                type: string
        450:
          headers:
            "Accept":
              schema:
                type: string
            "Cache-Control":
              schema:
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

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "header_definition", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), map[string]interface{}{
		"headers": []string{"Accept||Cache-Control", "Content-Type"},
	})
	drDocument := drModel.NewDrDocument(m)
	ctx.DrDocument = drDocument
	def := HeaderDefinition{}
	ctx.Rule = &rule

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 4)

}

func TestHeaderDefinition_RateLimit(t *testing.T) {

	yml := `openapi: 3.1'
paths:
  /pizza/:
    get:
      responses:
        499:
          headers:
            "Rate-Limit":
              schema:
                type: string
        461:
          headers:
            "Content-Type":
              schema:
                type: string
        450:
          headers:
            "Accept":
              schema:
                type: string
            "Cache-Control":
              schema:
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

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "header_definition", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), map[string]interface{}{
		"headers": []string{"Accept||Cache-Control", "Content-Type", "Rate-Limit"},
	})
	drDocument := drModel.NewDrDocument(m)
	ctx.DrDocument = drDocument
	def := HeaderDefinition{}
	ctx.Rule = &rule

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}
