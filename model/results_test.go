package model

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestRuleResultSet_PrepareForSerialization(t *testing.T) {

	r1 := RuleFunctionResult{Rule: &Rule{
		Description:  "one",
		Severity:     SeverityError,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 1, Column: 10}, EndNode: &yaml.Node{Line: 20, Column: 20}}
	r2 := RuleFunctionResult{Rule: &Rule{
		Description:  "two",
		Severity:     SeverityInfo,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 1, Column: 40}, EndNode: &yaml.Node{Line: 50, Column: 30}}
	r3 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityWarn,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 1, Column: 15}, EndNode: &yaml.Node{Line: 100, Column: 10}}
	r4 := RuleFunctionResult{Rule: &Rule{
		Description:  "three",
		Severity:     SeverityHint,
		RuleCategory: RuleCategories[CategoryInfo],
	}, StartNode: &yaml.Node{Line: 1, Column: 1999}, EndNode: &yaml.Node{Line: 8989899, Column: 10}}

	results := NewRuleResultSet([]RuleFunctionResult{r1, r2, r3, r4})

	d := []byte("what a lovely bucket and spade\nI do love to be beside the seaside.")

	specInfo := datamodel.SpecInfo{
		SpecBytes: &d,
	}

	results.PrepareForSerialization(&specInfo)

	for _, r := range results.Results {
		assert.NotNil(t, r.Range)
		assert.Greater(t, r.Range.Start.Line, 0)
		assert.Greater(t, r.Range.End.Line, 0)
		assert.NotNil(t, r.RuleId)
		assert.NotNil(t, r.RuleSeverity)
	}

}
