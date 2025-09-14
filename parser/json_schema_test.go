package parser

import (
	"sync"
	"testing"

	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// test we can generate a schema from a simple object
func TestConvertNode_Simple(t *testing.T) {
	yml := `components:
  schemas:
    Citrus:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        savory:
          $ref: '#/components/schemas/Savory'  
    Savory:
      type: object
      properties:
        tasteIndex:
          type: integer
        butter:
          type: boolean`

	var node yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &node)
	assert.NoError(t, mErr)

	config := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&node, config)

	resolver := index.NewResolver(idx)
	resolver.Resolve()

	p, _ := jsonpath.NewPath("$.components.schemas.Citrus")
	r := p.Query(&node)

	schema, err := ConvertNodeIntoJSONSchema(r[0], idx)
	assert.NoError(t, err)
	assert.NotNil(t, schema)
	assert.Equal(t, 3, orderedmap.Len(schema.Properties))

	// now check the schema is valid
	res, e := ValidateNodeAgainstSchema(nil, schema, r[0], false)
	assert.Nil(t, e)
	assert.True(t, res)
}


// TestConcurrentSchemaValidation tests for issue #512 - non-deterministic validation
func TestConcurrentSchemaValidation(t *testing.T) {
	// Schema with simple regex patterns that work with Go's regexp package
	// The original issue had Perl-style lookahead assertions which aren't supported
	// This test ensures concurrent validation produces consistent results
	schemaYAML := `
type: object
properties:
  passCode:
    type: string
    pattern: '^[a-zA-Z0-9]{6,100}$'
  phoneVerification:
    type: string
    pattern: '^[0-9]{10,15}$'
  name:
    type: string
    pattern: '^[a-zA-Z\s]{2,50}$'
`

	// Valid data that should pass validation
	validData := `
passCode: "abc123"
phoneVerification: "1234567890"
name: "John Doe"
`

	// Invalid data that should fail validation
	invalidData := `
passCode: "12345"  # missing letters
phoneVerification: "123"  # too short
name: "John123"  # contains numbers
`

	testCases := []struct {
		name       string
		data       string
		shouldPass bool
	}{
		{"valid data", validData, true},
		{"invalid data", invalidData, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run validation multiple times concurrently
			const iterations = 100
			const concurrency = 10

			results := make([]bool, iterations)
			errors := make([]error, iterations)
			var wg sync.WaitGroup

			for i := 0; i < iterations; i++ {
				idx := i
				wg.Add(1)

				// Use goroutines to simulate concurrent validation
				go func() {
					defer wg.Done()

					// Create a fresh schema for each iteration
					schema, err := ConvertYAMLIntoJSONSchema(schemaYAML, nil)
					if err != nil {
						errors[idx] = err
						return
					}

					// Parse the data
					var dataNode yaml.Node
					err = yaml.Unmarshal([]byte(tc.data), &dataNode)
					if err != nil {
						errors[idx] = err
						return
					}

					// Validate
					result, _ := ValidateNodeAgainstSchema(nil, schema, dataNode.Content[0], false)
					results[idx] = result
				}()

				// Add some concurrency by not waiting immediately
				if (i+1)%concurrency == 0 {
					wg.Wait()
				}
			}

			wg.Wait()

			// Check for any errors during setup
			for i, err := range errors {
				assert.NoError(t, err, "Iteration %d had an error", i)
			}

			// All results should be consistent
			firstResult := results[0]
			for i, result := range results {
				assert.Equal(t, tc.shouldPass, result,
					"Iteration %d: expected validation to be %v but got %v",
					i, tc.shouldPass, result)
				assert.Equal(t, firstResult, result,
					"Iteration %d: result inconsistent with first iteration", i)
			}
		})
	}
}

// TestIssue512_NonDeterministicValidation specifically tests the scenarios from issue #512
// https://github.com/daveshanley/vacuum/issues/512
func TestIssue512_NonDeterministicValidation(t *testing.T) {
	// Test case 1: Invalid type that caused "schema invalid: missing properties: '$ref'" error
	t.Run("invalid_type_consistency", func(t *testing.T) {
		schemaYAML := `
type: obejct  # Intentional typo
properties:
  phone_verification:
    type: super_string  # Invalid type
`
		// Run validation multiple times
		const iterations = 50
		errors := make([]error, iterations)

		var wg sync.WaitGroup
		for i := 0; i < iterations; i++ {
			idx := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := ConvertYAMLIntoJSONSchema(schemaYAML, nil)
				errors[idx] = err
			}()
		}
		wg.Wait()

		// All errors should be consistent (either all nil or all non-nil)
		firstErr := errors[0]
		for i, err := range errors {
			if (firstErr == nil && err != nil) || (firstErr != nil && err == nil) {
				t.Errorf("Inconsistent error at iteration %d", i)
			}
		}
	})

	// Test case 2: Complex but valid schema validation
	t.Run("complex_schema_consistency", func(t *testing.T) {
		// Schema similar to what might cause issues with references
		schemaYAML := `
type: object
required:
  - id
  - name
properties:
  id:
    type: string
    minLength: 5
    maxLength: 50
  name:
    type: string
    pattern: "^[a-zA-Z0-9_-]+$"
  tags:
    type: array
    items:
      type: string
    minItems: 1
    maxItems: 10
  metadata:
    type: object
    additionalProperties:
      type: string
`

		validData := `
id: "12345"
name: "test-name"
tags: ["tag1", "tag2"]
metadata:
  key1: "value1"
  key2: "value2"
`

		// Create schema once
		schema, err := ConvertYAMLIntoJSONSchema(schemaYAML, nil)
		assert.NoError(t, err)

		// Parse data once
		var dataNode yaml.Node
		err = yaml.Unmarshal([]byte(validData), &dataNode)
		assert.NoError(t, err)

		// Run concurrent validations
		const iterations = 100
		results := make([]bool, iterations)
		validationErrors := make([]int, iterations)

		var wg sync.WaitGroup
		for i := 0; i < iterations; i++ {
			idx := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				result, errs := ValidateNodeAgainstSchema(nil, schema, dataNode.Content[0], false)
				results[idx] = result
				validationErrors[idx] = len(errs)
			}()
		}
		wg.Wait()

		// All results should be consistent
		firstResult := results[0]
		firstErrorCount := validationErrors[0]
		for i := range results {
			assert.Equal(t, firstResult, results[i],
				"Result at iteration %d inconsistent with first result", i)
			assert.Equal(t, firstErrorCount, validationErrors[i],
				"Error count at iteration %d inconsistent with first count", i)
		}
	})

	// Test case 3: Schema with pattern that might cause issues
	t.Run("pattern_validation_consistency", func(t *testing.T) {
		// Note: Go's regexp doesn't support Perl-style lookahead assertions
		// This test uses a simpler pattern that works with Go's regexp
		schemaYAML := `
type: object
properties:
  code:
    type: string
    pattern: "^[A-Z]{2}[0-9]{4}$"
`

		testCases := []struct {
			name  string
			data  string
			valid bool
		}{
			{"valid", `code: "AB1234"`, true},
			{"invalid", `code: "ab1234"`, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				schema, err := ConvertYAMLIntoJSONSchema(schemaYAML, nil)
				assert.NoError(t, err)

				var dataNode yaml.Node
				err = yaml.Unmarshal([]byte(tc.data), &dataNode)
				assert.NoError(t, err)

				// Run multiple validations concurrently
				const iterations = 50
				results := make([]bool, iterations)

				var wg sync.WaitGroup
				for i := 0; i < iterations; i++ {
					idx := i
					wg.Add(1)
					go func() {
						defer wg.Done()
						result, _ := ValidateNodeAgainstSchema(nil, schema, dataNode.Content[0], false)
						results[idx] = result
					}()
				}
				wg.Wait()

				// All results should match expected
				for i, result := range results {
					assert.Equal(t, tc.valid, result,
						"Iteration %d: expected %v but got %v", i, tc.valid, result)
				}
			})
		}
	})
}

// TestIssue512_SharedSchemaReferences tests concurrent validation with shared schema references
func TestIssue512_SharedSchemaReferences(t *testing.T) {
	// Create a complex schema with internal references
	schemaYAML := `
type: object
properties:
  users:
    type: array
    items:
      type: object
      properties:
        id:
          type: string
        email:
          type: string
          format: email
        profile:
          type: object
          properties:
            age:
              type: integer
              minimum: 0
              maximum: 150
`

	validData := `
users:
  - id: "user1"
    email: "user1@example.com"
    profile:
      age: 25
  - id: "user2"
    email: "user2@example.com"
    profile:
      age: 30
`

	// Create schema and index
	var schemaNode yaml.Node
	err := yaml.Unmarshal([]byte(schemaYAML), &schemaNode)
	assert.NoError(t, err)

	schema, err := ConvertNodeIntoJSONSchema(schemaNode.Content[0], index.NewSpecIndexWithConfig(&schemaNode, index.CreateClosedAPIIndexConfig()))
	assert.NoError(t, err)

	var dataNode yaml.Node
	err = yaml.Unmarshal([]byte(validData), &dataNode)
	assert.NoError(t, err)

	// Run highly concurrent validations
	const iterations = 200

	results := make([]bool, iterations)
	var wg sync.WaitGroup

	// Create goroutines that will all try to validate at the same time
	start := make(chan struct{})
	for i := 0; i < iterations; i++ {
		idx := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // Wait for signal to start
			result, _ := ValidateNodeAgainstSchema(nil, schema, dataNode.Content[0], false)
			results[idx] = result
		}()
	}

	// Start all goroutines at once to maximize concurrency
	close(start)
	wg.Wait()

	// All results should be true (valid)
	for i, result := range results {
		assert.True(t, result, "Iteration %d: validation should have passed", i)
	}
}
