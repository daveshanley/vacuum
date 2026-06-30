package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"go.yaml.in/yaml/v4"
)

type GitHubAnnotationRenderOptions struct {
	SpectralRanges bool
}

type cachedNodeSpan struct {
	start reports.RangeItem
	end   reports.RangeItem
}

// severityToGitHubLevel maps a vacuum severity string to a GitHub Actions
// workflow command annotation level.
func severityToGitHubLevel(severity string) string {
	switch severity {
	case model.SeverityError:
		return "error"
	case model.SeverityWarn:
		return "warning"
	default: // info, hint, and anything else
		return "notice"
	}
}

// escapeGitHubAnnotationProperty escapes special characters that are not
// allowed inside a GitHub Actions workflow command property value
// (the key=value section before the final "::" separator).
func escapeGitHubAnnotationProperty(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	s = strings.ReplaceAll(s, "\n", "%0A")
	s = strings.ReplaceAll(s, ":", "%3A")
	s = strings.ReplaceAll(s, ",", "%2C")
	return s
}

// escapeGitHubAnnotationMessage escapes special characters that are not
// allowed inside a GitHub Actions workflow command message value
// (the section after the final "::" separator).
func escapeGitHubAnnotationMessage(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	s = strings.ReplaceAll(s, "\n", "%0A")
	return s
}

// toAnnotationFilePath converts a file path to a workspace-relative form
// suitable for the "file=" property of a GitHub Actions annotation.
//
// URL inputs and stdin are returned as empty strings so the caller can omit
// the property; absolute paths are made relative to the current working
// directory. If any operation fails the original value is returned unchanged.
func toAnnotationFilePath(filePath string) string {
	if filePath == "" || filePath == "stdin" {
		return ""
	}
	if strings.Contains(filePath, "://") {
		return ""
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return filePath
	}

	cwd, err := os.Getwd()
	if err != nil {
		return filePath
	}

	rel, err := filepath.Rel(cwd, absPath)
	if err != nil {
		return filePath
	}
	return rel
}

// RenderGitHubAnnotations writes GitHub Actions workflow command annotation
// lines to stdout for each result in results.
//
// Format per violation:
//
//	::{level} title={rule-id},file={path},col={n},endColumn={n},line={n},endLine={n}::{message}
//
// Severity mapping: error→error, warn→warning, info/hint→notice.
func RenderGitHubAnnotations(results []*model.RuleFunctionResult, fileName string) {
	RenderGitHubAnnotationsWithOptions(results, fileName, GitHubAnnotationRenderOptions{SpectralRanges: true})
}

func RenderGitHubAnnotationError(err error, fileName string) {
	if err == nil {
		return
	}

	RenderGitHubAnnotations([]*model.RuleFunctionResult{{
		RuleSeverity: model.SeverityError,
		Message:      err.Error(),
	}}, fileName)
}

func RenderGitHubAnnotationsWithOptions(
	results []*model.RuleFunctionResult,
	fileName string,
	options GitHubAnnotationRenderOptions,
) {
	var output strings.Builder
	printed := 0
	spanCache := make(map[*yaml.Node]cachedNodeSpan)

	for _, r := range results {
		if r == nil {
			continue
		}

		level := severityToGitHubLevel(r.RuleSeverity)

		var props []string

		title := r.RuleId
		if title == "" && r.Rule != nil {
			title = r.Rule.Id
		}
		if title != "" {
			props = append(props, fmt.Sprintf("title=%s", escapeGitHubAnnotationProperty(title)))
		}

		sourcePath := fileName
		if r.Origin != nil && r.Origin.AbsoluteLocation != "" {
			sourcePath = r.Origin.AbsoluteLocation
		}
		relFile := toAnnotationFilePath(sourcePath)
		props = append(props, fmt.Sprintf("file=%s", escapeGitHubAnnotationProperty(relFile)))

		start, end := annotationRange(r, options, spanCache)
		if start.Char > 0 {
			props = append(props, fmt.Sprintf("col=%d", start.Char))
		}
		if end.Char > 0 {
			props = append(props, fmt.Sprintf("endColumn=%d", end.Char))
		}
		if start.Line > 0 {
			props = append(props, fmt.Sprintf("line=%d", start.Line))
		}
		if end.Line > 0 {
			props = append(props, fmt.Sprintf("endLine=%d", end.Line))
		}

		message := escapeGitHubAnnotationMessage(r.Message)
		if r.Rule != nil && r.Rule.DocumentationURL != "" {
			message += "%0ADocumentation: " + escapeGitHubAnnotationMessage(r.Rule.DocumentationURL)
		}

		if printed > 0 {
			output.WriteByte('\n')
		}
		if len(props) > 0 {
			output.WriteString(fmt.Sprintf("::%s %s::%s", level, strings.Join(props, ","), message))
		} else {
			output.WriteString(fmt.Sprintf("::%s::%s", level, message))
		}
		printed++
	}

	if printed > 0 {
		output.WriteByte('\n')
		fmt.Print(output.String())
	}
}

func annotationRange(
	result *model.RuleFunctionResult,
	options GitHubAnnotationRenderOptions,
	spanCache map[*yaml.Node]cachedNodeSpan,
) (reports.RangeItem, reports.RangeItem) {
	if result == nil {
		return reports.RangeItem{}, reports.RangeItem{}
	}

	start := result.Range.Start
	end := result.Range.End
	if options.SpectralRanges && shouldExpandSpectralRange(result, start, end) {
		if result.StartNode != nil {
			s, e := nodeSpan(result.StartNode, spanCache)
			start = earlierRangeItem(start, s)
			end = laterRangeItem(end, e)
		}
		if result.EndNode != nil {
			s, e := nodeSpan(result.EndNode, spanCache)
			start = earlierRangeItem(start, s)
			end = laterRangeItem(end, e)
		}
	}

	if start.Line > 0 || start.Char > 0 || end.Line > 0 || end.Char > 0 {
		if end.Line == 0 {
			end.Line = start.Line
		}
		if end.Char == 0 {
			end.Char = start.Char
		}
		return start, end
	}

	if result.StartNode != nil {
		start = reports.RangeItem{Line: result.StartNode.Line, Char: result.StartNode.Column}
	}
	if result.EndNode != nil {
		end = reports.RangeItem{Line: result.EndNode.Line, Char: result.EndNode.Column}
	}
	if end.Line == 0 {
		end.Line = start.Line
	}
	if end.Char == 0 {
		end.Char = start.Char
	}

	return start, end
}

func nodeSpan(node *yaml.Node, spanCache map[*yaml.Node]cachedNodeSpan) (reports.RangeItem, reports.RangeItem) {
	if node == nil {
		return reports.RangeItem{}, reports.RangeItem{}
	}
	if cached, ok := spanCache[node]; ok {
		return cached.start, cached.end
	}

	start := reports.RangeItem{Line: node.Line, Char: node.Column}
	end := scalarNodeEnd(node)

	for _, child := range node.Content {
		cStart, cEnd := nodeSpan(child, spanCache)
		start = earlierRangeItem(start, cStart)
		end = laterRangeItem(end, cEnd)
	}

	if end.Line == 0 {
		end.Line = start.Line
	}
	if end.Char == 0 {
		end.Char = start.Char
	}
	spanCache[node] = cachedNodeSpan{start: start, end: end}

	return start, end
}

func scalarNodeEnd(node *yaml.Node) reports.RangeItem {
	if node == nil {
		return reports.RangeItem{}
	}

	modifier := 0
	if node.Style == yaml.DoubleQuotedStyle || node.Style == yaml.SingleQuotedStyle {
		modifier = 2
	}

	return reports.RangeItem{
		Line: node.Line,
		Char: node.Column + len(node.Value) + modifier,
	}
}

func earlierRangeItem(current reports.RangeItem, candidate reports.RangeItem) reports.RangeItem {
	if candidate.Line <= 0 {
		return current
	}
	if current.Line <= 0 {
		return candidate
	}
	if candidate.Line < current.Line {
		return candidate
	}
	if candidate.Line == current.Line && candidate.Char > 0 && (current.Char <= 0 || candidate.Char < current.Char) {
		return candidate
	}
	return current
}

func laterRangeItem(current reports.RangeItem, candidate reports.RangeItem) reports.RangeItem {
	if candidate.Line <= 0 {
		return current
	}
	if current.Line <= 0 {
		return candidate
	}
	if candidate.Line > current.Line {
		return candidate
	}
	if candidate.Line == current.Line && candidate.Char > current.Char {
		return candidate
	}
	return current
}

func shouldExpandSpectralRange(result *model.RuleFunctionResult, start, end reports.RangeItem) bool {
	if result == nil || result.StartNode == nil {
		return false
	}
	msg := strings.ToLower(result.Message)
	if !strings.Contains(msg, "must be present") && !strings.Contains(msg, "must be set") {
		return false
	}
	if start.Line <= 0 {
		return true
	}
	if end.Line <= 0 {
		return true
	}
	return start.Line == end.Line
}
