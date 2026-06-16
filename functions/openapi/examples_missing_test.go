// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"go.yaml.in/yaml/v4"
	"testing"
)

func TestExamplesMissing_GetSchema(t *testing.T) {
	def := ExamplesMissing{}
	assert.Equal(t, "oasExampleMissing", def.GetSchema().Name)
}

func TestExamplesMissing_RunRule(t *testing.T) {
	def := ExamplesMissing{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestExamplesMissing(t *testing.T) {

	yml := `openapi: 3.1
paths:
  /pizza:
    get:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              description: I need an example`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "media type is missing `examples` or `example`", res[0].Message)
	assert.Contains(t, res[0].Path, "$.paths['/pizza'].get.requestBody.content['application/json']")
}

func TestExamples_ContentOK(t *testing.T) {

	yml := `openapi: 3.1
components:
  parameters:
    ParameterA:     
      name: paramA     
      required: false
      in: query
      description: “some random text” 
      content:
        application/json:
          schema:
            type: object
            additionalProperties:
              type: string
              example: hey
            example:
              “a”: “5
          examples:
            ParameterA:
              value:
                Key: “value”`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestExamplesMissing_TrainTravel(t *testing.T) {

	yml := `openapi: 3.1
paths:
  /trips:
    get:
      parameters:
        - name: origin
          in: query
          schema:
            type: string
            format: uuid
          example: efdbb9d1-02c2-4bc3-afb7-6788d8782b1ee`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestExamplesMissing_Alt(t *testing.T) {

	yml := `openapi: 3.1
components:
  schemas:
    Pizza:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "schema property `id` is missing `examples` or `example`", res[0].Message)
	assert.Contains(t, res[0].Path, "$.components.schemas['Pizza']")
}

func TestExamplesMissing_PropertySchemaWithoutValueDoesNotPanic(t *testing.T) {
	yml := `openapi: 3.1
info:
  title: Test API
  version: 1.0.0
paths: {}
`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	properties := orderedmap.New[string, *drV3.SchemaProxy]()
	properties.Set("broken", &drV3.SchemaProxy{
		Schema: &drV3.Schema{
			Foundation: drV3.Foundation{
				Key:         "broken",
				PathSegment: "broken",
			},
		},
	})
	rootSchema := &drV3.Schema{
		Value: &highbase.Schema{
			Type: []string{"object"},
			Example: &yaml.Node{
				Kind:   yaml.MappingNode,
				Tag:    "!!map",
				Line:   1,
				Column: 1,
			},
		},
		Properties: properties,
		Foundation: drV3.Foundation{
			Key:         "Root",
			PathSegment: "Root",
		},
	}
	drDocument := &drModel.DrDocument{Schemas: []*drV3.Schema{rootSchema}}

	rule := buildOpenApiTestRuleAction("$", "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	var res []model.RuleFunctionResult
	assert.NotPanics(t, func() {
		res = def.RunRule(nil, ctx)
	})
	assert.Empty(t, res)
}

func TestExamplesMissing_MediaTypeWithoutSchemaDoesNotPanic(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: media type no schema
  version: 1.0.0
paths:
  /plain:
    get:
      responses:
        '204':
          description: No Content
          content:
            application/json:
              example:
                ok: true`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocumentWithConfig(m, &drModel.DrConfig{
		UseSchemaCache:     true,
		DeterministicPaths: true,
	})

	rule := buildOpenApiTestRuleAction("$", "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	assert.NotPanics(t, func() {
		def.RunRule(nil, ctx)
	})
}

func TestExamplesMissing_MediaTypeSchemaCacheAliasesUseHydratedSchemaGuards(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{
			name: "schema_example",
			schema: `type: object
      example:
        id: abc`,
		},
		{
			name: "schema_default",
			schema: `type: object
      default:
        id: abc`,
		},
		{
			name:   "simple_string",
			schema: `type: string`,
		},
		{
			name: "array_item_example",
			schema: `type: array
      items:
        type: string
        example: abc`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yml := fmt.Sprintf(`openapi: 3.1.0
info:
  title: media type cache alias %s
  version: 1.0.0
paths:
  /first:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Aliased'
  /second:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Aliased'
components:
  schemas:
    Aliased:
      %s`, tt.name, tt.schema)

			document, err := libopenapi.NewDocument([]byte(yml))
			if err != nil {
				panic(fmt.Sprintf("cannot create new document: %e", err))
			}

			m, _ := document.BuildV3Model()
			drDocument := drModel.NewDrDocumentWithConfig(m, &drModel.DrConfig{
				UseSchemaCache:     true,
				DeterministicPaths: true,
			})

			rule := buildOpenApiTestRuleAction("$", "examples_missing", "", nil)
			ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
			ctx.Document = document
			ctx.DrDocument = drDocument
			ctx.Rule = &rule

			def := ExamplesMissing{}
			res := def.RunRule(nil, ctx)

			assert.Empty(t, res)
		})
	}
}

func TestExamplesMissing_Header(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /cake:
    get:
      responses:
        '200':
          headers:
            bingo:
              description: I need an example`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0) // no schema? no examples are valid.

}

func TestExamplesMissing_Header_Alt(t *testing.T) {
	yml := `openapi: 3.1
components:
  headers:
    Cake:
      description: I need an example`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0) // no schema? no examples are possible.

}

func TestExamplesMissing_MediaType(t *testing.T) {
	yml := `openapi: 3.1
paths:
  /herbs:
    get:
      requestBody:
        content:
          application/json:
            description: I need an example`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "media type is missing `examples` or `example`", res[0].Message)
	assert.Equal(t, "$.paths['/herbs'].get.requestBody.content['application/json']", res[0].Path)

}

func TestExamplesMissing_MediaType_Alt(t *testing.T) {
	yml := `openapi: 3.1
components:
  requestBodies:
    Herbs:
      content:
        slapsication/json:
          description: I need an example`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "media type is missing `examples` or `example`", res[0].Message)
	assert.Equal(t, "$.components.requestBodies['Herbs'].content['slapsication/json']", res[0].Path)

}

func TestExamplesMissing_MediaType_EmptyArray(t *testing.T) {
	yml := `openapi: 3.1
components:
  schemas:
    Herby:
      type: object
      examples:
        -
`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema is missing `examples` or `example`", res[0].Message)
	assert.Equal(t, "$.components.schemas['Herby']", res[0].Path)

}

func TestExamplesMissing_ConstAsImplicitExample(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Status:
      type: string
      const: "active"
`
	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0, "const should be treated as implicit example")
}

func TestExamplesMissing_DefaultAsImplicitExample(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    PageSize:
      type: integer
      default: 10
`
	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0, "default should be treated as implicit example")
}

func TestExamplesMissing_NestedPropertyWithConst(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Order:
      type: object
      properties:
        status:
          type: string
          const: "pending"
`
	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	// nested properties with const should not trigger missing example error
	for _, r := range res {
		assert.NotContains(t, r.Path, "status", "nested property with const should not be flagged")
	}
}

func TestExamplesMissing_ArrayItemWithDefault(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Tags:
      type: array
      items:
        type: string
        default: "general"
`
	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	// array items with default should not trigger missing example error for the items schema
	for _, r := range res {
		assert.NotContains(t, r.Path, "items", "array items with default should not be flagged")
	}
}
func TestExamplesMissing_XExtensibleEnum(t *testing.T) {
	yml := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - name: country_codes
          in: query
          schema:
            type: array
            items:
              $ref: '#/components/schemas/CountryCode'
      responses:
        '200':
          description: Success
components:
  schemas:
    CountryCode:
      type: string
      description: Country code
      x-extensible-enum:
        - US
        - CA
        - GB`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestExamplesMissing_DoubleRefWithXExtensibleEnum(t *testing.T) {
	yml := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      parameters:
        - name: codes
          in: query
          schema:
            type: array
            items:
              $ref: '#/components/schemas/RefToExtensible'
      responses:
        '200':
          description: Success
components:
  schemas:
    RefToExtensible:
      $ref: '#/components/schemas/StatusCode'
    StatusCode:
      type: string
      x-extensible-enum:
        - active
        - inactive`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction("$", "examples_missing", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := ExamplesMissing{}

	assert.NotPanics(t, func() {
		def.RunRule(nil, ctx)
	})
}
