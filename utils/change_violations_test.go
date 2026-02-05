// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"
	"time"

	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateChangeViolations_NilChanges(t *testing.T) {
	opts := ChangeViolationOptions{
		WarnOnChanges:   true,
		ErrorOnBreaking: true,
	}

	results := GenerateChangeViolations(nil, opts)
	assert.Nil(t, results)
}

func TestGenerateChangeViolations_NoOptionsEnabled(t *testing.T) {
	changes := &wcModel.DocumentChanges{}

	opts := ChangeViolationOptions{
		WarnOnChanges:   false,
		ErrorOnBreaking: false,
	}

	results := GenerateChangeViolations(changes, opts)
	assert.Nil(t, results)
}

func TestGenerateChangeViolations_EmptyChanges(t *testing.T) {
	changes := &wcModel.DocumentChanges{}

	opts := ChangeViolationOptions{
		WarnOnChanges:   true,
		ErrorOnBreaking: true,
	}

	results := GenerateChangeViolations(changes, opts)
	assert.Nil(t, results)
}

func TestGenerateChangeViolations_BreakingChange(t *testing.T) {
	// Create a breaking change
	change := &wcModel.Change{
		ChangeType: wcModel.Modified,
		Property:   "type",
		Original:   "string",
		New:        "integer",
		Breaking:   true,
		Path:       "$.components.schemas.User.properties.id.type",
	}

	changes := createDocumentChangesWithChange(change)

	opts := ChangeViolationOptions{
		WarnOnChanges:   false,
		ErrorOnBreaking: true,
	}

	results := GenerateChangeViolations(changes, opts)
	require.NotNil(t, results)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, RuleIDBreakingChange, result.RuleId)
	assert.Equal(t, "error", result.RuleSeverity)
	assert.Contains(t, result.Message, "Breaking change")
	assert.Contains(t, result.Message, "type")
	assert.Equal(t, "$.components.schemas.User.properties.id.type", result.Path)
}

func TestGenerateChangeViolations_NonBreakingChange(t *testing.T) {
	// Create a non-breaking change
	change := &wcModel.Change{
		ChangeType: wcModel.Modified,
		Property:   "description",
		Original:   "Old description",
		New:        "New description",
		Breaking:   false,
		Path:       "$.info.description",
	}

	changes := createDocumentChangesWithChange(change)

	opts := ChangeViolationOptions{
		WarnOnChanges:   true,
		ErrorOnBreaking: false,
	}

	results := GenerateChangeViolations(changes, opts)
	require.NotNil(t, results)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, RuleIDAPIChange, result.RuleId)
	assert.Equal(t, "warn", result.RuleSeverity)
	assert.Contains(t, result.Message, "API change")
	assert.Contains(t, result.Message, "description")
}

func TestGenerateChangeViolations_BothOptions(t *testing.T) {
	// Create both breaking and non-breaking changes
	breakingChange := &wcModel.Change{
		ChangeType: wcModel.PropertyRemoved,
		Property:   "required_field",
		Original:   "true",
		Breaking:   true,
		Path:       "$.paths./users.get.responses.200.content",
	}

	nonBreakingChange := &wcModel.Change{
		ChangeType: wcModel.PropertyAdded,
		Property:   "optional_field",
		New:        "value",
		Breaking:   false,
		Path:       "$.paths./users.get.responses.200",
	}

	changes := createDocumentChangesWithChanges([]*wcModel.Change{breakingChange, nonBreakingChange})

	opts := ChangeViolationOptions{
		WarnOnChanges:   true,
		ErrorOnBreaking: true,
	}

	results := GenerateChangeViolations(changes, opts)
	require.NotNil(t, results)
	require.Len(t, results, 2)

	// Find breaking and non-breaking results
	var breakingResult, nonBreakingResult *struct {
		ruleId   string
		severity string
	}

	for _, r := range results {
		if r.RuleId == RuleIDBreakingChange {
			breakingResult = &struct {
				ruleId   string
				severity string
			}{r.RuleId, r.RuleSeverity}
		} else if r.RuleId == RuleIDAPIChange {
			nonBreakingResult = &struct {
				ruleId   string
				severity string
			}{r.RuleId, r.RuleSeverity}
		}
	}

	require.NotNil(t, breakingResult)
	assert.Equal(t, "error", breakingResult.severity)

	require.NotNil(t, nonBreakingResult)
	assert.Equal(t, "warn", nonBreakingResult.severity)
}

func TestGenerateChangeViolations_OnlyBreakingOption(t *testing.T) {
	// Create both types of changes but only enable breaking option
	breakingChange := &wcModel.Change{
		ChangeType: wcModel.ObjectRemoved,
		Property:   "endpoint",
		Breaking:   true,
		Path:       "$.paths./deprecated",
	}

	nonBreakingChange := &wcModel.Change{
		ChangeType: wcModel.ObjectAdded,
		Property:   "new_endpoint",
		Breaking:   false,
		Path:       "$.paths./new",
	}

	changes := createDocumentChangesWithChanges([]*wcModel.Change{breakingChange, nonBreakingChange})

	opts := ChangeViolationOptions{
		WarnOnChanges:   false,
		ErrorOnBreaking: true,
	}

	results := GenerateChangeViolations(changes, opts)
	require.NotNil(t, results)
	require.Len(t, results, 1)

	assert.Equal(t, RuleIDBreakingChange, results[0].RuleId)
}

func TestGenerateChangeViolations_OnlyWarningsOption(t *testing.T) {
	// Create both types of changes but only enable warnings option
	breakingChange := &wcModel.Change{
		ChangeType: wcModel.Modified,
		Property:   "type",
		Breaking:   true,
		Path:       "$.components.schemas.User.type",
	}

	nonBreakingChange := &wcModel.Change{
		ChangeType: wcModel.Modified,
		Property:   "description",
		Breaking:   false,
		Path:       "$.components.schemas.User.description",
	}

	changes := createDocumentChangesWithChanges([]*wcModel.Change{breakingChange, nonBreakingChange})

	opts := ChangeViolationOptions{
		WarnOnChanges:   true,
		ErrorOnBreaking: false,
	}

	results := GenerateChangeViolations(changes, opts)
	require.NotNil(t, results)
	require.Len(t, results, 1)

	assert.Equal(t, RuleIDAPIChange, results[0].RuleId)
}

func TestFormatChangeMessage(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		change   *wcModel.Change
		contains []string
	}{
		{
			name:   "modified with original and new",
			prefix: "Breaking change",
			change: &wcModel.Change{
				ChangeType: wcModel.Modified,
				Property:   "type",
				Original:   "string",
				New:        "integer",
			},
			contains: []string{"Breaking change", "modified", "type", "string", "integer"},
		},
		{
			name:   "property removed",
			prefix: "API change",
			change: &wcModel.Change{
				ChangeType: wcModel.PropertyRemoved,
				Property:   "deprecated_field",
				Original:   "value",
			},
			contains: []string{"API change", "property removed", "deprecated_field", "was:"},
		},
		{
			name:   "property added",
			prefix: "API change",
			change: &wcModel.Change{
				ChangeType: wcModel.PropertyAdded,
				Property:   "new_field",
				New:        "default_value",
			},
			contains: []string{"API change", "property added", "new_field", "now:"},
		},
		{
			name:   "object added",
			prefix: "API change",
			change: &wcModel.Change{
				ChangeType: wcModel.ObjectAdded,
				Property:   "new_schema",
			},
			contains: []string{"API change", "object added", "new_schema"},
		},
		{
			name:   "object removed",
			prefix: "Breaking change",
			change: &wcModel.Change{
				ChangeType: wcModel.ObjectRemoved,
				Property:   "old_schema",
			},
			contains: []string{"Breaking change", "object removed", "old_schema"},
		},
		{
			name:   "no property",
			prefix: "API change",
			change: &wcModel.Change{
				ChangeType: wcModel.Modified,
			},
			contains: []string{"API change", "modified", "detected"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatChangeMessage(tt.prefix, tt.change)
			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestGetChangeTypeString(t *testing.T) {
	tests := []struct {
		changeType int
		expected   string
	}{
		{wcModel.PropertyAdded, "property added"},
		{wcModel.PropertyRemoved, "property removed"},
		{wcModel.Modified, "modified"},
		{wcModel.ObjectAdded, "object added"},
		{wcModel.ObjectRemoved, "object removed"},
		{999, "change"}, // unknown type
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := getChangeTypeString(tt.changeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildChangePath(t *testing.T) {
	tests := []struct {
		name     string
		change   *wcModel.Change
		expected string
	}{
		{
			name: "with path",
			change: &wcModel.Change{
				Path:     "$.components.schemas.User",
				Property: "type",
			},
			expected: "$.components.schemas.User",
		},
		{
			name: "with property only",
			change: &wcModel.Change{
				Property: "description",
			},
			expected: "$.description",
		},
		{
			name:     "no path or property",
			change:   &wcModel.Change{},
			expected: "$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildChangePath(tt.change)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildRangeFromContext(t *testing.T) {
	line10 := 10
	col5 := 5
	line20 := 20
	col15 := 15

	tests := []struct {
		name         string
		ctx          *wcModel.ChangeContext
		expectedLine int
		expectedChar int
	}{
		{
			name: "new line and column",
			ctx: &wcModel.ChangeContext{
				NewLine:   &line10,
				NewColumn: &col5,
			},
			expectedLine: 10,
			expectedChar: 5,
		},
		{
			name: "original line and column fallback",
			ctx: &wcModel.ChangeContext{
				OriginalLine:   &line20,
				OriginalColumn: &col15,
			},
			expectedLine: 20,
			expectedChar: 15,
		},
		{
			name: "new takes precedence over original",
			ctx: &wcModel.ChangeContext{
				NewLine:        &line10,
				NewColumn:      &col5,
				OriginalLine:   &line20,
				OriginalColumn: &col15,
			},
			expectedLine: 10,
			expectedChar: 5,
		},
		{
			name:         "empty context",
			ctx:          &wcModel.ChangeContext{},
			expectedLine: 0,
			expectedChar: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRangeFromContext(tt.ctx)
			assert.Equal(t, tt.expectedLine, result.Start.Line)
			assert.Equal(t, tt.expectedChar, result.Start.Char)
			assert.Equal(t, tt.expectedLine, result.End.Line)
			assert.Equal(t, tt.expectedChar, result.End.Char)
		})
	}
}

func TestBuildNodeFromContext(t *testing.T) {
	line42 := 42
	col10 := 10
	line100 := 100
	col5 := 5

	tests := []struct {
		name           string
		ctx            *wcModel.ChangeContext
		expectedLine   int
		expectedColumn int
	}{
		{
			name: "new line and column",
			ctx: &wcModel.ChangeContext{
				NewLine:   &line42,
				NewColumn: &col10,
			},
			expectedLine:   42,
			expectedColumn: 10,
		},
		{
			name: "original line and column fallback",
			ctx: &wcModel.ChangeContext{
				OriginalLine:   &line100,
				OriginalColumn: &col5,
			},
			expectedLine:   100,
			expectedColumn: 5,
		},
		{
			name: "new takes precedence over original",
			ctx: &wcModel.ChangeContext{
				OriginalLine:   &line100,
				OriginalColumn: &col5,
				NewLine:        &line42,
				NewColumn:      &col10,
			},
			expectedLine:   42,
			expectedColumn: 10,
		},
		{
			name:           "empty context",
			ctx:            &wcModel.ChangeContext{},
			expectedLine:   0,
			expectedColumn: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildNodeFromContext(tt.ctx)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedLine, result.Line)
			assert.Equal(t, tt.expectedColumn, result.Column)
		})
	}
}

func TestExtractLineColumn(t *testing.T) {
	line42 := 42
	col10 := 10
	line100 := 100
	col5 := 5

	tests := []struct {
		name           string
		ctx            *wcModel.ChangeContext
		expectedLine   int
		expectedColumn int
	}{
		{
			name: "new line and column",
			ctx: &wcModel.ChangeContext{
				NewLine:   &line42,
				NewColumn: &col10,
			},
			expectedLine:   42,
			expectedColumn: 10,
		},
		{
			name: "original line and column fallback",
			ctx: &wcModel.ChangeContext{
				OriginalLine:   &line100,
				OriginalColumn: &col5,
			},
			expectedLine:   100,
			expectedColumn: 5,
		},
		{
			name: "new takes precedence over original",
			ctx: &wcModel.ChangeContext{
				OriginalLine:   &line100,
				OriginalColumn: &col5,
				NewLine:        &line42,
				NewColumn:      &col10,
			},
			expectedLine:   42,
			expectedColumn: 10,
		},
		{
			name:           "empty context returns zeros",
			ctx:            &wcModel.ChangeContext{},
			expectedLine:   0,
			expectedColumn: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, col := extractLineColumn(tt.ctx)
			assert.Equal(t, tt.expectedLine, line)
			assert.Equal(t, tt.expectedColumn, col)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a very long string", 10, "this is..."},
		{"tiny", 3, "tin"},
		{"ab", 3, "ab"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountBreakingChanges(t *testing.T) {
	t.Run("nil changes", func(t *testing.T) {
		assert.Equal(t, 0, CountBreakingChanges(nil))
	})

	t.Run("empty changes", func(t *testing.T) {
		// Empty DocumentChanges needs PropertyChanges initialized to avoid nil panic
		changes := &wcModel.DocumentChanges{
			PropertyChanges: &wcModel.PropertyChanges{},
		}
		assert.Equal(t, 0, CountBreakingChanges(changes))
	})

	t.Run("with breaking changes", func(t *testing.T) {
		changes := createDocumentChangesWithChanges([]*wcModel.Change{
			{Breaking: true, Property: "test1"},
			{Breaking: true, Property: "test2"},
			{Breaking: false, Property: "test3"},
		})
		assert.Equal(t, 2, CountBreakingChanges(changes))
	})
}

func TestCountNonBreakingChanges(t *testing.T) {
	t.Run("nil changes", func(t *testing.T) {
		assert.Equal(t, 0, CountNonBreakingChanges(nil))
	})

	t.Run("empty changes", func(t *testing.T) {
		// Empty DocumentChanges needs PropertyChanges initialized to avoid nil panic
		changes := &wcModel.DocumentChanges{
			PropertyChanges: &wcModel.PropertyChanges{},
		}
		assert.Equal(t, 0, CountNonBreakingChanges(changes))
	})

	t.Run("with non-breaking changes", func(t *testing.T) {
		changes := createDocumentChangesWithChanges([]*wcModel.Change{
			{Breaking: true, Property: "test1"},
			{Breaking: false, Property: "test2"},
			{Breaking: false, Property: "test3"},
		})
		assert.Equal(t, 2, CountNonBreakingChanges(changes))
	})
}

func TestChangeViolationRules(t *testing.T) {
	// Verify the pre-configured rules have correct values
	assert.Equal(t, RuleIDBreakingChange, ChangeViolationRules.BreakingChange.Id)
	assert.Equal(t, "error", ChangeViolationRules.BreakingChange.Severity)
	assert.NotEmpty(t, ChangeViolationRules.BreakingChange.Description)
	assert.NotEmpty(t, ChangeViolationRules.BreakingChange.HowToFix)

	assert.Equal(t, RuleIDAPIChange, ChangeViolationRules.APIChange.Id)
	assert.Equal(t, "warn", ChangeViolationRules.APIChange.Severity)
	assert.NotEmpty(t, ChangeViolationRules.APIChange.Description)
	assert.NotEmpty(t, ChangeViolationRules.APIChange.HowToFix)
}

func TestCreateBreakingViolation(t *testing.T) {
	now := time.Now()
	line := 42
	col := 10

	change := &wcModel.Change{
		ChangeType: wcModel.Modified,
		Property:   "type",
		Original:   "string",
		New:        "number",
		Breaking:   true,
		Path:       "$.components.schemas.User.properties.id.type",
		Context: &wcModel.ChangeContext{
			NewLine:   &line,
			NewColumn: &col,
		},
	}

	result := createBreakingViolation(change, &now)

	assert.Equal(t, RuleIDBreakingChange, result.RuleId)
	assert.Equal(t, "error", result.RuleSeverity)
	assert.Equal(t, ChangeViolationRules.BreakingChange, result.Rule)
	assert.Equal(t, "$.components.schemas.User.properties.id.type", result.Path)
	assert.Contains(t, result.Message, "Breaking change")
	assert.Equal(t, 42, result.Range.Start.Line)
	assert.Equal(t, 10, result.Range.Start.Char)
	assert.Equal(t, &now, result.Timestamp)
	// Verify StartNode is set for proper line/column display
	assert.NotNil(t, result.StartNode)
	assert.Equal(t, 42, result.StartNode.Line)
	assert.Equal(t, 10, result.StartNode.Column)
	assert.NotNil(t, result.EndNode)
}

func TestCreateChangeViolation(t *testing.T) {
	now := time.Now()
	line := 100
	col := 5

	change := &wcModel.Change{
		ChangeType: wcModel.PropertyAdded,
		Property:   "newField",
		New:        "defaultValue",
		Breaking:   false,
		Path:       "$.info",
		Context: &wcModel.ChangeContext{
			NewLine:   &line,
			NewColumn: &col,
		},
	}

	result := createChangeViolation(change, &now)

	assert.Equal(t, RuleIDAPIChange, result.RuleId)
	assert.Equal(t, "warn", result.RuleSeverity)
	assert.Equal(t, ChangeViolationRules.APIChange, result.Rule)
	assert.Equal(t, "$.info", result.Path)
	assert.Contains(t, result.Message, "API change")
	assert.Equal(t, 100, result.Range.Start.Line)
	assert.Equal(t, 5, result.Range.Start.Char)
	assert.Equal(t, &now, result.Timestamp)
	// Verify StartNode is set for proper line/column display
	assert.NotNil(t, result.StartNode)
	assert.Equal(t, 100, result.StartNode.Line)
	assert.Equal(t, 5, result.StartNode.Column)
	assert.NotNil(t, result.EndNode)
}

// Helper function to create DocumentChanges with a single change for testing
func createDocumentChangesWithChange(change *wcModel.Change) *wcModel.DocumentChanges {
	return createDocumentChangesWithChanges([]*wcModel.Change{change})
}

// Helper function to create DocumentChanges with multiple changes for testing
func createDocumentChangesWithChanges(changes []*wcModel.Change) *wcModel.DocumentChanges {
	// Create a minimal DocumentChanges structure that will return changes via GetAllChanges()
	// DocumentChanges embeds *PropertyChanges which must also be initialized to avoid nil panics
	return &wcModel.DocumentChanges{
		PropertyChanges: &wcModel.PropertyChanges{
			Changes: changes,
		},
	}
}
