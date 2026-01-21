package openapi

import (
	"math/big"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
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

	specInfo, err := datamodel.ExtractSpecInfo([]byte(yml))

	// The malformed JSON should cause an error
	assert.Error(t, err)
	assert.Nil(t, specInfo)
	// Since we can't parse the malformed JSON, we can't run the schema validation
	// The test is confirming that malformed JSON is properly detected
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

func TestExtractLeafValidationErrors_NilError(t *testing.T) {
	results := extractLeafValidationErrors(nil)
	assert.Len(t, results, 0)
}

func TestExtractLeafValidationErrors_LeafError(t *testing.T) {
	err := &jsonschema.ValidationError{
		InstanceLocation: []string{"foo", "bar"},
		ErrorKind:        &kind.Required{Missing: []string{"name"}},
		Causes:           nil,
	}
	results := extractLeafValidationErrors(err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0], "/foo/bar")
	assert.Contains(t, results[0], "missing properties")
}

func TestExtractLeafValidationErrors_NestedErrors(t *testing.T) {
	leaf1 := &jsonschema.ValidationError{
		InstanceLocation: []string{"user", "name"},
		ErrorKind:        &kind.MinLength{Want: 1},
		Causes:           nil,
	}
	leaf2 := &jsonschema.ValidationError{
		InstanceLocation: []string{"user", "age"},
		ErrorKind:        &kind.Minimum{Want: big.NewRat(0, 1)},
		Causes:           nil,
	}
	parent := &jsonschema.ValidationError{
		InstanceLocation: []string{"user"},
		ErrorKind:        nil,
		Causes:           []*jsonschema.ValidationError{leaf1, leaf2},
	}
	results := extractLeafValidationErrors(parent)
	assert.Len(t, results, 2)
}

func TestExtractLeafValidationErrors_Deduplication(t *testing.T) {
	leaf1 := &jsonschema.ValidationError{
		InstanceLocation: []string{"foo"},
		ErrorKind:        &kind.Required{Missing: []string{"bar"}},
		Causes:           nil,
	}
	leaf2 := &jsonschema.ValidationError{
		InstanceLocation: []string{"foo"},
		ErrorKind:        &kind.Required{Missing: []string{"bar"}},
		Causes:           nil,
	}
	parent := &jsonschema.ValidationError{
		InstanceLocation: []string{},
		ErrorKind:        nil,
		Causes:           []*jsonschema.ValidationError{leaf1, leaf2},
	}
	results := extractLeafValidationErrors(parent)
	assert.Len(t, results, 1)
}

func TestErrorKindToString_NilKind(t *testing.T) {
	result := errorKindToString(nil)
	assert.Equal(t, "", result)
}

func TestErrorKindToString_Required(t *testing.T) {
	result := errorKindToString(&kind.Required{Missing: []string{"name", "age"}})
	assert.Contains(t, result, "missing properties")
	assert.Contains(t, result, "name")
}

func TestErrorKindToString_AdditionalProperties(t *testing.T) {
	result := errorKindToString(&kind.AdditionalProperties{Properties: []string{"extra"}})
	assert.Contains(t, result, "additional properties not allowed")
}

func TestErrorKindToString_Type(t *testing.T) {
	result := errorKindToString(&kind.Type{Want: []string{"string"}, Got: "integer"})
	assert.Contains(t, result, "expected type")
	assert.Contains(t, result, "string")
	assert.Contains(t, result, "integer")
}

func TestErrorKindToString_Enum(t *testing.T) {
	result := errorKindToString(&kind.Enum{Want: []any{"a", "b", "c"}})
	assert.Contains(t, result, "value must be one of")
}

func TestErrorKindToString_FalseSchema(t *testing.T) {
	result := errorKindToString(&kind.FalseSchema{})
	assert.Equal(t, "property not allowed", result)
}

func TestErrorKindToString_Pattern(t *testing.T) {
	result := errorKindToString(&kind.Pattern{Want: "^[a-z]+$"})
	assert.Contains(t, result, "does not match pattern")
	assert.Contains(t, result, "^[a-z]+$")
}

func TestErrorKindToString_MinLength(t *testing.T) {
	result := errorKindToString(&kind.MinLength{Want: 5})
	assert.Contains(t, result, "length must be >=")
	assert.Contains(t, result, "5")
}

func TestErrorKindToString_MaxLength(t *testing.T) {
	result := errorKindToString(&kind.MaxLength{Want: 10})
	assert.Contains(t, result, "length must be <=")
	assert.Contains(t, result, "10")
}

func TestErrorKindToString_Minimum(t *testing.T) {
	result := errorKindToString(&kind.Minimum{Want: big.NewRat(0, 1)})
	assert.Contains(t, result, "must be >=")
}

func TestErrorKindToString_Maximum(t *testing.T) {
	result := errorKindToString(&kind.Maximum{Want: big.NewRat(100, 1)})
	assert.Contains(t, result, "must be <=")
	assert.Contains(t, result, "100")
}

func TestErrorKindToString_MinItems(t *testing.T) {
	result := errorKindToString(&kind.MinItems{Want: 1})
	assert.Contains(t, result, "must have >=")
	assert.Contains(t, result, "items")
}

func TestErrorKindToString_MaxItems(t *testing.T) {
	result := errorKindToString(&kind.MaxItems{Want: 5})
	assert.Contains(t, result, "must have <=")
	assert.Contains(t, result, "items")
}

func TestErrorKindToString_MinProperties(t *testing.T) {
	result := errorKindToString(&kind.MinProperties{Want: 1})
	assert.Contains(t, result, "must have >=")
	assert.Contains(t, result, "properties")
}

func TestErrorKindToString_MaxProperties(t *testing.T) {
	result := errorKindToString(&kind.MaxProperties{Want: 10})
	assert.Contains(t, result, "must have <=")
	assert.Contains(t, result, "properties")
}

func TestErrorKindToString_Const(t *testing.T) {
	result := errorKindToString(&kind.Const{Want: "fixed"})
	assert.Contains(t, result, "must be")
	assert.Contains(t, result, "fixed")
}

func TestErrorKindToString_Format(t *testing.T) {
	result := errorKindToString(&kind.Format{Want: "email"})
	assert.Contains(t, result, "invalid format")
	assert.Contains(t, result, "email")
}
