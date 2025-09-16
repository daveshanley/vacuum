package tui

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestBuildTableRows(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			Message: "Test message 1",
			Path:    "$.test.path1",
			Rule: &model.Rule{
				Id:       "test-rule-1",
				Severity: "error",
				RuleCategory: &model.RuleCategory{
					Name: "validation",
				},
			},
			StartNode: &yaml.Node{Line: 10, Column: 5},
		},
		{
			Message: "Test message 2",
			Path:    "$.test.path2",
			Rule: &model.Rule{
				Id:       "test-rule-2",
				Severity: "warn",
				RuleCategory: &model.RuleCategory{
					Name: "schemas",
				},
			},
			Origin: &index.NodeOrigin{Line: 20, Column: 3},
		},
	}

	t.Run("with path column", func(t *testing.T) {
		rows := buildTableRows(results, "test.yaml", true)
		assert.Len(t, rows, 2)
		assert.Len(t, rows[0], 6) // 6 columns with path
		assert.Equal(t, "test.yaml:10:5", rows[0][0])
		assert.Equal(t, "✗ error", rows[0][1])
		assert.Equal(t, "Test message 1", rows[0][2])
		assert.Equal(t, "test-rule-1", rows[0][3])
		assert.Equal(t, "validation", rows[0][4])
		assert.Equal(t, "$.test.path1", rows[0][5])
	})

	t.Run("without path column", func(t *testing.T) {
		rows := buildTableRows(results, "test.yaml", false)
		assert.Len(t, rows, 2)
		assert.Len(t, rows[0], 5) // 5 columns without path
		assert.Equal(t, "test.yaml:10:5", rows[0][0])
		assert.Equal(t, "✗ error", rows[0][1])
		assert.Equal(t, "Test message 1", rows[0][2])
		assert.Equal(t, "test-rule-1", rows[0][3])
		assert.Equal(t, "validation", rows[0][4])
	})
}

func TestCalculateContentWidths(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			Rule: &model.Rule{
				Id: "very-long-rule-name-that-is-quite-verbose",
				RuleCategory: &model.RuleCategory{
					Name: "extremely-long-category-name",
				},
			},
			StartNode: &yaml.Node{Line: 1000, Column: 100},
		},
		{
			Rule: &model.Rule{
				Id: "short",
				RuleCategory: &model.RuleCategory{
					Name: "tiny",
				},
			},
			StartNode: &yaml.Node{Line: 1, Column: 1},
		},
	}

	widths := calculateContentWidths(results, "very-long-filename-with-lots-of-characters.yaml")

	// should be max of content or header
	assert.GreaterOrEqual(t, widths.location, len("very-long-filename-with-lots-of-characters.yaml:1000:100"))
	assert.GreaterOrEqual(t, widths.rule, len("very-long-rule-name-that-is-quite-verbose"))
	assert.GreaterOrEqual(t, widths.category, len("extremely-long-category-name"))
}

func TestCalculateColumnWidths(t *testing.T) {
	content := contentWidths{
		location: 20,
		rule:     15,
		category: 12,
	}

	t.Run("plenty of space", func(t *testing.T) {
		widths := calculateColumnWidths(200, content, false)

		// with plenty of space, message should get most of it
		assert.Equal(t, 20, widths.location)
		assert.Equal(t, SeverityColumnWidth+1, widths.severity)
		assert.Equal(t, 15, widths.rule)
		assert.Equal(t, 12, widths.category)
		assert.Greater(t, widths.message, 100) // message gets the rest
		assert.Equal(t, 0, widths.path)        // no path column
	})

	t.Run("limited space", func(t *testing.T) {
		widths := calculateColumnWidths(100, content, false)

		// with limited space, columns should be compressed
		assert.Equal(t, 20, widths.location)
		assert.Equal(t, SeverityColumnWidth+1, widths.severity)
		assert.GreaterOrEqual(t, widths.message, 30) // minimum message width
	})

	t.Run("with path column", func(t *testing.T) {
		widths := calculateColumnWidths(200, content, true)

		assert.Greater(t, widths.path, 0) // path column should be present
		assert.Greater(t, widths.message, 0)
	})
}

func TestCompressColumn(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		minWidth    int
		needToSave  int
		expectWidth int
		expectSaved int
	}{
		{
			name:        "no compression needed",
			width:       50,
			minWidth:    20,
			needToSave:  0,
			expectWidth: 50,
			expectSaved: 0,
		},
		{
			name:        "partial compression",
			width:       50,
			minWidth:    20,
			needToSave:  10,
			expectWidth: 40,
			expectSaved: 0,
		},
		{
			name:        "compress to minimum",
			width:       50,
			minWidth:    20,
			needToSave:  40,
			expectWidth: 20,
			expectSaved: 10, // 40 - 30 = 10 still needed
		},
		{
			name:        "already at minimum",
			width:       20,
			minWidth:    20,
			needToSave:  10,
			expectWidth: 20,
			expectSaved: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width := tt.width
			remaining := compressColumn(&width, tt.minWidth, tt.needToSave)
			assert.Equal(t, tt.expectWidth, width)
			assert.Equal(t, tt.expectSaved, remaining)
		})
	}
}

func TestBuildTableColumns(t *testing.T) {
	widths := columnWidths{
		location: 25,
		severity: 10,
		message:  80,
		rule:     20,
		category: 15,
		path:     30,
	}

	t.Run("without path", func(t *testing.T) {
		columns := buildTableColumns(widths, false)
		assert.Len(t, columns, 5)
		assert.Equal(t, "Location", columns[0].Title)
		assert.Equal(t, 25, columns[0].Width)
		assert.Equal(t, "Message", columns[2].Title)
		assert.Equal(t, 80, columns[2].Width)
	})

	t.Run("with path", func(t *testing.T) {
		columns := buildTableColumns(widths, true)
		assert.Len(t, columns, 6)
		assert.Equal(t, "Path", columns[5].Title)
		assert.Equal(t, 30, columns[5].Width)
	})
}

func TestBuildResultTableData_Integration(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			Message: "Test error message that is quite long and detailed",
			Path:    "$.components.schemas.TestSchema.properties.field",
			Rule: &model.Rule{
				Id:       "test-rule",
				Severity: "error",
				RuleCategory: &model.RuleCategory{
					Name: "validation",
				},
			},
			StartNode: &yaml.Node{Line: 100, Column: 5},
		},
		{
			Message: "Another message",
			Path:    "$.paths./users.get",
			Rule: &model.Rule{
				Id:       "path-rule",
				Severity: "warn",
				RuleCategory: &model.RuleCategory{
					Name: "operations",
				},
			},
			Origin: &index.NodeOrigin{Line: 200, Column: 10},
		},
	}

	t.Run("narrow terminal", func(t *testing.T) {
		columns, rows := BuildResultTableData(results, "spec.yaml", 80, false)

		assert.Len(t, columns, 5)
		assert.Len(t, rows, 2)

		// verify columns fit within terminal width
		totalWidth := 0
		for _, col := range columns {
			totalWidth += col.Width
		}
		// account for padding (2 per column)
		totalWidth += len(columns) * 2
		assert.LessOrEqual(t, totalWidth, 80)
	})

	t.Run("wide terminal with path", func(t *testing.T) {
		columns, rows := BuildResultTableData(results, "spec.yaml", 200, true)

		assert.Len(t, columns, 6)
		assert.Len(t, rows, 2)

		// verify path column is included
		assert.Equal(t, "Path", columns[5].Title)

		// verify columns fit within terminal width
		totalWidth := 0
		for _, col := range columns {
			totalWidth += col.Width
		}
		totalWidth += len(columns) * 2
		assert.LessOrEqual(t, totalWidth, 200)
	})

	t.Run("empty results", func(t *testing.T) {
		columns, rows := BuildResultTableData([]*model.RuleFunctionResult{}, "spec.yaml", 100, false)

		assert.Len(t, columns, 5)
		assert.Len(t, rows, 0)
	})
}

func TestCalculateWithPathColumn(t *testing.T) {
	content := contentWidths{
		location: 20,
		rule:     25,
		category: 20,
	}

	t.Run("natural widths fit", func(t *testing.T) {
		widths := columnWidths{
			location: 20,
			severity: 10,
			rule:     25,
			category: 20,
		}

		// 20 + 10 + 80 + 25 + 20 + 50 = 205, available = 250
		calculateWithPathColumn(250, &widths, content)

		assert.GreaterOrEqual(t, widths.message, 80)
		assert.GreaterOrEqual(t, widths.path, 50)
	})

	t.Run("needs compression", func(t *testing.T) {
		widths := columnWidths{
			location: 20,
			severity: 10,
			rule:     25,
			category: 20,
		}

		// force compression with small available width
		calculateWithPathColumn(150, &widths, content)

		// path should compress first, then category, then rule, then message
		assert.GreaterOrEqual(t, widths.message, 40) // min message width
		assert.GreaterOrEqual(t, widths.path, 20)    // min path width
	})
}

func TestCalculateWithoutPathColumn(t *testing.T) {
	content := contentWidths{
		location: 20,
		rule:     25,
		category: 20,
	}

	t.Run("plenty of space", func(t *testing.T) {
		widths := columnWidths{
			location: 20,
			severity: 10,
			rule:     25,
			category: 20,
		}

		// 20 + 10 + 100 + 25 + 20 = 175, available = 200
		calculateWithoutPathColumn(200, &widths, content)

		assert.Greater(t, widths.message, 100) // message gets extra space
		assert.Equal(t, 0, widths.path)
	})

	t.Run("needs compression", func(t *testing.T) {
		widths := columnWidths{
			location: 20,
			severity: 10,
			rule:     25,
			category: 20,
		}

		// force compression
		calculateWithoutPathColumn(100, &widths, content)

		// category compresses first, then rule, then message
		assert.GreaterOrEqual(t, widths.message, 40) // min message width
	})
}
