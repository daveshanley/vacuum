package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperationTags_GetSchema(t *testing.T) {
	def := OperationTags{}
	assert.Equal(t, "oasOperationTags", def.GetSchema().Name)
}

func TestOperationTags_RunRule(t *testing.T) {
	def := OperationTags{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOperationTags_RunRule_Success(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /hello:
    post:
      tags:
       - a
       - b
    get:
      tags:
       - a
  /there/yeah:
    post:
      tags:
       - b`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "tag-defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationTags{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
	//assert.Equal(t, "the 'get' operation at path '/ember' contains a duplicate operationId 'littleSong'", res[0].Message)
}

func TestOperationTags_RunRule_NoTags(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /hello:
    post:
      description: hi
    get:
      tags:
       - a
  /there/yeah:
    post:
      tags:
       - b`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "tag-defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationTags{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "tags for `POST` operation are missing", res[0].Message)
	assert.Equal(t, "$.paths['/hello'].post", res[0].Path)

}

func TestOperationTags_RunRule_EmptyTags(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /hello:
    post:
      tags:
       - a
       - b
    get:
      tags:
  /there/yeah:
    post:
      tags:
       - b`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "tag-defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationTags{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "tags for `GET` operation are missing", res[0].Message)
	assert.Equal(t, "$.paths['/hello'].get", res[0].Path)

}

func TestOperationTags_RunRule_IgnoreParameters(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /hello:
    post:
      tags:
        - a
        - b
    parameters:
      - in: query`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "tag-defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationTags{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestOperationTags_RunRule_IgnoreServers(t *testing.T) {
	yml := `openapi: 3.1.0
paths:
  /hello:
    post:
      tags:
        - a
        - b
    servers:
      - url: https://api.example.com/v1`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "tag-defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationTags{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestOperationTags_RunRule_IgnoreExtensions(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /hello:
    post:
      tags:
        - a
        - b
    x-parameters:
      - in: query
    parameters:
      - in: path`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "tag-defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationTags{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestOperationTags_RunRule_HandleNoPaths(t *testing.T) {

	yml := `openapi: 3.0.3`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "tag-defined", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := OperationTags{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}
