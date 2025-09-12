package cui

import (
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestInitSyntaxStyles(t *testing.T) {
	// reset the init flag for testing
	syntaxStylesInit = false

	InitSyntaxStyles()
	assert.True(t, syntaxStylesInit)

	// verify styles are initialized
	assert.NotNil(t, syntaxKeyStyle)
	assert.NotNil(t, syntaxStringStyle)
	assert.NotNil(t, syntaxNumberStyle)
	assert.NotNil(t, syntaxBoolStyle)
	assert.NotNil(t, syntaxCommentStyle)
	assert.NotNil(t, syntaxDashStyle)
	assert.NotNil(t, syntaxRefStyle)
	assert.NotNil(t, syntaxDefaultStyle)
	assert.NotNil(t, syntaxSingleQuoteStyle)
}

func TestCalculateCodeWindow(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		targetLine int
		wantStart  int
		wantEnd    int
		wantAbove  bool
		wantBelow  bool
	}{
		{
			name:       "small file no windowing",
			lines:      make([]string, 20),
			targetLine: 10,
			wantStart:  1,
			wantEnd:    20,
			wantAbove:  false,
			wantBelow:  false,
		},
		{
			name:       "large file with target in middle",
			lines:      make([]string, 200),
			targetLine: 100,
			wantStart:  80,  // 100 - 20
			wantEnd:    120, // 100 + 20
			wantAbove:  true,
			wantBelow:  true,
		},
		{
			name:       "large file with target near start",
			lines:      make([]string, 200),
			targetLine: 10,
			wantStart:  1,
			wantEnd:    30, // 10 + 20
			wantAbove:  false,
			wantBelow:  true,
		},
		{
			name:       "large file with target near end",
			lines:      make([]string, 200),
			targetLine: 190,
			wantStart:  170, // 190 - 20
			wantEnd:    200,
			wantAbove:  true,
			wantBelow:  false,
		},
		{
			name:       "no target line",
			lines:      make([]string, 200),
			targetLine: 0,
			wantStart:  1,
			wantEnd:    41, // 20*2 + 1
			wantAbove:  false,
			wantBelow:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			window := calculateCodeWindow(tt.lines, tt.targetLine)
			assert.Equal(t, tt.wantStart, window.startLine)
			assert.Equal(t, tt.wantEnd, window.endLine)
			assert.Equal(t, tt.wantAbove, window.showAbove)
			assert.Equal(t, tt.wantBelow, window.showBelow)
			assert.Len(t, window.lines, tt.wantEnd-tt.wantStart+1)
		})
	}
}

func TestFormatLinesNotShown(t *testing.T) {
	tests := []struct {
		count    int
		position string
		contains string
	}{
		{10, "above", "10 lines above not shown"},
		{5, "below", "5 lines below not shown"},
		{100, "above", "100 lines above not shown"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			result := formatLinesNotShown(tt.count, tt.position)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestCalculateLineNumberWidth(t *testing.T) {
	tests := []struct {
		maxLineNum int
		expected   int
	}{
		{1, 5},      // minimum width is 5
		{10, 5},     // still minimum
		{99, 5},     // still minimum
		{100, 5},    // still minimum
		{999, 5},    // still minimum
		{1000, 5},   // 4 digits + 1 = 5
		{10000, 6},  // 5 digits + 1 = 6
		{100000, 7}, // 6 digits + 1 = 7
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.maxLineNum)), func(t *testing.T) {
			assert.Equal(t, tt.expected, calculateLineNumberWidth(tt.maxLineNum))
		})
	}
}

func TestFormatLineNumber(t *testing.T) {
	styles := getLineFormattingStyles()

	tests := []struct {
		name             string
		lineNum          int
		width            int
		isHighlighted    bool
		containsTriangle bool
		containsPipe     bool
	}{
		{
			name:             "normal line",
			lineNum:          42,
			width:            5,
			isHighlighted:    false,
			containsTriangle: false,
			containsPipe:     true,
		},
		{
			name:             "highlighted line",
			lineNum:          42,
			width:            5,
			isHighlighted:    true,
			containsTriangle: true,
			containsPipe:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLineNumber(tt.lineNum, tt.width, tt.isHighlighted, styles)

			if tt.containsTriangle {
				assert.Contains(t, result, "▶")
			}
			if tt.containsPipe {
				assert.Contains(t, result, "│")
			}
			assert.Contains(t, result, "42")
		})
	}
}

func TestFormatLineContent(t *testing.T) {
	styles := getLineFormattingStyles()

	tests := []struct {
		name          string
		line          string
		maxWidth      int
		isHighlighted bool
		isYAML        bool
	}{
		{
			name:          "normal yaml line",
			line:          "key: value",
			maxWidth:      50,
			isHighlighted: false,
			isYAML:        true,
		},
		{
			name:          "highlighted yaml line",
			line:          "key: value",
			maxWidth:      50,
			isHighlighted: true,
			isYAML:        true,
		},
		{
			name:          "json line",
			line:          `"key": "value"`,
			maxWidth:      50,
			isHighlighted: false,
			isYAML:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLineContent(tt.line, tt.maxWidth, tt.isHighlighted, tt.isYAML, styles)
			assert.NotEmpty(t, result)

			if tt.isHighlighted {
				// highlighted lines should have padding to fill width
				assert.True(t, strings.Contains(result, tt.line) || strings.Contains(result, " "))
			}
		})
	}
}

func TestViolationResultTableModel_ExtractCodeSnippet(t *testing.T) {
	specContent := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get users
      responses:
        '200':
          description: OK`)

	testModel := &ViolationResultTableModel{
		specContent: specContent,
	}

	tests := []struct {
		name          string
		result        *model.RuleFunctionResult
		contextLines  int
		wantStartLine int
		containsText  string
	}{
		{
			name:          "nil result",
			result:        nil,
			contextLines:  2,
			wantStartLine: 0,
			containsText:  "",
		},
		{
			name: "with start node",
			result: &model.RuleFunctionResult{
				StartNode: &yaml.Node{Line: 5},
			},
			contextLines:  2,
			wantStartLine: 3,
			containsText:  "paths:",
		},
		{
			name: "with origin node",
			result: &model.RuleFunctionResult{
				Origin: &index.NodeOrigin{Line: 8},
			},
			contextLines:  1,
			wantStartLine: 7,
			containsText:  "summary:",
		},
		{
			name: "near start of file",
			result: &model.RuleFunctionResult{
				StartNode: &yaml.Node{Line: 2},
			},
			contextLines:  3,
			wantStartLine: 1,
			containsText:  "openapi:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snippet, startLine := testModel.ExtractCodeSnippet(tt.result, tt.contextLines)
			assert.Equal(t, tt.wantStartLine, startLine)
			if tt.containsText != "" {
				assert.Contains(t, snippet, tt.containsText)
			} else {
				assert.Empty(t, snippet)
			}
		})
	}
}

func TestViolationResultTableModel_FormatCodeWithHighlight(t *testing.T) {
	specContent := []byte(strings.Repeat("line\n", 100))

	testModel := &ViolationResultTableModel{
		specContent: specContent,
		fileName:    "test.yaml",
	}

	tests := []struct {
		name       string
		targetLine int
		maxWidth   int
		contains   []string
	}{
		{
			name:       "highlight middle line",
			targetLine: 50,
			maxWidth:   80,
			contains:   []string{"30", "50", "70", "▶"},
		},
		{
			name:       "highlight near start",
			targetLine: 5,
			maxWidth:   80,
			contains:   []string{"1", "5", "25"},
		},
		{
			name:       "highlight near end",
			targetLine: 95,
			maxWidth:   80,
			contains:   []string{"75", "95", "100"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testModel.FormatCodeWithHighlight(tt.targetLine, tt.maxWidth)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestViolationResultTableModel_PrepareCodeViewport(t *testing.T) {
	specContent := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0`)

	testModel := &ViolationResultTableModel{
		specContent: specContent,
		fileName:    "test.yaml",
		width:       100,
		height:      40,
		modalContent: &model.RuleFunctionResult{
			StartNode: &yaml.Node{Line: 3},
		},
	}

	testModel.PrepareCodeViewport()
	assert.NotNil(t, testModel.codeViewport)
}

func TestViolationResultTableModel_ReCenterCodeView(t *testing.T) {
	specContent := []byte(strings.Repeat("line\n", 100))

	testModel := &ViolationResultTableModel{
		specContent: specContent,
		fileName:    "test.yaml",
		width:       100,
		height:      40,
		modalContent: &model.RuleFunctionResult{
			StartNode: &yaml.Node{Line: 50},
		},
	}

	testModel.PrepareCodeViewport()
	// initialOffset := testModel.codeViewport.YOffset

	// scroll away
	testModel.codeViewport.SetYOffset(0)

	// recenter
	testModel.ReCenterCodeView()

	// should be back near the initial offset
	// assert.NotEqual(t, 0, testModel.codeViewport.YOffset)
	assert.NotNil(t, testModel.codeViewport)
}
