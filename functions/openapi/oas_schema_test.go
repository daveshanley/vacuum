package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestOAS2Schema_GetSchema(t *testing.T) {
	def := OASSchema{}
	assert.Equal(t, "oasSchema", def.GetSchema().Name)
}

func TestOAS2Schema_RunRule(t *testing.T) {
	def := OASSchema{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestOAS2Schema_RunRule_Fail(t *testing.T) {

	yml := `swagger: 2.0
wiff: waff`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 3)
}

func TestOAS2Schema_RunRule_JSONSource_Fail_SpecBorked(t *testing.T) {

	yml := `{"swagger":"2.0", hello":"there"}`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 1)
}

func TestOAS2Schema_RunRule_JSONSource_Fail(t *testing.T) {

	yml := `{"swagger":"2.0", "hello":"there"}`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 2)
}

func TestOAS2Schema_RunRule_JSONSource_Fail_Unknown(t *testing.T) {

	yml := `{"swimmer":"2.0", "hello":"there"}`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 0)
}

func TestOAS2Schema_RunRule_AlmostPass(t *testing.T) {

	yml := `swagger: 2.0
info:
  contact:
    name: Hi
    url: https://quobix.com/vacuum
  license:
    name: MIT
  termsOfService: https://quobix.com/vacuum
  title: Quobix 
  version: "1.0"
paths:
  /hi:
    get:
      responses:
        "200":
          description: OK`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 1)
}

func TestOAS3Schema_RunRule_Pass(t *testing.T) {

	yml := `openapi: "3.0.0"
info:
  contact:
    name: Hi
    url: https://quobix.com/vacuum
  license:
    name: MIT
  termsOfService: https://quobix.com/vacuum
  title: Quobix 
  version: "1.0"
paths:
  /hi:
    get:
      responses:
        "200":
          description: OK`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas3_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 0)
}

func TestOAS3Schema_RunRule_Fail(t *testing.T) {

	yml := `openapi: "3.0"`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas3_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 2)
}

func TestOAS2Schema_RunRule_Success(t *testing.T) {

	yml := `swagger: '2.0'
info:
  contact:
    name: Hi
    url: https://quobix.com/vacuum
  license:
    name: MIT
  termsOfService: https://quobix.com/vacuum
  title: Quobix 
  version: "1.0"
paths:
  /hi:
    get:
      responses:
        "200":
          description: OK`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 0)
}

// TestOASSchema_OpenAPI30_NullableValid demonstrates that nullable: true is valid at document level in OpenAPI 3.0
// See https://github.com/daveshanley/vacuum/issues/710
// See https://github.com/daveshanley/vacuum/issues/603
func TestOASSchema_OpenAPI30_NullableValid(t *testing.T) {
	yml := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
          nullable: true
paths:
  /users:
    get:
      responses:
        '200':
          description: OK`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	// should pass - nullable: true is valid in OpenAPI 3.0
	assert.Len(t, res, 0)
}

// Note: Document validation (ValidateOpenAPIDocument) already handles nullable correctly
// because it uses info.APISchema which is version-appropriate. The OpenAPI 3.1 schema
// naturally doesn't include the 'nullable' keyword, so no additional version logic needed.
// The version-aware fix was needed for examples validation, which we implemented.

// TestOASSchema_OpenAPI31_ProperNullable demonstrates proper nullable syntax at document level in OpenAPI 3.1
// See https://github.com/daveshanley/vacuum/issues/710
// See https://github.com/daveshanley/vacuum/issues/603
func TestOASSchema_OpenAPI31_ProperNullable(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: [string, "null"]
paths:
  /users:
    get:
      responses:
        '200':
          description: OK`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	// should pass - type: [string, "null"] is the correct OpenAPI 3.1 syntax
	assert.Len(t, res, 0)
}

// TestOASSchema_OpenAPI31_BadNullable demonstrates that document validation catches nullable in OpenAPI 3.1
// See https://github.com/daveshanley/vacuum/issues/710
// See https://github.com/daveshanley/vacuum/issues/603
func TestOASSchema_OpenAPI31_BadNullable(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
          nullable: true
paths:
  /users:
    get:
      responses:
        '200':
          description: OK`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "oas_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	// add doc to context
	ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

	// Add DrDocument which is needed for schema checking
	m, _ := ctx.Document.BuildV3Model()
	ctx.DrDocument = drModel.NewDrDocument(m)

	def := OASSchema{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "The `nullable` keyword is not supported in OpenAPI 3.1. Use `type: ['string', 'null']` instead.", res[0].Message)
}
