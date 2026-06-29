package cmd

import (
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
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

	assert.Contains(t, out, "line=42")
	assert.Contains(t, out, "col=7")
	assert.Contains(t, out, "title=my-rule")
	assert.Contains(t, out, "::bad thing")
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

func TestRenderGitHubAnnotations_EndLine(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{
			RuleSeverity: model.SeverityWarn,
			RuleId:       "span-rule",
			Message:      "spans lines",
			StartNode:    &yaml.Node{Line: 5, Column: 1},
			EndNode:      &yaml.Node{Line: 10, Column: 1},
		},
	}

	out := captureAnnotationsOutput(t, func() {
		RenderGitHubAnnotations(results, "spec.yaml")
	})

	assert.Contains(t, out, "endLine=10")
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

	assert.NotContains(t, out, "file=")
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

	assert.NotContains(t, out, "file=")
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

	assert.Equal(t, "::notice::bare message\n", out)
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
