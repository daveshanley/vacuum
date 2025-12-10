// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
)

func TestUnnecessaryCombinator_GetSchema(t *testing.T) {
	def := UnnecessaryCombinator{}
	assert.Equal(t, "oasUnnecessaryCombinator", def.GetSchema().Name)
}

func TestUnnecessaryCombinator_GetCategory(t *testing.T) {
	def := UnnecessaryCombinator{}
	assert.Equal(t, model.FunctionCategoryOpenAPI, def.GetCategory())
}

func TestUnnecessaryCombinator_RunRule_NoSchemas(t *testing.T) {
	def := UnnecessaryCombinator{}
	ctx := model.RuleFunctionContext{
		DrDocument: nil,
	}

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0)
}

func TestUnnecessaryCombinator_RunRule_EmptyDocument(t *testing.T) {
	def := UnnecessaryCombinator{}
	ctx := model.RuleFunctionContext{
		DrDocument: &drModel.DrDocument{},
	}

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0)
}

func TestUnnecessaryCombinator_RunRule_SingleAllOf(t *testing.T) {
	def := UnnecessaryCombinator{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      allOf:
        - type: object
          properties:
            name:
              type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "allOf")
	assert.Contains(t, res[0].Message, "only one item")
}

func TestUnnecessaryCombinator_RunRule_SingleAnyOf(t *testing.T) {
	def := UnnecessaryCombinator{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Product:
      anyOf:
        - type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "anyOf")
	assert.Contains(t, res[0].Message, "only one item")
}

func TestUnnecessaryCombinator_RunRule_SingleOneOf(t *testing.T) {
	def := UnnecessaryCombinator{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Order:
      oneOf:
        - type: object
          properties:
            id:
              type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "oneOf")
	assert.Contains(t, res[0].Message, "only one item")
}

func TestUnnecessaryCombinator_RunRule_MultipleAllOfValid(t *testing.T) {
	def := UnnecessaryCombinator{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      allOf:
        - type: object
          properties:
            name:
              type: string
        - type: object
          properties:
            userId:
              type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0) // Should not trigger for multiple items
}

func TestUnnecessaryCombinator_RunRule_MultipleAnyOfValid(t *testing.T) {
	def := UnnecessaryCombinator{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Product:
      anyOf:
        - type: string
        - type: number
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0) // Should not trigger for multiple items
}

func TestUnnecessaryCombinator_RunRule_AllThreeCombinatorsWithSingleItems(t *testing.T) {
	def := UnnecessaryCombinator{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      allOf:
        - type: object
          properties:
            name:
              type: string
    Product:
      anyOf:
        - type: string
    Order:
      oneOf:
        - type: object
          properties:
            id:
              type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 3)

	// Check that all three combinators were found
	combinatorsFound := make(map[string]bool)
	for _, result := range res {
		if strings.Contains(result.Message, "allOf") {
			combinatorsFound["allOf"] = true
		}
		if strings.Contains(result.Message, "anyOf") {
			combinatorsFound["anyOf"] = true
		}
		if strings.Contains(result.Message, "oneOf") {
			combinatorsFound["oneOf"] = true
		}
	}

	assert.True(t, combinatorsFound["allOf"], "Should find allOf violation")
	assert.True(t, combinatorsFound["anyOf"], "Should find anyOf violation")
	assert.True(t, combinatorsFound["oneOf"], "Should find oneOf violation")
}

func TestUnnecessaryCombinator_RunRule_EmptyCombinators(t *testing.T) {
	def := UnnecessaryCombinator{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      allOf: []
    Product:
      anyOf: []
    Order:
      oneOf: []
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0) // Empty arrays should not trigger the rule
}

func TestUnnecessaryCombinator_RunRule_NoCombinatorsSchema(t *testing.T) {
	def := UnnecessaryCombinator{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        name:
          type: string
    Product:
      type: string
    BaseProduct:
      type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0) // Schemas without combinators should not trigger
}

// OAS 3.0.x: single allOf with $ref and sibling description is legitimate workaround
func TestUnnecessaryCombinator_RunRule_OAS30_AllOfWithRefAndDescription(t *testing.T) {
	def := UnnecessaryCombinator{}
	yml := `openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    BaseModel:
      type: object
      properties:
        id:
          type: string
    ExtendedModel:
      description: "Extended description that overrides the ref"
      allOf:
        - $ref: '#/components/schemas/BaseModel'
`
	ctx := buildTestContext(yml, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0, "OAS 3.0.x allOf with $ref and sibling description should not trigger")
}

// OAS 3.1: single allOf with $ref and sibling description should still trigger
func TestUnnecessaryCombinator_RunRule_OAS31_AllOfWithRefAndDescription(t *testing.T) {
	def := UnnecessaryCombinator{}
	yml := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    BaseModel:
      type: object
      properties:
        id:
          type: string
    ExtendedModel:
      description: "Extended description"
      allOf:
        - $ref: '#/components/schemas/BaseModel'
`
	ctx := buildTestContext(yml, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 1, "OAS 3.1 should trigger because $ref siblings are supported natively")
	assert.Contains(t, res[0].Message, "allOf")
}

func buildTestContext(yamlContent string, t *testing.T) model.RuleFunctionContext {
	document, err := libopenapi.NewDocument([]byte(yamlContent))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	drDoc := drModel.NewDrDocument(m)

	return model.RuleFunctionContext{
		DrDocument: drDoc,
		SpecInfo:   document.GetSpecInfo(),
		Rule:       &model.Rule{},
	}
}