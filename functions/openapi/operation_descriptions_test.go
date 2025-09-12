package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"testing"
)

func TestOperationDescription_GetSchema(t *testing.T) {
	def := OperationDescription{}
	assert.Equal(t, "oasDescriptions", def.GetSchema().Name)
}

func TestOperationDescription_RunRule(t *testing.T) {
	def := OperationDescription{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperationDescription_CheckDescriptionMissing(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /fish/paste:
    get:
      operationId: a
    put:
      operationId: b
    post:
      description: this is a description that is great and 10 words long at least
      operationId: c`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)
	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "operation method `GET` at path `/fish/paste` is missing a description or summary", res[0].Message)
	assert.Equal(t, "operation method `PUT` at path `/fish/paste` is missing a description or summary", res[1].Message)
}

func TestOperationDescription_CheckDescriptionTooShort(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /fish/paste:
    post:
      description: this is a thing that does nothing`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)
	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "operation method `POST` at path `/fish/paste` has a `description` that must be at least `10` words long", res[0].Message)
}

func TestOperationDescription_SummaryButNoDescription(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /fish/paste:
    post:
      description: something boring but valid
      summary: this is a thing that does nothing`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)
	opts := make(map[string]string)
	opts["minWords"] = "1"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}

func TestOperationDescription_CheckRequestBodyDescriptionExists(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /fish/paste:
    post:
      description: this is a thing
      requestBody:
        content:
          application/json:
            schema: 
              type: string`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)
	opts := make(map[string]string)
	opts["minWords"] = "1"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "operation method `POST` `requestBody` at path `/fish/paste` is missing a `description`", res[0].Message)

}

func TestOperationDescription_CheckRequestBodyDescriptionMeetsLength(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /fish/paste:
    post:
      description: this is a thing yeah
      requestBody:
        description: This is another thing
        content:
          application/json:
            schema: 
              type: string`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)
	opts := make(map[string]string)
	opts["minWords"] = "5"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "operation method `POST` `requestBody` at path `/fish/paste` has a `description` that must be at least `5` words long", res[0].Message)

}

func TestOperationDescription_CheckResponsesDescriptionExist(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /fish/paste:
    post:
      description: this is a thing
      summary: this is a summary
      requestBody:
        description: this is a thing
      responses:
        '200':
          content:
            application/json:
              schema:
                type: string`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)
	opts := make(map[string]string)
	opts["minWords"] = "1"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "operation method `POST` response code `200` `responseBody` at path `/fish/paste` is missing a `description`",
		res[0].Message)

}

func TestOperationDescription_CheckResponsesDescriptionLongEnough(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /fish/paste:
    post:
      description: this is a thing
      summary: this is a summary
      requestBody:
        description: this is a thing
      responses:
        '200':
          description: cake
          content:
            application/json:
              schema:
                type: string`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)
	opts := make(map[string]string)
	opts["minWords"] = "2"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.NotNil(t, res[0].Path)
	assert.Equal(t, "operation method `POST` response code `200` `responseBody` at path `/fish/paste` has a "+
		"`description` that must be at least `2` words long", res[0].Message)

}

func TestOperationDescription_CheckParametersIgnored(t *testing.T) {

	yml := `openapi: 3.0.1
paths:
  /fish/paste:
    parameters:
      - in: query
    post:
      description: this is a description that is great and 10 words long at least
      summary: hey hey hey
      operationId: c`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)
	opts := make(map[string]string)
	opts["minWords"] = "2"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestOperationDescription_CheckServersIgnored(t *testing.T) {
	yml := `openapi: 3.1.0
paths:
  /fish/paste:
    servers:
      - url: https://api.example.com/v1
    post:
      description: this is a description that is great and 10 words long at least
      summary: cakes
      operationId: c`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)
	opts := make(map[string]string)
	opts["minWords"] = "2"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "operation method `POST` at path `/fish/paste` has a `summary` that must be at least `2` words long", res[0].Message)
}

func TestOperationDescription_CheckExtensionsIgnored(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /fish/paste:
    x-parameters:
      - in: query
    post:
      summary: hey hey
      description: this is a description that is great and 10 words long at least
      operationId: c`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}
	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)
	opts := make(map[string]string)
	opts["minWords"] = "2"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationDescription{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}

func TestOperationDescription_CheckForNoPaths(t *testing.T) {

	yml := `openapi: 3.0.3`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	opts := make(map[string]string)
	opts["minWords"] = "10"

	rule := buildOpenApiTestRuleAction(path, "operation-description", "", opts)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OperationDescription{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}
