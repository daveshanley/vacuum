package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// TestAutoFixIntegration tests basic auto-fix functionality
func TestAutoFixIntegration(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  description: ""
`

	emptyDescriptionFix := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		if node.Value == "" {
			node.Value = "TODO: Add description"
		}
		return node, nil
	}

	customRule := model.Rule{
		Id:              "empty-description-autofix",
		Message:         "Empty description found",
		Given:           "$.info.description",
		Severity:        model.SeverityWarn,
		AutoFixFunction: "fixEmptyDescription",
		Then: &model.RuleAction{
			Function: "truthy",
		},
	}

	execution := &RuleSetExecution{
		RuleSet:          &rulesets.RuleSet{Rules: map[string]*model.Rule{"empty-description-autofix": &customRule}},
		Spec:             []byte(spec),
		SpecFileName:     "test.yaml",
		ApplyAutoFixes:   true,
		AutoFixFunctions: map[string]model.AutoFixFunction{"fixEmptyDescription": emptyDescriptionFix},
	}

	result := ApplyRulesToRuleSet(execution)

	// Should have fixed results
	assert.Greater(t, len(result.FixedResults), 0, "Should have fixed some violations")
	
	// Fixed results should have AutoFixed flag set
	for _, r := range result.FixedResults {
		assert.True(t, r.AutoFixed)
		assert.Equal(t, "empty-description-autofix", r.RuleId)
	}
}

// TestAutoFixDisabled tests that autofix doesn't run when disabled
func TestAutoFixDisabled(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  description: ""
`

	customRule := model.Rule{
		Id:       "empty-description",
		Message:  "Empty description found", 
		Given:    "$.info.description",
		Severity: model.SeverityWarn,
		Then: &model.RuleAction{Function: "truthy"},
	}

	execution := &RuleSetExecution{
		RuleSet:        &rulesets.RuleSet{Rules: map[string]*model.Rule{"empty-description": &customRule}},
		Spec:           []byte(spec),
		ApplyAutoFixes: false,
	}

	result := ApplyRulesToRuleSet(execution)

	// Should have regular results, no fixed results
	assert.Greater(t, len(result.Results), 0)
	assert.Equal(t, 0, len(result.FixedResults))
}
