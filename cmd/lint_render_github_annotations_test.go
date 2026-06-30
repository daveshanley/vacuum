package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"go.yaml.in/yaml/v4"
)

// captureAnnotationsOutput captures stdout while calling fn, returning what was printed.
func captureAnnotationsOutput(t *testing.T, fn func()) string {
	t.Helper()
	stdout, _ := captureOSStreams(t, fn)
	return stdout
}

func TestRenderGitHubAnnotations_Empty(t *testing.T) {
	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(nil, "spec.yaml")
	})
	assert.Empty(t, out)
}

func TestRenderGitHubAnnotations_SkipsNilEntries(t *testing.T) {
	results := []*model.RuleFunctionResult{
		nil,
		{
			RuleSeverity: model.SeverityError,
			RuleId:       "rule-error",
			Message:      "an error",
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "stdin")
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "::error ")
	assert.Contains(t, lines[0], "file=")
	assert.Contains(t, lines[0], "title=rule-error")
	assert.Contains(t, lines[0], "::an error")
}

func TestRenderGitHubAnnotations_SeverityMapping(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityError,
			RuleId:       "rule-error",
			Message:      "an error",
			StartNode:    &yaml.Node{Line: 10, Column: 3},
		},
		{
			RuleSeverity: model.SeverityWarn,
			RuleId:       "rule-warn",
			Message:      "a warning",
			StartNode:    &yaml.Node{Line: 20, Column: 1},
		},
		{
			RuleSeverity: model.SeverityInfo,
			RuleId:       "rule-info",
			Message:      "an info",
			StartNode:    &yaml.Node{Line: 30, Column: 5},
		},
		{
			RuleSeverity: "hint",
			RuleId:       "rule-hint",
			Message:      "a hint",
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "spec.yaml")
	})

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 4)
	assert.Contains(t, lines[0], "::error ")
	assert.Contains(t, lines[1], "::warning ")
	assert.Contains(t, lines[2], "::notice ")
	assert.Contains(t, lines[3], "::notice ")
}

func TestRenderGitHubAnnotations_LineColTitle(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityError,
			RuleId:       "my-rule",
			Message:      "bad thing",
			StartNode:    &yaml.Node{Line: 42, Column: 7},
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "spec.yaml")
	})

	assert.Equal(t, "::error title=my-rule,file=spec.yaml,col=7,endColumn=7,line=42,endLine=42::bad thing", out)
}

func TestRenderGitHubAnnotations_UsesRuleIDFallback(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityWarn,
			Rule:         &model.Rule{Id: "fallback-rule"},
			Message:      "fallback title",
			StartNode:    &yaml.Node{Line: 8, Column: 2},
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "spec.yaml")
	})

	assert.Contains(t, out, "title=fallback-rule")
}

func TestRenderGitHubAnnotations_EndRange(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityWarn,
			RuleId:       "span-rule",
			Message:      "spans lines",
			StartNode:    &yaml.Node{Line: 5, Column: 1},
			EndNode:      &yaml.Node{Line: 10, Column: 4},
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "spec.yaml")
	})

	assert.Contains(t, out, "endLine=10")
	assert.Contains(t, out, "endColumn=4")
}

func TestRenderGitHubAnnotations_URLInput_NoFileProperty(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityError,
			RuleId:       "rule-x",
			Message:      "msg",
			StartNode:    &yaml.Node{Line: 1, Column: 1},
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "https://example.com/spec.yaml")
	})

	assert.Contains(t, out, "file=")
	assert.Contains(t, out, "::error ")
}

func TestRenderGitHubAnnotations_StdinInput_NoFileProperty(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityError,
			RuleId:       "rule-x",
			Message:      "msg",
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "stdin")
	})

	assert.Contains(t, out, "file=")
}

func TestRenderGitHubAnnotations_NoPropertiesPrintsBareAnnotation(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityInfo,
			Message:      "bare message",
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "stdin")
	})

	assert.Equal(t, "::notice file=::bare message", out)
}

func TestRenderGitHubAnnotations_UsesRangeWhenAvailable(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityError,
			RuleId:       "range-rule",
			Message:      "range message",
			Range: reports.Range{
				Start: reports.RangeItem{Line: 3, Char: 6},
				End:   reports.RangeItem{Line: 4, Char: 12},
			},
			StartNode: &yaml.Node{Line: 99, Column: 99},
			EndNode:   &yaml.Node{Line: 100, Column: 100},
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "spec.yaml")
	})

	assert.Equal(t, "::error title=range-rule,file=spec.yaml,col=6,endColumn=12,line=3,endLine=4::range message", out)
}

func TestRenderGitHubAnnotations_UsesOriginAbsoluteLocation(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityWarn,
			RuleId:       "origin-rule",
			Message:      "origin message",
			Range: reports.Range{
				Start: reports.RangeItem{Line: 7, Char: 2},
				End:   reports.RangeItem{Line: 7, Char: 9},
			},
			Origin: &index.NodeOrigin{AbsoluteLocation: filepathForTest(t, "nested/spec.yaml")},
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "spec.yaml")
	})

	assert.Contains(t, out, "file=nested/spec.yaml")
}

func TestRenderGitHubAnnotations_AppendsDocumentationURL(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityWarn,
			RuleId:       "doc-rule",
			Message:      "doc message",
			Rule: &model.Rule{
				Id:               "doc-rule",
				DocumentationURL: "https://example.com/rule-docs",
			},
			Range: reports.Range{
				Start: reports.RangeItem{Line: 2, Char: 3},
				End:   reports.RangeItem{Line: 2, Char: 8},
			},
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "spec.yaml")
	})

	assert.Equal(t, "::warning title=doc-rule,file=spec.yaml,col=3,endColumn=8,line=2,endLine=2::doc message%0ADocumentation: https://example.com/rule-docs", out)
}

func TestRenderGitHubAnnotations_SpectralRangesIsOptional(t *testing.T) {
	var doc yaml.Node
	err := yaml.Unmarshal([]byte(`
paths:
  /widgets:
    get:
      responses:
        '200':
          description: OK
`), &doc)
	assert.NoError(t, err)

	root := doc.Content[0]
	paths := root.Content[1]
	pathItem := paths.Content[1]
	getOp := pathItem.Content[1]

	results := []*model.RuleFunctionResult{{
		RuleSeverity: model.SeverityWarn,
		RuleId:       "optional-range-rule",
		Message:      "operationId must be present",
		StartNode:    getOp,
		EndNode:      getOp,
	}}

	defaultOut := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotationsWithOptions(results, "spec.yaml", GitHubAnnotationRenderOptions{SpectralRanges: false})
	})
	assert.Contains(t, defaultOut, fmt.Sprintf("line=%d,endLine=%d", getOp.Line, getOp.Line))

	expandedOut := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotationsWithOptions(results, "spec.yaml", GitHubAnnotationRenderOptions{SpectralRanges: true})
	})
	assert.Contains(t, expandedOut, fmt.Sprintf("line=%d", getOp.Line))

	re := regexp.MustCompile(`endLine=(\d+)`)
	parts := re.FindStringSubmatch(expandedOut)
	if assert.Len(t, parts, 2) {
		endLine, convErr := strconv.Atoi(parts[1])
		assert.NoError(t, convErr)
		assert.Greater(t, endLine, getOp.Line)
	}
}

func TestRenderGitHubAnnotations_MessageEscaping(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityError,
			RuleId:       "rule-x",
			Message:      "line one\nline two",
			StartNode:    &yaml.Node{Line: 1, Column: 1},
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "spec.yaml")
	})

	assert.Contains(t, out, "%0A")
	assert.NotContains(t, out, "\nline two")
}

func TestEscapeGitHubAnnotationProperty(t *testing.T) {
	assert.Equal(t, "100%25done", escapeGitHubAnnotationProperty("100%done"))
	assert.Equal(t, "a%3Ab", escapeGitHubAnnotationProperty("a:b"))
	assert.Equal(t, "a%2Cb", escapeGitHubAnnotationProperty("a,b"))
	assert.Equal(t, "a%0Ab", escapeGitHubAnnotationProperty("a\nb"))
	assert.Equal(t, "a%0Db", escapeGitHubAnnotationProperty("a\rb"))
}

func TestEscapeGitHubAnnotationMessage(t *testing.T) {
	assert.Equal(t, "100%25done", escapeGitHubAnnotationMessage("100%done"))
	assert.Equal(t, "a%0Ab", escapeGitHubAnnotationMessage("a\nb"))
	assert.Equal(t, "a%0Db", escapeGitHubAnnotationMessage("a\rb"))
	// colons and commas are allowed unescaped in the message
	assert.Equal(t, "a:b", escapeGitHubAnnotationMessage("a:b"))
	assert.Equal(t, "a,b", escapeGitHubAnnotationMessage("a,b"))
}

func filepathForTest(t *testing.T, rel string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return fmt.Sprintf("%s/%s", wd, rel)
}
