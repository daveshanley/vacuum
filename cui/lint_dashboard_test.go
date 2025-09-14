package cui

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestUIState_Initialization(t *testing.T) {
	state := UIState{
		ViewMode:       ViewModeTable,
		ActiveModal:    ModalNone,
		ShowPath:       false,
		FilterState:    AllSeverity,
		CategoryFilter: "",
		RuleFilter:     "",
	}

	assert.Equal(t, ViewModeTable, state.ViewMode)
	assert.Equal(t, ModalNone, state.ActiveModal)
	assert.False(t, state.ShowPath)
	assert.Equal(t, AllSeverity, state.FilterState)
	assert.Empty(t, state.CategoryFilter)
	assert.Empty(t, state.RuleFilter)
}

func TestViolationResultTableModel_Init(t *testing.T) {
	model := &ViolationResultTableModel{}
	cmd := model.Init()
	assert.NotNil(t, cmd) // Init now returns a spinner tick command
}

func TestViolationResultTableModel_ToggleSplitView(t *testing.T) {
	model := &ViolationResultTableModel{
		uiState: UIState{
			ViewMode: ViewModeTable,
		},
	}

	// toggle to split view
	model.ToggleSplitView()
	assert.Equal(t, ViewModeTableWithSplit, model.uiState.ViewMode)

	// toggle back to table
	model.ToggleSplitView()
	assert.Equal(t, ViewModeTable, model.uiState.ViewMode)
}

func TestViolationResultTableModel_OpenModal(t *testing.T) {
	testModel := &ViolationResultTableModel{
		uiState: UIState{
			ActiveModal: ModalNone,
		},
	}

	testModel.OpenModal(ModalDocs)
	assert.Equal(t, ModalDocs, testModel.uiState.ActiveModal)

	testModel.OpenModal(ModalCode)
	assert.Equal(t, ModalCode, testModel.uiState.ActiveModal)
}

func TestViolationResultTableModel_CloseActiveModal(t *testing.T) {
	model := &ViolationResultTableModel{
		uiState: UIState{
			ActiveModal: ModalDocs,
		},
	}

	model.CloseActiveModal()
	assert.Equal(t, ModalNone, model.uiState.ActiveModal)
}

func TestViolationResultTableModel_TogglePathColumn(t *testing.T) {
	model := &ViolationResultTableModel{
		uiState: UIState{
			ShowPath: false,
		},
		allResults: []*model.RuleFunctionResult{
			{Message: "test"},
		},
		width: 100,
	}

	// toggle to show path
	model.TogglePathColumn()
	assert.True(t, model.uiState.ShowPath)

	// toggle to hide path
	model.TogglePathColumn()
	assert.False(t, model.uiState.ShowPath)
}

func TestViolationResultTableModel_UpdateFilterState(t *testing.T) {
	testModel := &ViolationResultTableModel{
		uiState: UIState{
			FilterState: AllSeverity,
		},
		allResults: []*model.RuleFunctionResult{
			{Message: "test"},
		},
	}

	// update filter state
	testModel.UpdateFilterState(ErrorSeverity)
	assert.Equal(t, ErrorSeverity, testModel.uiState.FilterState)

	testModel.UpdateFilterState(WarningSeverity)
	assert.Equal(t, WarningSeverity, testModel.uiState.FilterState)

	testModel.UpdateFilterState(InfoSeverity)
	assert.Equal(t, InfoSeverity, testModel.uiState.FilterState)

	testModel.UpdateFilterState(AllSeverity)
	assert.Equal(t, AllSeverity, testModel.uiState.FilterState)
}

func TestViolationResultTableModel_UpdateCategoryFilter(t *testing.T) {
	testModel := &ViolationResultTableModel{
		uiState: UIState{
			CategoryFilter: "",
		},
		allResults: []*model.RuleFunctionResult{
			{
				Rule: &model.Rule{
					RuleCategory: &model.RuleCategory{Name: "validation"},
				},
			},
		},
	}

	// update category filter
	testModel.UpdateCategoryFilter("validation")
	assert.Equal(t, "validation", testModel.uiState.CategoryFilter)

	testModel.UpdateCategoryFilter("schemas")
	assert.Equal(t, "schemas", testModel.uiState.CategoryFilter)

	testModel.UpdateCategoryFilter("")
	assert.Equal(t, "", testModel.uiState.CategoryFilter)
}

func TestViolationResultTableModel_UpdateRuleFilter(t *testing.T) {
	testModel := &ViolationResultTableModel{
		uiState: UIState{
			RuleFilter: "",
		},
		allResults: []*model.RuleFunctionResult{
			{Rule: &model.Rule{Id: "rule1"}},
		},
	}

	// update rule filter
	testModel.UpdateRuleFilter("rule1")
	assert.Equal(t, "rule1", testModel.uiState.RuleFilter)

	testModel.UpdateRuleFilter("rule2")
	assert.Equal(t, "rule2", testModel.uiState.RuleFilter)

	testModel.UpdateRuleFilter("")
	assert.Equal(t, "", testModel.uiState.RuleFilter)
}

func TestFilterState_String(t *testing.T) {
	tests := []struct {
		state    FilterState
		expected string
	}{
		{AllSeverity, "All"},
		{ErrorSeverity, "Errors"},
		{WarningSeverity, "Warnings"},
		{InfoSeverity, "Info"},
		{FilterState(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestGetRuleSeverity(t *testing.T) {
	tests := []struct {
		name     string
		result   *model.RuleFunctionResult
		expected string
	}{
		{
			name:     "nil result",
			result:   nil,
			expected: "✗ error",
		},
		{
			name:     "nil rule",
			result:   &model.RuleFunctionResult{},
			expected: "✗ error",
		},
		{
			name: "error severity",
			result: &model.RuleFunctionResult{
				Rule: &model.Rule{Severity: "error"},
			},
			expected: "✗ error",
		},
		{
			name: "warn severity",
			result: &model.RuleFunctionResult{
				Rule: &model.Rule{Severity: "warn"},
			},
			expected: "▲ warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getRuleSeverity(tt.result))
		})
	}
}

func TestFormatFileLocation(t *testing.T) {
	tests := []struct {
		name     string
		result   *model.RuleFunctionResult
		fileName string
		expected string
	}{
		{
			name:     "nil result",
			result:   nil,
			fileName: "test.yaml",
			expected: "test.yaml",
		},
		{
			name: "with start node",
			result: &model.RuleFunctionResult{
				StartNode: &yaml.Node{Line: 10, Column: 5},
			},
			fileName: "test.yaml",
			expected: "test.yaml:10:5",
		},
		{
			name: "with origin node",
			result: &model.RuleFunctionResult{
				Origin: &index.NodeOrigin{Line: 20, Column: 3},
			},
			fileName: "test.yaml",
			expected: "test.yaml:20:3",
		},
		{
			name: "both nodes (start takes precedence)",
			result: &model.RuleFunctionResult{
				StartNode: &yaml.Node{Line: 10, Column: 5},
				Origin:    &index.NodeOrigin{Line: 20, Column: 3},
			},
			fileName: "test.yaml",
			expected: "test.yaml:10:5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatFileLocation(tt.result, tt.fileName))
		})
	}
}
