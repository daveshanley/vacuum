// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
)

func TestMissingType_PropertyWithoutType(t *testing.T) {
	// Test case from issue #517
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: string
        name:
          description: this is the name of the pet`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "missing-type",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	mt := MissingType{}
	res := mt.RunRule(nil, ctx)

	// We expect 2 results: one for the property check and one for the schema check
	// since DrDocument.Schemas contains ALL schemas including property schemas
	assert.Len(t, res, 2)
	
	// Find the property-specific message
	foundPropertyMessage := false
	foundSchemaMessage := false
	for _, r := range res {
		if contains(r.Message, "schema property") && contains(r.Message, "name") {
			foundPropertyMessage = true
			assert.Contains(t, r.Path, "properties['name']")
		} else if r.Message == "schema is missing a `type` field" {
			foundSchemaMessage = true
			assert.Contains(t, r.Path, "properties['name']")
		}
	}
	assert.True(t, foundPropertyMessage, "Should have property-specific message")
	assert.True(t, foundSchemaMessage, "Should have schema message")
}

func TestMissingType_SchemaWithoutType(t *testing.T) {
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    MissingType:
      description: A schema with no type defined
      minLength: 5`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "missing-type",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	mt := MissingType{}
	res := mt.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "missing a `type` field")
	assert.Contains(t, res[0].Path, "schemas['MissingType']")
}

func TestMissingType_MultiplePropertiesWithoutType(t *testing.T) {
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: string
        name:
          description: pet name
        age:
          description: pet age
        color:
          type: string
        weight:
          minimum: 0`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "missing-type",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	mt := MissingType{}
	res := mt.RunRule(nil, ctx)

	// Should detect 6 results total: 3 property checks + 3 schema checks
	// (since each property schema also appears in DrDocument.Schemas)
	assert.Len(t, res, 6)
	
	// Check that the correct properties were flagged
	messages := []string{}
	for _, r := range res {
		messages = append(messages, r.Message)
	}
	
	assert.Contains(t, messages[0]+messages[1]+messages[2], "name")
	assert.Contains(t, messages[0]+messages[1]+messages[2], "age")
	assert.Contains(t, messages[0]+messages[1]+messages[2], "weight")
}

func TestMissingType_NestedProperties(t *testing.T) {
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    Person:
      type: object
      properties:
        name:
          type: string
        address:
          type: object
          properties:
            street:
              type: string
            city:
              description: City name
            zipCode:
              pattern: "^[0-9]{5}$"`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "missing-type",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	mt := MissingType{}
	res := mt.RunRule(nil, ctx)

	// Should detect 4 results: 2 property checks + 2 schema checks for city and zipCode
	assert.Len(t, res, 4)
	
	foundCity := false
	foundZip := false
	for _, r := range res {
		if contains(r.Message, "city") {
			foundCity = true
		}
		if contains(r.Message, "zipCode") {
			foundZip = true
		}
	}
	
	assert.True(t, foundCity, "Should detect city property missing type")
	assert.True(t, foundZip, "Should detect zipCode property missing type")
}

func TestMissingType_SkipPolymorphicSchemas(t *testing.T) {
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Dog'
    Animal:
      allOf:
        - type: object
          properties:
            name:
              type: string
    Cat:
      type: object
    Dog:
      type: object`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "missing-type",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	mt := MissingType{}
	res := mt.RunRule(nil, ctx)

	// Should not report errors for polymorphic schemas
	assert.Empty(t, res)
}

func TestMissingType_SkipEnumAndConst(t *testing.T) {
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    Status:
      enum:
        - active
        - inactive
        - pending
    ConstValue:
      const: "fixed-value"
    ValidObject:
      type: object
      properties:
        status:
          enum:
            - on
            - off
        fixed:
          const: 42`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "missing-type",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	mt := MissingType{}
	res := mt.RunRule(nil, ctx)

	// Should not report errors for enum/const schemas as type can be inferred
	assert.Empty(t, res)
}

func TestMissingType_ImpliedObjectType(t *testing.T) {
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    ImpliedObject:
      properties:
        field1:
          type: string
        field2:
          type: integer
    ImpliedArray:
      items:
        type: string
    HasAdditionalProperties:
      additionalProperties:
        type: boolean`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "missing-type",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	mt := MissingType{}
	res := mt.RunRule(nil, ctx)

	// Should not report errors for schemas with properties/items/additionalProperties
	// as these imply a type
	assert.Empty(t, res)
}

func TestMissingType_PathsPropertySet(t *testing.T) {
	// Test that the Paths property is always set in results
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    MissingTypeSchema:
      description: Schema without type
    ObjectWithMissingTypes:
      type: object
      properties:
        field1:
          description: Missing type`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "missing-type",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	mt := MissingType{}
	res := mt.RunRule(nil, ctx)

	// Should have 3 results: one for MissingTypeSchema, one property check for field1, 
	// and one schema check for field1 (since it's also in DrDocument.Schemas)
	assert.Len(t, res, 3)
	
	// Check that all results have the Paths property set
	for _, r := range res {
		assert.NotNil(t, r.Paths, "Paths property should be set")
		assert.NotEmpty(t, r.Paths, "Paths array should not be empty")
		assert.Greater(t, len(r.Paths), 0, "Paths array should have at least one element")
		
		// The main Path should be included in Paths
		found := false
		for _, p := range r.Paths {
			if p == r.Path {
				found = true
				break
			}
		}
		assert.True(t, found, "Main Path should be included in Paths array")
	}
}

func TestMissingType_AllPropertiesValid(t *testing.T) {
	yml := `openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    ValidSchema:
      type: object
      properties:
        stringProp:
          type: string
        numberProp:
          type: number
        integerProp:
          type: integer
        booleanProp:
          type: boolean
        arrayProp:
          type: array
          items:
            type: string
        objectProp:
          type: object
          properties:
            nested:
              type: string`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	drDocument := drModel.NewDrDocument(m)

	rule := model.Rule{
		Name: "missing-type",
	}
	ctx := model.RuleFunctionContext{
		Rule:       &rule,
		DrDocument: drDocument,
		Document:   document,
	}

	mt := MissingType{}
	res := mt.RunRule(nil, ctx)

	// Should not report any errors as all types are defined
	assert.Empty(t, res)
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		   (s == substr || len(s) > len(substr) && 
		   (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		   containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	if len(s) <= len(substr) {
		return false
	}
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}