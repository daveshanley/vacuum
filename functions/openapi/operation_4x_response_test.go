package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/assert"
	"os"
	"testing"
)

func TestOperation4xResponse_GetSchema(t *testing.T) {
	def := Operation4xResponse{}
	assert.Equal(t, "oasOpErrorResponse", def.GetSchema().Name)
}

func TestOperation4xResponse_RunRule(t *testing.T) {
	def := Operation4xResponse{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperation4xResponse_RunRule_Success(t *testing.T) {

	sampleYaml, _ := os.ReadFile("../../model/test_files/burgershop.openapi.yaml")

	document, err := libopenapi.NewDocument(sampleYaml)
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "responses", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := Operation4xResponse{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestOperation4xResponse_RunRule_ExitEarly(t *testing.T) {

	sampleYaml := []byte("openapi: 3.0.1\nhi: there")

	document, err := libopenapi.NewDocument(sampleYaml)
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "responses", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := Operation4xResponse{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestOperation4xResponse_RunRule_Fail(t *testing.T) {

	sampleYaml, _ := os.ReadFile("../../model/test_files/stripe.yaml")

	document, err := libopenapi.NewDocument(sampleYaml)
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "responses", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := Operation4xResponse{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 402)
}

func TestOperation4xResponse_RunRule_RangeCode(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /melody:
    get:
      responses:
        "200":
          description: hello
        "4XX":
          description: hello
    post:
      responses:
        "200":
          description: hello
        "5XX":
          description: hello`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	rule := buildOpenApiTestRuleAction("$", "responses", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drModel.NewDrDocument(m)
	ctx.Rule = &rule

	def := Operation4xResponse{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "$.paths['/melody'].post.responses", res[0].Path)
}
