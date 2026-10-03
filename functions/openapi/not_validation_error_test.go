// Copyright 2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"testing"

	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"github.com/pb33f/testify/assert"
)

func TestIsNotValidationError(t *testing.T) {
	assert.False(t, isNotValidationError(nil))

	assert.True(t, isNotValidationError(&errors.SchemaValidationFailure{
		Reason: "'not' failed",
	}))

	assert.True(t, isNotValidationError(&errors.SchemaValidationFailure{
		Reason: "something 'not' failed here",
	}))

	assert.False(t, isNotValidationError(&errors.SchemaValidationFailure{
		Reason: "missing properties: [id]",
	}))

	// Orig error with *kind.Not
	assert.True(t, isNotValidationError(&errors.SchemaValidationFailure{
		Reason: "custom reason",
		OriginalJsonSchemaError: &jsonschema.ValidationError{
			Causes: []*jsonschema.ValidationError{
				{ErrorKind: &kind.Not{}},
			},
		},
	}))
}

func TestFormatExampleValidationReason_NonNot(t *testing.T) {
	err := &errors.SchemaValidationFailure{
		Reason: "expected type string, got integer",
	}
	assert.Equal(t, "expected type string, got integer", formatExampleValidationReason(err))
	assert.Equal(t, "", formatExampleValidationReason(nil))
}

func TestFormatExampleValidationReason_RequiredSingle(t *testing.T) {
	refSchema := `
type: object
then:
  not:
    required:
      - reason
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		KeywordLocation: "/then",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "example violates `not`: property `reason` must not be present", msg)
}

func TestFormatExampleValidationReason_RequiredMultiple(t *testing.T) {
	refSchema := `
not:
  required:
    - first
    - second
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		KeywordLocation: "",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "example violates `not`: properties `first`, `second` must not be present", msg)
}

func TestFormatExampleValidationReason_TypeSingle(t *testing.T) {
	refSchema := `
not:
  type: string
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "example violates `not`: value must not be of type `string`", msg)
}

func TestFormatExampleValidationReason_TypeMultiple(t *testing.T) {
	refSchema := `
not:
  type:
    - string
    - integer
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "example violates `not`: value must not be of type `string` or `integer`", msg)
}

func TestFormatExampleValidationReason_Const(t *testing.T) {
	refSchema := `
not:
  const: admin
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "example violates `not`: value must not be `admin`", msg)
}

func TestFormatExampleValidationReason_Enum(t *testing.T) {
	refSchema := `
not:
  enum:
    - forbidden
    - blocked
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "example violates `not`: value must not be one of: `forbidden`, `blocked`", msg)
}

func TestFormatExampleValidationReason_Pattern(t *testing.T) {
	refSchema := `
not:
  pattern: '^[0-9]+$'
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "example violates `not`: value must not match pattern `^[0-9]+$`", msg)
}

func TestFormatExampleValidationReason_MinMax(t *testing.T) {
	refSchema := `
not:
  minimum: 10
  maximum: 20
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Contains(t, msg, "value must not be >= `10`")
	assert.Contains(t, msg, "value must not be <= `20`")
}

func TestFormatExampleValidationReason_Properties(t *testing.T) {
	refSchema := `
not:
  properties:
    disallowed:
      type: string
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "example violates `not`: property `disallowed` must not be of type `string`", msg)
}

func TestFormatExampleValidationReason_Ref(t *testing.T) {
	refSchema := `
not:
  $ref: '#/components/schemas/ForbiddenObject'
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "example violates `not`: matches schema at `#/components/schemas/ForbiddenObject`", msg)
}

func TestFormatExampleValidationReason_BooleanNot(t *testing.T) {
	refSchema := `
not: true
`
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		ReferenceSchema: refSchema,
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "example violates `not`: schema allows no instances (`not: true`)", msg)
}

func TestFormatExampleValidationReason_Fallback(t *testing.T) {
	err := &errors.SchemaValidationFailure{
		Reason:          "'not' failed",
		KeywordLocation: "/then",
		ReferenceSchema: "",
	}
	msg := formatExampleValidationReason(err)
	assert.Equal(t, "'not' failed", msg)
}
