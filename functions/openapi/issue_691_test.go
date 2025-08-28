package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
)

// TestSchemaType_Issue691_ComprehensiveFix tests the complete fix for issue #691
// This ensures that:
// 1. allOf with $ref doesn't falsely report "no properties" when properties exist
// 2. Invalid required fields ARE caught even when using allOf with $ref
func TestSchemaType_Issue691_ComprehensiveFix(t *testing.T) {
	yml := `openapi: 3.0.0
components:
  schemas:
    BaseSchema:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
    ExtendedSchema:
      type: object
      properties:
        extra:
          type: string
    CompleteSchema:
      type: object
      allOf:
        - $ref: '#/components/schemas/BaseSchema'
        - $ref: '#/components/schemas/ExtendedSchema'
      required:
        - id          # Valid - in BaseSchema
        - name        # Valid - in BaseSchema  
        - extra       # Valid - in ExtendedSchema
        - missing     # Invalid - not defined anywhere
    PartialSchema:
      type: object
      allOf:
        - $ref: '#/components/schemas/BaseSchema'
      properties:
        local:
          type: string
      required:
        - id          # Valid - in BaseSchema via allOf
        - local       # Valid - in direct properties
        - nonexistent # Invalid - not defined anywhere`

	document, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "schema-type-check", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := SchemaTypeCheck{}
	res := def.RunRule(nil, ctx)

	// Check results
	t.Logf("Total results: %d", len(res))
	for _, r := range res {
		t.Logf("Result: %s at %s", r.Message, r.Path)
	}

	// Should have exactly 2 errors: one for 'missing' and one for 'nonexistent'
	missingFieldErrors := 0
	foundMissingError := false
	foundNonexistentError := false

	for _, r := range res {
		if r.Message == "`required` field `missing` is not defined in `properties`" {
			foundMissingError = true
			missingFieldErrors++
		}
		if r.Message == "`required` field `nonexistent` is not defined in `properties`" {
			foundNonexistentError = true
			missingFieldErrors++
		}
	}

	assert.True(t, foundMissingError, "Should report error for 'missing' field in CompleteSchema")
	assert.True(t, foundNonexistentError, "Should report error for 'nonexistent' field in PartialSchema")
	assert.Equal(t, 2, missingFieldErrors, "Should have exactly 2 missing field errors")

	// Should NOT have any "object contains `required` fields but no `properties`" errors
	hasNoPropertiesError := false
	for _, r := range res {
		if r.Message == "object contains `required` fields but no `properties`" {
			hasNoPropertiesError = true
			t.Errorf("Unexpected error: %s at %s", r.Message, r.Path)
		}
	}
	assert.False(t, hasNoPropertiesError,
		"Should NOT report 'no properties' error when properties exist via allOf")
}
