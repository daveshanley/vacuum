package cui

import (
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestMinimum(t *testing.T) {
	assert.Equal(t, 5, minimum(5, 10))
	assert.Equal(t, 5, minimum(10, 5))
	assert.Equal(t, 5, minimum(5, 5))
	assert.Equal(t, -10, minimum(-10, 5))
}

func TestViolationResultTableModel_CalculateSplitViewDimensions(t *testing.T) {
	model := &ViolationResultTableModel{
		width:  200,
		height: 50,
	}

	dims := model.calculateSplitViewDimensions()
	
	assert.Equal(t, 200, dims.splitWidth)
	assert.Equal(t, SplitViewHeight, dims.splitHeight)
	assert.Equal(t, SplitContentHeight, dims.contentHeight)
	
	innerWidth := dims.splitWidth - 4
	expectedDetails := int(float64(innerWidth) * float64(DetailsColumnPercent) / 100)
	expectedHowToFix := int(float64(innerWidth) * float64(HowToFixColumnPercent) / 100)
	expectedCode := innerWidth - expectedDetails - expectedHowToFix
	
	assert.Equal(t, expectedDetails, dims.detailsWidth)
	assert.Equal(t, expectedHowToFix, dims.howToFixWidth)
	assert.Equal(t, expectedCode, dims.codeWidth)
}

func TestViolationResultTableModel_BuildPathBar(t *testing.T) {
	tests := []struct {
		name     string
		content  *model.RuleFunctionResult
		width    int
		contains string
	}{
		{
			name: "single path",
			content: &model.RuleFunctionResult{
				Path: "$.components.schemas.User",
			},
			width:    100,
			contains: "$.components.schemas.User",
		},
		{
			name: "multiple paths uses first",
			content: &model.RuleFunctionResult{
				Paths: []string{"$.path1", "$.path2", "$.path3"},
			},
			width:    100,
			contains: "$.path1",
		},
		{
			name: "truncated long path",
			content: &model.RuleFunctionResult{
				Path: "$.very.long.path.that.exceeds.the.available.width.and.needs.to.be.truncated",
			},
			width:    50,
			contains: "...",
		},
		{
			name:     "empty path",
			content:  &model.RuleFunctionResult{},
			width:    100,
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &ViolationResultTableModel{
				modalContent: tt.content,
			}
			result := model.buildPathBar(tt.width)
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
			}
		})
	}
}

func TestViolationResultTableModel_BuildDetailsPanel(t *testing.T) {
	content := &model.RuleFunctionResult{
		Message: "This is a test error message",
		Rule: &model.Rule{
			Id:       "test-rule",
			Severity: "error",
		},
		StartNode: &yaml.Node{Line: 10, Column: 5},
	}

	model := &ViolationResultTableModel{
		modalContent: content,
		fileName:     "test.yaml",
	}

	panel := model.buildDetailsPanel(50, 10)
	
	// check for expected content
	assert.Contains(t, panel, "test-rule")
	assert.Contains(t, panel, "test.yaml:10:5")
	assert.Contains(t, panel, "test error message")
}

func TestViolationResultTableModel_BuildHowToFixPanel(t *testing.T) {
	tests := []struct {
		name     string
		content  *model.RuleFunctionResult
		contains string
	}{
		{
			name: "with fix suggestions",
			content: &model.RuleFunctionResult{
				Rule: &model.Rule{
					HowToFix: "To fix this issue:\n1. Do this\n2. Then do that",
				},
			},
			contains: "To fix this issue",
		},
		{
			name:     "no fix suggestions",
			content:  &model.RuleFunctionResult{},
			contains: "No fix suggestions available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &ViolationResultTableModel{
				modalContent: tt.content,
			}
			panel := model.buildHowToFixPanel(50, 10)
			assert.Contains(t, panel, tt.contains)
		})
	}
}

func TestViolationResultTableModel_BuildCodePanel(t *testing.T) {
	specContent := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get users`)

	tests := []struct {
		name     string
		content  *model.RuleFunctionResult
		fileName string
		contains []string
	}{
		{
			name: "yaml file with code",
			content: &model.RuleFunctionResult{
				StartNode: &yaml.Node{Line: 3},
			},
			fileName: "test.yaml",
			contains: []string{"title:", "3", "▶"},
		},
		{
			name: "json file with code",
			content: &model.RuleFunctionResult{
				Origin: &index.NodeOrigin{Line: 7},
			},
			fileName: "test.json",
			contains: []string{"get:", "7", "▶"},
		},
		{
			name:     "no code context",
			content:  &model.RuleFunctionResult{},
			fileName: "test.yaml",
			contains: []string{"No code context available"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &ViolationResultTableModel{
				modalContent: tt.content,
				fileName:     tt.fileName,
				specContent:  specContent,
			}
			
			if tt.content.StartNode == nil && tt.content.Origin == nil {
				model.specContent = nil
			}
			
			panel := model.buildCodePanel(80, 10)
			for _, expected := range tt.contains {
				assert.Contains(t, panel, expected)
			}
		})
	}
}

func TestViolationResultTableModel_AssembleSplitView(t *testing.T) {
	dims := splitViewDimensions{
		splitWidth:    200,
		splitHeight:   15,
		contentHeight: 10,
		detailsWidth:  50,
		howToFixWidth: 70,
		codeWidth:     76,
	}

	pathBar := "$.test.path"
	detailsPanel := "Details content"
	howToFixPanel := "How to fix content"
	codePanel := "Code content"

	model := &ViolationResultTableModel{}
	result := model.assembleSplitView(dims, pathBar, detailsPanel, howToFixPanel, codePanel)
	
	// verify all content is included
	assert.Contains(t, result, "$.test.path")
	// the panels are rendered with lipgloss, so the exact content might be styled
	assert.NotEmpty(t, result)
}

func TestViolationResultTableModel_BuildDetailsView(t *testing.T) {
	specContent := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0`)

	t.Run("with valid content", func(t *testing.T) {
		model := &ViolationResultTableModel{
			modalContent: &model.RuleFunctionResult{
				Message: "Test error",
				Path:    "$.info.title",
				Rule: &model.Rule{
					Id:       "test-rule",
					Severity: "error",
					HowToFix: "Fix it like this",
				},
				StartNode: &yaml.Node{Line: 3, Column: 3},
			},
			specContent: specContent,
			fileName:    "test.yaml",
			width:       200,
			height:      50,
		}

		view := model.BuildDetailsView()
		assert.NotEmpty(t, view)
		assert.Contains(t, view, "$.info.title")
	})

	t.Run("nil modal content", func(t *testing.T) {
		model := &ViolationResultTableModel{
			modalContent: nil,
			width:        200,
			height:       50,
		}

		view := model.BuildDetailsView()
		assert.Empty(t, view)
	})

	t.Run("terminal too small", func(t *testing.T) {
		model := &ViolationResultTableModel{
			modalContent: &model.RuleFunctionResult{},
			width:        200,
			height:       10, // too small
		}

		view := model.BuildDetailsView()
		assert.Empty(t, view)
	})
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected int // expected number of lines
	}{
		{
			name:     "short text",
			text:     "Short text",
			width:    20,
			expected: 1,
		},
		{
			name:     "long text needs wrapping",
			text:     "This is a very long text that needs to be wrapped across multiple lines",
			width:    20,
			expected: 4,
		},
		{
			name:     "text with existing newlines",
			text:     "Line 1\nLine 2\nLine 3",
			width:    50,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.text, tt.width)
			lines := strings.Split(result, "\n")
			assert.GreaterOrEqual(t, len(lines), tt.expected)
			
			// verify no line exceeds width
			for _, line := range lines {
				assert.LessOrEqual(t, len(line), tt.width)
			}
		})
	}
}

func TestBuildCodePanelHelpers(t *testing.T) {
	t.Run("line truncation", func(t *testing.T) {
		specContent := []byte(strings.Repeat("x", 200) + "\n" + "short line")
		
		model := &ViolationResultTableModel{
			modalContent: &model.RuleFunctionResult{
				StartNode: &yaml.Node{Line: 1},
			},
			specContent: specContent,
			fileName:    "test.yaml",
		}

		panel := model.buildCodePanel(50, 10)
		// should contain truncation indicator
		assert.Contains(t, panel, "...")
	})

	t.Run("highlighted line padding", func(t *testing.T) {
		specContent := []byte("key: value\nerror line\nmore: data")
		
		model := &ViolationResultTableModel{
			modalContent: &model.RuleFunctionResult{
				StartNode: &yaml.Node{Line: 2},
			},
			specContent: specContent,
			fileName:    "test.yaml",
		}

		panel := model.buildCodePanel(100, 10)
		// highlighted line should have triangle marker
		assert.Contains(t, panel, "▶")
	})
}