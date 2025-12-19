// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"fmt"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"go.yaml.in/yaml/v4"
)

// Rule IDs for change violations
const (
	RuleIDBreakingChange = "breaking-change"
	RuleIDAPIChange      = "api-change"
)

// ChangeViolationOptions configures which changes should be converted to violations
type ChangeViolationOptions struct {
	WarnOnChanges   bool // Inject warning violations for non-breaking changes
	ErrorOnBreaking bool // Inject error violations for breaking changes
}

// ChangeViolationRules contains pre-configured rules for change violations
var ChangeViolationRules = struct {
	BreakingChange *model.Rule
	APIChange      *model.Rule
}{
	BreakingChange: &model.Rule{
		Id:           RuleIDBreakingChange,
		Description:  "Detects breaking API changes that may affect consumers",
		Severity:     "error",
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		HowToFix:     "Review this breaking change carefully. Consider versioning your API or communicating the change to consumers before deployment.",
	},
	APIChange: &model.Rule{
		Id:           RuleIDAPIChange,
		Description:  "Detects API changes between specification versions",
		Severity:     "warn",
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		HowToFix:     "Review this API change to ensure it behaves as expected.",
	},
}

// GenerateChangeViolations converts DocumentChanges into vacuum RuleFunctionResult violations.
// It walks all changes and creates violations based on whether they are breaking or not.
// Returns nil if changes is nil or options don't require any violations.
func GenerateChangeViolations(changes *wcModel.DocumentChanges, opts ChangeViolationOptions) []*model.RuleFunctionResult {
	if changes == nil {
		return nil
	}

	if !opts.WarnOnChanges && !opts.ErrorOnBreaking {
		return nil
	}

	// GetAllChanges can panic if internal fields are nil, so we check TotalChanges first
	if changes.TotalChanges() == 0 {
		return nil
	}

	allChanges := changes.GetAllChanges()

	var results []*model.RuleFunctionResult
	now := time.Now()

	for _, change := range allChanges {
		if change == nil {
			continue
		}

		// Determine if we should create a violation for this change
		if change.Breaking && opts.ErrorOnBreaking {
			results = append(results, createBreakingViolation(change, &now))
		} else if !change.Breaking && opts.WarnOnChanges {
			results = append(results, createChangeViolation(change, &now))
		}
	}

	return results
}

// createViolation creates a violation for a change with the specified parameters.
// StartNode and EndNode point to the same node since changes represent a single location.
func createViolation(change *wcModel.Change, timestamp *time.Time, ruleID, severity, messagePrefix string, rule *model.Rule) *model.RuleFunctionResult {
	result := &model.RuleFunctionResult{
		Message:      formatChangeMessage(messagePrefix, change),
		Path:         buildChangePath(change),
		RuleId:       ruleID,
		RuleSeverity: severity,
		Rule:         rule,
		Timestamp:    timestamp,
	}

	if change.Context != nil {
		result.Range = buildRangeFromContext(change.Context)
		result.StartNode = buildNodeFromContext(change.Context)
		result.EndNode = result.StartNode
	}

	return result
}

// createBreakingViolation creates an error-level violation for a breaking change
func createBreakingViolation(change *wcModel.Change, timestamp *time.Time) *model.RuleFunctionResult {
	return createViolation(change, timestamp, RuleIDBreakingChange, "error", "Breaking change", ChangeViolationRules.BreakingChange)
}

// createChangeViolation creates a warning-level violation for a non-breaking change
func createChangeViolation(change *wcModel.Change, timestamp *time.Time) *model.RuleFunctionResult {
	return createViolation(change, timestamp, RuleIDAPIChange, "warn", "API change", ChangeViolationRules.APIChange)
}

// formatChangeMessage creates a human-readable message describing the change
func formatChangeMessage(prefix string, change *wcModel.Change) string {
	changeType := getChangeTypeString(change.ChangeType)

	if change.Property != "" {
		if change.Original != "" && change.New != "" {
			return fmt.Sprintf("%s: %s '%s' changed from '%s' to '%s'",
				prefix, changeType, change.Property, truncate(change.Original, 50), truncate(change.New, 50))
		} else if change.Original != "" {
			return fmt.Sprintf("%s: %s '%s' (was: '%s')",
				prefix, changeType, change.Property, truncate(change.Original, 50))
		} else if change.New != "" {
			return fmt.Sprintf("%s: %s '%s' (now: '%s')",
				prefix, changeType, change.Property, truncate(change.New, 50))
		}
		return fmt.Sprintf("%s: %s '%s'", prefix, changeType, change.Property)
	}

	return fmt.Sprintf("%s: %s detected", prefix, changeType)
}

// getChangeTypeString converts a change type constant to a human-readable string
func getChangeTypeString(changeType int) string {
	switch changeType {
	case wcModel.PropertyAdded:
		return "property added"
	case wcModel.PropertyRemoved:
		return "property removed"
	case wcModel.Modified:
		return "modified"
	case wcModel.ObjectAdded:
		return "object added"
	case wcModel.ObjectRemoved:
		return "object removed"
	default:
		return "change"
	}
}

// buildChangePath constructs the path for the change
func buildChangePath(change *wcModel.Change) string {
	if change.Path != "" {
		return change.Path
	}
	if change.Property != "" {
		return "$." + change.Property
	}
	return "$"
}

// extractLineColumn gets line and column from context, preferring new over original
func extractLineColumn(ctx *wcModel.ChangeContext) (line, column int) {
	if ctx.NewLine != nil {
		line = *ctx.NewLine
	} else if ctx.OriginalLine != nil {
		line = *ctx.OriginalLine
	}

	if ctx.NewColumn != nil {
		column = *ctx.NewColumn
	} else if ctx.OriginalColumn != nil {
		column = *ctx.OriginalColumn
	}

	return line, column
}

// buildRangeFromContext creates a Range from a ChangeContext
func buildRangeFromContext(ctx *wcModel.ChangeContext) reports.Range {
	line, col := extractLineColumn(ctx)
	return reports.Range{
		Start: reports.RangeItem{Line: line, Char: col},
		End:   reports.RangeItem{Line: line, Char: col},
	}
}

// buildNodeFromContext creates a yaml.Node with line/column from a ChangeContext
func buildNodeFromContext(ctx *wcModel.ChangeContext) *yaml.Node {
	line, col := extractLineColumn(ctx)
	return &yaml.Node{
		Kind:   yaml.ScalarNode,
		Line:   line,
		Column: col,
	}
}

// truncate shortens a string to the specified length with ellipsis if needed
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// CountBreakingChanges returns the number of breaking changes in a DocumentChanges
func CountBreakingChanges(changes *wcModel.DocumentChanges) int {
	if changes == nil {
		return 0
	}
	return changes.TotalBreakingChanges()
}

// CountNonBreakingChanges returns the number of non-breaking changes in a DocumentChanges
func CountNonBreakingChanges(changes *wcModel.DocumentChanges) int {
	if changes == nil {
		return 0
	}
	return changes.TotalChanges() - changes.TotalBreakingChanges()
}
