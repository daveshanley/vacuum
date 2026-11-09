package model

import (
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/assert"
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

func TestSpectralReportKeepsNativeProjectionUnlessOptedIn(t *testing.T) {
	result := RuleFunctionResult{
		Rule:      &Rule{Id: "r", Severity: SeverityWarn},
		Path:      `$.paths['/v1/file.json#stream'].responses['200']`,
		StartNode: &yaml.Node{Line: 2, Column: 3}, EndNode: &yaml.Node{Line: 2, Column: 8},
	}
	// Existing native functions can populate a one-based range. It must not
	// override report coordinates derived from source nodes.
	result.Range.Start.Line = 2
	result.Range.Start.Char = 3
	report := NewRuleResultSet([]RuleFunctionResult{result}).GenerateSpectralReport("api.yaml")
	assert.Equal(t, []string{"paths['/v1/file", "json#stream']", "responses", "200"}, report[0].Path)
	assert.Equal(t, 2, report[0].Range.Start.Line)
	assert.Equal(t, 3, report[0].Range.Start.Char)
	result.RangeIsZeroBased = true
	result.Range.Start.Line = 0
	result.Range.Start.Char = 0
	result.Range.End.Line = 0
	result.Range.End.Char = 4
	report = NewRuleResultSet([]RuleFunctionResult{result}).GenerateSpectralReport("api.yaml")
	assert.Equal(t, []string{"paths", "/v1/file.json#stream", "responses", "200"}, report[0].Path)
	assert.Equal(t, 0, report[0].Range.Start.Char)
	assert.Equal(t, 4, report[0].Range.End.Char)
}

func TestSpectralNativeRootPathRemainsNil(t *testing.T) {
	result := RuleFunctionResult{Rule: &Rule{Id: "r", Severity: SeverityWarn}, Path: "$", StartNode: &yaml.Node{Line: 1, Column: 1}}
	report := NewRuleResultSet([]RuleFunctionResult{result}).GenerateSpectralReport("api.yaml")
	assert.Nil(t, report[0].Path)
	assert.Equal(t, 1, report[0].Range.Start.Line)
	result.RangeIsZeroBased = true
	report = NewRuleResultSet([]RuleFunctionResult{result}).GenerateSpectralReport("api.yaml")
	assert.NotNil(t, report[0].Path)
	assert.Empty(t, report[0].Path)
	assert.Zero(t, report[0].Range.Start.Line)
}
