package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

// TestRenderMarkdownSummary_Issue700 tests that each rule displays only its own violations
// This test validates the fix for issue #700 where violations were accumulating across rules
func TestRenderMarkdownSummary_Issue700(t *testing.T) {
	// Create mock rule results with different rules in the same category
	results := []model.RuleFunctionResult{
		// Rule 1: plugin_service_protection - 1 violation
		{
			Rule: &model.Rule{
				Id:           "plugin_service_protection_039-partial-id-should-have-default-value-or-not-set",
				RuleCategory: &model.RuleCategory{Id: "validation", Name: "Validation"},
				Severity:     model.SeverityError,
			},
			StartNode: &yaml.Node{Line: 38, Column: 11},
			Path:      "$.services[*].plugins[?(@.name == 'service-protection')].partials[*]",
		},
		// Rule 2: upstream-should-follow-naming-convention - 1 violation
		{
			Rule: &model.Rule{
				Id:           "upstream-should-follow-naming-convention",
				RuleCategory: &model.RuleCategory{Id: "validation", Name: "Validation"},
				Severity:     model.SeverityError,
			},
			StartNode: &yaml.Node{Line: 79, Column: 9},
			Path:      "$..upstreams[*]",
		},
		// Rule 3: service-should-use-upstream-as-host - 2 violations
		{
			Rule: &model.Rule{
				Id:           "service-should-use-upstream-as-host",
				RuleCategory: &model.RuleCategory{Id: "validation", Name: "Validation"},
				Severity:     model.SeverityError,
			},
			StartNode: &yaml.Node{Line: 22, Column: 9},
			Path:      "$.services[*].host",
		},
		{
			Rule: &model.Rule{
				Id:           "service-should-use-upstream-as-host",
				RuleCategory: &model.RuleCategory{Id: "validation", Name: "Validation"},
				Severity:     model.SeverityError,
			},
			StartNode: &yaml.Node{Line: 23, Column: 9},
			Path:      "$..services[*]",
		},
	}

	// Create RuleResultSet using constructor
	rs := model.NewRuleResultSet(results)

	// Create categories
	categories := []*model.RuleCategory{
		{Id: "validation", Name: "Validation"},
	}

	// Create report statistics
	stats := &reports.ReportStatistics{
		TotalErrors:   4,
		TotalWarnings: 0,
		TotalInfo:     0,
		OverallScore:  75,  // Arbitrary score for testing
	}

	// Create render options
	opts := RenderSummaryOptions{
		RuleResultSet:  rs,
		RuleSet:        &rulesets.RuleSet{},
		RuleCategories: categories,
		TotalFiles:     1,
		Severity:       "error",
		Filename:       "test.yaml",
		Silent:         false,
		PipelineOutput: true,  // Enable pipeline output to see violation details
		ReportStats:    stats,
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call the function
	RenderMarkdownSummary(opts)

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify that each rule shows only its own violations
	// Split output by rule sections
	sections := strings.Split(output, "🔴")

	// Find the plugin_service_protection section
	for i, section := range sections {
		if strings.Contains(section, "plugin_service_protection_039") {
			// This section should contain exactly 1 table row (plus header)
			lines := strings.Split(section, "\n")
			tableRows := 0
			for _, line := range lines {
				if strings.Contains(line, "|") && strings.Contains(line, "`") {
					// This is a data row (not header)
					if strings.Contains(line, "38:11") {
						tableRows++
						// Verify this is the correct path
						assert.Contains(t, line, "$.services[*].plugins")
					} else if strings.Contains(line, "79:9") || strings.Contains(line, "22:9") || strings.Contains(line, "23:9") {
						// These locations should NOT be in this rule's table
						t.Errorf("Rule plugin_service_protection should not contain location from other rules: %s", line)
					}
				}
			}
			assert.Equal(t, 1, tableRows, "plugin_service_protection should have exactly 1 violation")
		}

		if strings.Contains(section, "upstream-should-follow-naming-convention") && !strings.Contains(section, "plugin_service_protection") {
			// This section should contain exactly 1 table row
			lines := strings.Split(section, "\n")
			tableRows := 0
			for _, line := range lines {
				if strings.Contains(line, "|") && strings.Contains(line, "`") {
					// This is a data row (not header)
					if strings.Contains(line, "79:9") {
						tableRows++
						// Verify this is the correct path
						assert.Contains(t, line, "$..upstreams[*]")
					} else if strings.Contains(line, "38:11") || strings.Contains(line, "22:9") || strings.Contains(line, "23:9") {
						// These locations should NOT be in this rule's table
						t.Errorf("Rule upstream-should-follow-naming-convention should not contain location from other rules: %s", line)
					}
				}
			}
			assert.Equal(t, 1, tableRows, "upstream-should-follow-naming-convention should have exactly 1 violation")
		}

		if strings.Contains(section, "service-should-use-upstream-as-host") && i > 0 {
			// This section should contain exactly 2 table rows
			lines := strings.Split(section, "\n")
			tableRows := 0
			for _, line := range lines {
				if strings.Contains(line, "|") && strings.Contains(line, "`") {
					// This is a data row (not header)
					if strings.Contains(line, "22:9") || strings.Contains(line, "23:9") {
						tableRows++
						// Verify these contain the correct paths
						if strings.Contains(line, "22:9") {
							assert.Contains(t, line, "$.services[*].host")
						}
						if strings.Contains(line, "23:9") {
							assert.Contains(t, line, "$..services[*]")
						}
					} else if strings.Contains(line, "38:11") || strings.Contains(line, "79:9") {
						// These locations should NOT be in this rule's table
						t.Errorf("Rule service-should-use-upstream-as-host should not contain location from other rules: %s", line)
					}
				}
			}
			assert.Equal(t, 2, tableRows, "service-should-use-upstream-as-host should have exactly 2 violations")
		}
	}

	// Verify output contains expected structure
	assert.Contains(t, output, "### `Validation` violations")
	assert.Contains(t, output, "plugin_service_protection_039-partial-id-should-have-default-value-or-not-set : 1")
	assert.Contains(t, output, "upstream-should-follow-naming-convention : 1")
	assert.Contains(t, output, "service-should-use-upstream-as-host : 2")
}

// TestRenderMarkdownSummary_MultipleCategories tests rendering with multiple categories
func TestRenderMarkdownSummary_MultipleCategories(t *testing.T) {
	// Create mock rule results across different categories
	results := []model.RuleFunctionResult{
		// Validation category
		{
			Rule: &model.Rule{
				Id:           "validation-rule-1",
				RuleCategory: &model.RuleCategory{Id: "validation", Name: "Validation"},
				Severity:     model.SeverityError,
			},
			StartNode: &yaml.Node{Line: 10, Column: 5},
			Path:      "$.test.path1",
		},
		// Security category
		{
			Rule: &model.Rule{
				Id:           "security-rule-1",
				RuleCategory: &model.RuleCategory{Id: "security", Name: "Security"},
				Severity:     model.SeverityWarn,
			},
			StartNode: &yaml.Node{Line: 20, Column: 10},
			Path:      "$.test.path2",
		},
		// Documentation category
		{
			Rule: &model.Rule{
				Id:           "docs-rule-1",
				RuleCategory: &model.RuleCategory{Id: "documentation", Name: "Documentation"},
				Severity:     model.SeverityInfo,
			},
			StartNode: &yaml.Node{Line: 30, Column: 15},
			Path:      "$.test.path3",
		},
	}

	// Create RuleResultSet using constructor
	rs := model.NewRuleResultSet(results)

	// Create categories
	categories := []*model.RuleCategory{
		{Id: "validation", Name: "Validation"},
		{Id: "security", Name: "Security"},
		{Id: "documentation", Name: "Documentation"},
	}

	// Create report statistics
	stats := &reports.ReportStatistics{
		TotalErrors:   1,
		TotalWarnings: 1,
		TotalInfo:     1,
		OverallScore:  85,  // Arbitrary score for testing
	}

	// Create render options
	opts := RenderSummaryOptions{
		RuleResultSet:  rs,
		RuleSet:        &rulesets.RuleSet{},
		RuleCategories: categories,
		TotalFiles:     1,
		Severity:       "error",
		Filename:       "test.yaml",
		Silent:         false,
		PipelineOutput: true,  // Enable pipeline output to see violation details
		ReportStats:    stats,
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call the function
	RenderMarkdownSummary(opts)

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Debug - check what the counts are
	t.Logf("Error count: %d", rs.GetErrorCount())
	t.Logf("Warning count: %d", rs.GetWarnCount())
	t.Logf("Info count: %d", rs.GetInfoCount())

	// Verify each category is rendered
	assert.Contains(t, output, "### `Validation` violations")
	assert.Contains(t, output, "### `Security` violations")
	assert.Contains(t, output, "### `Documentation` violations")

	// Verify each rule appears in its correct category with correct severity
	assert.Contains(t, output, "🔴 validation-rule-1 : 1")
	assert.Contains(t, output, "⚠️️ security-rule-1: 1")
	assert.Contains(t, output, "ℹ️️ docs-rule-1: 1")
}