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
