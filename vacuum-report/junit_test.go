// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package vacuum_report

import (
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestBuildJUnitReport(t *testing.T) {
	j := testhelp_generateReport()
	j.ResultSet.Results[0].Message = "testing, 123"
	j.ResultSet.Results[0].Path = "$.somewhere.out.there"
	j.ResultSet.Results[0].RuleId = "R0001"
	f := time.Now().Add(-time.Millisecond * 5)
	data := BuildJUnitReport(j.ResultSet, f, []string{"test", "args"})
	assert.GreaterOrEqual(t, len(data), 407)
}

func TestBuildJUnitReportWithConfig_DefaultOnlyErrorsAreFailures(t *testing.T) {
	// Create results with different severities
	results := []model.RuleFunctionResult{
		{
			Rule:      &model.Rule{Id: "error-rule", Severity: model.SeverityError, RuleCategory: model.RuleCategories[model.CategoryInfo]},
			Message:   "This is an error",
			StartNode: &yaml.Node{Line: 1, Column: 1},
			EndNode:   &yaml.Node{Line: 1, Column: 10},
		},
		{
			Rule:      &model.Rule{Id: "warn-rule", Severity: model.SeverityWarn, RuleCategory: model.RuleCategories[model.CategoryInfo]},
			Message:   "This is a warning",
			StartNode: &yaml.Node{Line: 2, Column: 1},
			EndNode:   &yaml.Node{Line: 2, Column: 10},
		},
		{
			Rule:      &model.Rule{Id: "info-rule", Severity: model.SeverityInfo, RuleCategory: model.RuleCategories[model.CategoryInfo]},
			Message:   "This is info",
			StartNode: &yaml.Node{Line: 3, Column: 1},
			EndNode:   &yaml.Node{Line: 3, Column: 10},
		},
		{
			Rule:      &model.Rule{Id: "hint-rule", Severity: model.SeverityHint, RuleCategory: model.RuleCategories[model.CategoryInfo]},
			Message:   "This is a hint",
			StartNode: &yaml.Node{Line: 4, Column: 1},
			EndNode:   &yaml.Node{Line: 4, Column: 10},
		},
	}

	resultSet := model.NewRuleResultSet(results)
	config := JUnitConfig{FailOnWarn: false} // Default: only errors are failures

	data := BuildJUnitReportWithConfig(resultSet, time.Now(), []string{"test.yaml"}, config)
	output := string(data)

	// Should have 4 tests but only 1 failure (the error)
	assert.Contains(t, output, `tests="4"`)
	assert.Contains(t, output, `failures="1"`)

	// Error should have failure element
	assert.Contains(t, output, `<failure message="This is an error" type="ERROR"`)

	// Warning, info, hint should NOT have failure elements
	assert.NotContains(t, output, `<failure message="This is a warning"`)
	assert.NotContains(t, output, `<failure message="This is info"`)
	assert.NotContains(t, output, `<failure message="This is a hint"`)

	// But all should still have testcase elements with properties
	assert.Contains(t, output, `classname="warn-rule"`)
	assert.Contains(t, output, `classname="info-rule"`)
	assert.Contains(t, output, `classname="hint-rule"`)
}

func TestBuildJUnitReportWithConfig_FailOnWarn(t *testing.T) {
	// Create results with different severities
	results := []model.RuleFunctionResult{
		{
			Rule:      &model.Rule{Id: "error-rule", Severity: model.SeverityError, RuleCategory: model.RuleCategories[model.CategoryInfo]},
			Message:   "This is an error",
			StartNode: &yaml.Node{Line: 1, Column: 1},
			EndNode:   &yaml.Node{Line: 1, Column: 10},
		},
		{
			Rule:      &model.Rule{Id: "warn-rule", Severity: model.SeverityWarn, RuleCategory: model.RuleCategories[model.CategoryInfo]},
			Message:   "This is a warning",
			StartNode: &yaml.Node{Line: 2, Column: 1},
			EndNode:   &yaml.Node{Line: 2, Column: 10},
		},
		{
			Rule:      &model.Rule{Id: "info-rule", Severity: model.SeverityInfo, RuleCategory: model.RuleCategories[model.CategoryInfo]},
			Message:   "This is info",
			StartNode: &yaml.Node{Line: 3, Column: 1},
			EndNode:   &yaml.Node{Line: 3, Column: 10},
		},
	}

	resultSet := model.NewRuleResultSet(results)
	config := JUnitConfig{FailOnWarn: true} // Warnings are also failures

	data := BuildJUnitReportWithConfig(resultSet, time.Now(), []string{"test.yaml"}, config)
	output := string(data)

	// Should have 3 tests and 2 failures (error + warning)
	assert.Contains(t, output, `tests="3"`)
	assert.Contains(t, output, `failures="2"`)

	// Error and warning should have failure elements
	assert.Contains(t, output, `<failure message="This is an error" type="ERROR"`)
	assert.Contains(t, output, `<failure message="This is a warning" type="WARN"`)

	// Info should NOT have failure element
	assert.NotContains(t, output, `<failure message="This is info"`)
}

func TestBuildJUnitReportWithConfig_InfoAndHintNeverFailures(t *testing.T) {
	// Even with FailOnWarn, info and hint should never be failures
	results := []model.RuleFunctionResult{
		{
			Rule:      &model.Rule{Id: "info-rule", Severity: model.SeverityInfo, RuleCategory: model.RuleCategories[model.CategoryInfo]},
			Message:   "This is info",
			StartNode: &yaml.Node{Line: 1, Column: 1},
			EndNode:   &yaml.Node{Line: 1, Column: 10},
		},
		{
			Rule:      &model.Rule{Id: "hint-rule", Severity: model.SeverityHint, RuleCategory: model.RuleCategories[model.CategoryInfo]},
			Message:   "This is a hint",
			StartNode: &yaml.Node{Line: 2, Column: 1},
			EndNode:   &yaml.Node{Line: 2, Column: 10},
		},
	}

	resultSet := model.NewRuleResultSet(results)
	config := JUnitConfig{FailOnWarn: true} // Even with this on, info/hint should not be failures

	data := BuildJUnitReportWithConfig(resultSet, time.Now(), []string{"test.yaml"}, config)
	output := string(data)

	// Should have 2 tests and 0 failures
	assert.Contains(t, output, `tests="2"`)
	assert.Contains(t, output, `failures="0"`)

	// Neither should have failure elements
	failureCount := strings.Count(output, "<failure")
	assert.Equal(t, 0, failureCount, "Info and hint should never create failure elements")
}

func TestBuildJUnitReport_BackwardsCompatibility(t *testing.T) {
	// The original BuildJUnitReport function should maintain backwards compatibility
	// by treating warnings as failures (FailOnWarn: true)
	results := []model.RuleFunctionResult{
		{
			Rule:      &model.Rule{Id: "warn-rule", Severity: model.SeverityWarn, RuleCategory: model.RuleCategories[model.CategoryInfo]},
			Message:   "This is a warning",
			StartNode: &yaml.Node{Line: 1, Column: 1},
			EndNode:   &yaml.Node{Line: 1, Column: 10},
		},
	}

	resultSet := model.NewRuleResultSet(results)
	data := BuildJUnitReport(resultSet, time.Now(), []string{"test.yaml"})
	output := string(data)

	// Original function should still treat warnings as failures for backwards compatibility
	assert.Contains(t, output, `failures="1"`)
	assert.Contains(t, output, `<failure message="This is a warning" type="WARN"`)
}
