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

func TestDefineErrorDefinition_GetSchema(t *testing.T) {
	def := DefineErrorDefinition{}
	assert.Equal(t, "define_error_definition", def.GetSchema().Name)
}

func TestDefineErrorDefinition_RunRule(t *testing.T) {
	def := DefineErrorDefinition{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestDefineErrorDefinition_ErrorDefinitionMissing(t *testing.T) {

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

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]interface{})
	opts["codes"] = []interface{}{"400", "4XX"}

	rule := buildOpenApiTestRuleAction(path, "define_error_definition", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	drDocument := drModel.NewDrDocument(m)

	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := DefineErrorDefinition{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "missing one of `400`, `4XX` response codes", res[0].Message)
	assert.Equal(t, "$.paths['/'].get.responses", res[0].Path)
}
