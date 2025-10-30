package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// TestAutoFixIntegration tests that auto-fix functions are applied during rule execution
func TestAutoFixIntegration(t *testing.T) {
	// Sample OpenAPI spec with empty description
	spec := `
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

	// Create a custom auto-fix function
	emptyDescriptionFix := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		if node.Value == "" {
			node.Value = "TODO: Add description"
		}
		return node, nil
	}

	// Create a custom rule with auto-fix
	customRule := model.Rule{
		Id:              "empty-description-autofix",
		Description:     "Descriptions should not be empty",
		Message:         "Empty description found",
		Given:           "$..description",
		Severity:        model.SeverityWarn,
		AutoFixFunction: emptyDescriptionFix,
		Then: &model.RuleAction{
			Function: "truthy",
		},
	}

	// Create a ruleset with our custom rule
	customRuleSet := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"empty-description-autofix": &customRule,
		},
	}

	// Execute the ruleset with auto-fix enabled
	execution := &RuleSetExecution{
		RuleSet:        customRuleSet,
		Spec:           []byte(spec),
		SpecFileName:   "test.yaml",
		ApplyAutoFixes: true,
	}

	result := ApplyRulesToRuleSet(execution)

	// Verify that violations were found
	assert.Greater(t, len(result.Results), 0, "Should find violations")

	// Check that some results were auto-fixed
	autoFixedCount := 0
	for _, r := range result.Results {
		if r.AutoFixed {
			autoFixedCount++
		}
	}

	assert.Greater(t, autoFixedCount, 0, "Should have auto-fixed some violations")

	// Verify that the auto-fixed results have the correct rule ID
	for _, r := range result.Results {
		if r.AutoFixed {
			assert.Equal(t, "empty-description-autofix", r.RuleId)
		}
	}
}

// TestAutoFixIntegration_NoAutoFix tests that rules without auto-fix functions work normally
func TestAutoFixIntegration_NoAutoFix(t *testing.T) {
	// Sample OpenAPI spec with empty description
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  description: ""
`

	// Create a rule without auto-fix
	customRule := model.Rule{
		Id:          "empty-description-no-fix",
		Description: "Descriptions should not be empty",
		Message:     "Empty description found",
		Given:       "$..description",
		Severity:    model.SeverityWarn,
		Then: &model.RuleAction{
			Function: "truthy",
		},
	}

	// Create a ruleset with our custom rule
	customRuleSet := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"empty-description-no-fix": &customRule,
		},
	}

	// Execute the ruleset without auto-fix enabled
	execution := &RuleSetExecution{
		RuleSet:        customRuleSet,
		Spec:           []byte(spec),
		SpecFileName:   "test.yaml",
		ApplyAutoFixes: false,
	}

	result := ApplyRulesToRuleSet(execution)

	// Verify that violations were found
	assert.Greater(t, len(result.Results), 0, "Should find violations")

	// Check that no results were auto-fixed
	for _, r := range result.Results {
		assert.False(t, r.AutoFixed, "Should not have auto-fixed any violations")
	}
}

// TestAutoFixIntegration_ResultIntegrity tests that rule results remain intact after auto-fix
func TestAutoFixIntegration_ResultIntegrity(t *testing.T) {
	spec := `
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
`

	emptyDescriptionFix := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		if node.Value == "" {
			node.Value = "TODO: Add description"
		}
		return node, nil
	}

	customRule := model.Rule{
		Id:              "empty-description-integrity",
		Description:     "Descriptions should not be empty",
		Message:         "Empty description found",
		Given:           "$..description",
		Severity:        model.SeverityWarn,
		AutoFixFunction: emptyDescriptionFix,
		Then: &model.RuleAction{
			Function: "truthy",
		},
	}

	customRuleSet := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"empty-description-integrity": &customRule,
		},
	}

	execution := &RuleSetExecution{
		RuleSet:        customRuleSet,
		Spec:           []byte(spec),
		SpecFileName:   "test.yaml",
		ApplyAutoFixes: true,
	}

	result := ApplyRulesToRuleSet(execution)

	assert.Greater(t, len(result.Results), 0, "Should find violations")

	// Verify all results have proper data integrity
	for _, r := range result.Results {
		assert.NotEmpty(t, r.RuleId, "RuleId should be populated")
		assert.NotEmpty(t, r.Message, "Message should be populated")
		assert.NotEmpty(t, r.Path, "Path should be populated")
		assert.NotEmpty(t, r.RuleSeverity, "RuleSeverity should be populated")
		assert.NotNil(t, r.Rule, "Rule reference should be populated")
		
		assert.Equal(t, "empty-description-integrity", r.Rule.Id)
		assert.Equal(t, "Empty description found", r.Message)
		assert.Equal(t, model.SeverityWarn, r.RuleSeverity)
	}

	// Verify we have auto-fixed results
	autoFixedCount := 0
	for _, r := range result.Results {
		if r.AutoFixed {
			autoFixedCount++
		}
	}
	assert.Greater(t, autoFixedCount, 0, "Should have auto-fixed some violations")
}

// TestAutoFixIntegration_ErrorHandling tests auto-fix error scenarios
func TestAutoFixIntegration_ErrorHandling(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  description: ""
`

	// Auto-fix function that always fails
	failingAutoFix := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		return nil, assert.AnError
	}

	customRule := model.Rule{
		Id:              "failing-autofix",
		Description:     "Test failing auto-fix",
		Message:         "Empty description found",
		Given:           "$..description",
		Severity:        model.SeverityWarn,
		AutoFixFunction: failingAutoFix,
		Then: &model.RuleAction{
			Function: "truthy",
		},
	}

	customRuleSet := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"failing-autofix": &customRule,
		},
	}

	execution := &RuleSetExecution{
		RuleSet:        customRuleSet,
		Spec:           []byte(spec),
		SpecFileName:   "test.yaml",
		ApplyAutoFixes: true,
	}

	result := ApplyRulesToRuleSet(execution)

	// Should still find violations despite auto-fix failure
	assert.Greater(t, len(result.Results), 0, "Should find violations")

	// No results should be marked as auto-fixed due to error
	for _, r := range result.Results {
		assert.False(t, r.AutoFixed, "Should not mark as auto-fixed when error occurs")
		assert.NotEmpty(t, r.RuleId, "RuleId should still be populated")
		assert.NotEmpty(t, r.Message, "Message should still be populated")
	}
}

// TestAutoFixIntegration_NilNode tests auto-fix with nil return
func TestAutoFixIntegration_NilNode(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  description: ""
`

	// Auto-fix function that returns nil node
	nilNodeAutoFix := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		return nil, nil
	}

	customRule := model.Rule{
		Id:              "nil-node-autofix",
		Description:     "Test nil node auto-fix",
		Message:         "Empty description found",
		Given:           "$..description",
		Severity:        model.SeverityWarn,
		AutoFixFunction: nilNodeAutoFix,
		Then: &model.RuleAction{
			Function: "truthy",
		},
	}

	customRuleSet := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"nil-node-autofix": &customRule,
		},
	}

	execution := &RuleSetExecution{
		RuleSet:        customRuleSet,
		Spec:           []byte(spec),
		SpecFileName:   "test.yaml",
		ApplyAutoFixes: true,
	}

	result := ApplyRulesToRuleSet(execution)

	assert.Greater(t, len(result.Results), 0, "Should find violations")

	// Should not mark as auto-fixed when nil is returned
	for _, r := range result.Results {
		assert.False(t, r.AutoFixed, "Should not mark as auto-fixed when nil returned")
	}
}

// TestAutoFixIntegration_NoStartNode tests auto-fix when StartNode is nil
func TestAutoFixIntegration_NoStartNode(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
`

	trackingAutoFix := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		return node, nil
	}

	// Rule that might not have StartNode populated
	customRule := model.Rule{
		Id:              "no-startnode-autofix",
		Description:     "Test no StartNode auto-fix",
		Message:         "Missing description",
		Given:           "$.info",
		Severity:        model.SeverityWarn,
		AutoFixFunction: trackingAutoFix,
		Then: &model.RuleAction{
			Function: "truthy",
			Field:    "description",
		},
	}

	customRuleSet := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"no-startnode-autofix": &customRule,
		},
	}

	execution := &RuleSetExecution{
		RuleSet:        customRuleSet,
		Spec:           []byte(spec),
		SpecFileName:   "test.yaml",
		ApplyAutoFixes: true,
	}

	result := ApplyRulesToRuleSet(execution)

	// Should handle gracefully when StartNode is nil
	assert.Greater(t, len(result.Results), 0, "Should find violations")
	
	// Auto-fix should not be called when StartNode is nil
	for _, r := range result.Results {
		if r.StartNode == nil {
			assert.False(t, r.AutoFixed, "Should not auto-fix when StartNode is nil")
		}
	}
}
