package model

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// TestAutoFixFunction_EmptyDescription tests a simple auto-fix that converts empty descriptions to a default value
func TestAutoFixFunction_EmptyDescription(t *testing.T) {
	// Sample YAML with empty description
	sampleYaml := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  description: ""
paths:
  /test:
    get:
      summary: Test endpoint
      description: ""
      responses:
        '200':
          description: ""
`

	// Parse the document
	var document yaml.Node
	err := yaml.Unmarshal([]byte(sampleYaml), &document)
	assert.NoError(t, err)

	// Find nodes with empty descriptions
	path := "$..description"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	
	// Should find 3 empty descriptions
	assert.Len(t, nodes, 3)

	// Define our auto-fix function
	autoFixEmptyDescription := func(node *yaml.Node, document *yaml.Node, context *RuleFunctionContext) (*yaml.Node, error) {
		if node.Value == "" {
			// Modify the node in place
			node.Value = "TODO: Add description"
			return node, nil
		}
		return node, nil
	}

	// Apply auto-fix to each empty description
	fixedCount := 0
	for _, node := range nodes {
		if node.Value == "" {
			_, err := autoFixEmptyDescription(node, &document, nil)
			assert.NoError(t, err)
			fixedCount++
		}
	}

	// Verify we fixed 3 descriptions
	assert.Equal(t, 3, fixedCount)

	// Verify that the nodes themselves were modified
	for _, node := range nodes {
		assert.Equal(t, "TODO: Add description", node.Value)
	}
}

// TestAutoFixFunction_CamelCaseProperty tests a simpler auto-fix concept
func TestAutoFixFunction_CamelCaseProperty(t *testing.T) {
	sampleYaml := `
user_name: "john"
first_name: "John"
last_name: "Doe"
`

	var document yaml.Node
	err := yaml.Unmarshal([]byte(sampleYaml), &document)
	assert.NoError(t, err)

	// Find all keys in the document
	path := "$.*~"
	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	
	// Should find 3 property names
	assert.Len(t, nodes, 3)

	// Auto-fix function to convert snake_case to camelCase
	autoFixCamelCase := func(node *yaml.Node, document *yaml.Node, context *RuleFunctionContext) (*yaml.Node, error) {
		if node.Kind == yaml.ScalarNode && node.Value != "" {
			// Simple snake_case to camelCase conversion
			camelCaseValue := toCamelCase(node.Value)
			if camelCaseValue != node.Value {
				node.Value = camelCaseValue
				return node, nil
			}
		}
		return node, nil
	}

	// Apply auto-fix
	fixedCount := 0
	for _, node := range nodes {
		originalValue := node.Value
		_, err := autoFixCamelCase(node, &document, nil)
		assert.NoError(t, err)
		
		if node.Value != originalValue {
			fixedCount++
		}
	}

	// Verify we fixed property names
	assert.Equal(t, 3, fixedCount)

	// Verify that the nodes themselves were modified
	expectedValues := []string{"userName", "firstName", "lastName"}
	actualValues := make([]string, len(nodes))
	for i, node := range nodes {
		actualValues[i] = node.Value
	}
	
	for _, expected := range expectedValues {
		assert.Contains(t, actualValues, expected)
	}
}

// TestRuleWithAutoFix tests integrating AutoFixFunction with the Rule struct
func TestRuleWithAutoFix(t *testing.T) {
	// Create a rule with an auto-fix function name
	rule := Rule{
		Id:              "empty-description",
		Description:     "Descriptions should not be empty",
		Message:         "Empty description found",
		Given:           "$..description",
		Severity:        SeverityWarn,
		AutoFixFunction: "fixEmptyDescription",
	}

	// Verify rule was created with autofix function name
	assert.Equal(t, "empty-description", rule.Id)
	assert.Equal(t, "fixEmptyDescription", rule.AutoFixFunction)

	// Define the actual auto-fix function
	autoFixFunction := func(node *yaml.Node, document *yaml.Node, context *RuleFunctionContext) (*yaml.Node, error) {
		if node.Value == "" {
			node.Value = "TODO: Add description"
			return node, nil
		}
		return node, nil
	}

	// Test the function directly
	testNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "",
	}

	fixedNode, err := autoFixFunction(testNode, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "TODO: Add description", fixedNode.Value)
}

// TestRuleWithAutoFixField tests the new AutoFixFunction field in Rule struct
func TestRuleWithAutoFixField_NewField(t *testing.T) {
	// Define a simple auto-fix function
	autoFixFunction := func(node *yaml.Node, document *yaml.Node, context *RuleFunctionContext) (*yaml.Node, error) {
		if node.Value == "" {
			node.Value = "TODO: Add description"
		}
		return node, nil
	}

	// Create a rule with the new AutoFixFunction field
	rule := Rule{
		Id:              "empty-description",
		Description:     "Descriptions should not be empty",
		Message:         "Empty description found",
		Given:           "$..description",
		Severity:        SeverityWarn,
		AutoFixFunction: "fixEmptyDescription",
	}

	// Verify the rule has the auto-fix function name
	assert.Equal(t, "empty-description", rule.Id)
	assert.Equal(t, "fixEmptyDescription", rule.AutoFixFunction)

	// Test that the auto-fix function works directly
	testNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "",
	}

	fixedNode, err := autoFixFunction(testNode, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "TODO: Add description", fixedNode.Value)
}

// TestRuleFunctionResult_AutoFixed tests the new AutoFixed field
func TestRuleFunctionResult_AutoFixed(t *testing.T) {
	// Create a result that was auto-fixed
	result := RuleFunctionResult{
		Message:      "Empty description found",
		Path:         "$.info.description",
		RuleId:       "empty-description",
		RuleSeverity: SeverityWarn,
		AutoFixed:    true,
	}

	assert.Equal(t, "Empty description found", result.Message)
	assert.Equal(t, "empty-description", result.RuleId)
	assert.True(t, result.AutoFixed)

	// Create a result that was not auto-fixed
	unfixedResult := RuleFunctionResult{
		Message:      "Complex issue found",
		Path:         "$.paths./test",
		RuleId:       "complex-rule",
		RuleSeverity: SeverityError,
		AutoFixed:    false,
	}

	assert.False(t, unfixedResult.AutoFixed)
}

// TestCustomAutoFix demonstrates how users can add their own auto-fix functions
func TestCustomAutoFix(t *testing.T) {
	// User defines their own auto-fix function
	customAutoFix := func(node *yaml.Node, document *yaml.Node, context *RuleFunctionContext) (*yaml.Node, error) {
		// Custom logic - e.g., fix specific naming convention
		if strings.HasPrefix(node.Value, "bad_") {
			node.Value = strings.TrimPrefix(node.Value, "bad_")
		}
		return node, nil
	}

	// Test the custom auto-fix function directly
	testNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "bad_example",
	}

	fixedNode, err := customAutoFix(testNode, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "example", fixedNode.Value)
}

// Helper function to convert snake_case to camelCase
func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	
	result := ""
	capitalizeNext := false
	
	for i, r := range s {
		if r == '_' {
			capitalizeNext = true
		} else if capitalizeNext && i > 0 {
			result += string(r - 32) // Convert to uppercase
			capitalizeNext = false
		} else {
			result += string(r)
		}
	}
	
	return result
}
