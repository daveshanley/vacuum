package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/model"
)

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
//	::{level} file={path},line={n},col={n},title={rule-id}::{message}
//
// Severity mapping: error→error, warn→warning, info/hint→notice.
// URL and stdin inputs omit the file= property.
func RenderGitHubAnnotations(results []*model.RuleFunctionResult, fileName string) {
	relFile := toAnnotationFilePath(fileName)

	for _, r := range results {
		if r == nil {
			continue
		}

		level := severityToGitHubLevel(r.RuleSeverity)

		var props []string

		if relFile != "" {
			props = append(props, fmt.Sprintf("file=%s", escapeGitHubAnnotationProperty(relFile)))
		}

		if r.StartNode != nil && r.StartNode.Line > 0 {
			props = append(props, fmt.Sprintf("line=%d", r.StartNode.Line))
			if r.StartNode.Column > 0 {
				props = append(props, fmt.Sprintf("col=%d", r.StartNode.Column))
			}
			if r.EndNode != nil && r.EndNode.Line > 0 && r.EndNode.Line != r.StartNode.Line {
				props = append(props, fmt.Sprintf("endLine=%d", r.EndNode.Line))
			}
		}

		title := r.RuleId
		if title == "" && r.Rule != nil {
			title = r.Rule.Id
		}
		if title != "" {
			props = append(props, fmt.Sprintf("title=%s", escapeGitHubAnnotationProperty(title)))
		}

		message := escapeGitHubAnnotationMessage(r.Message)

		if len(props) > 0 {
			fmt.Printf("::%s %s::%s\n", level, strings.Join(props, ","), message)
		} else {
			fmt.Printf("::%s::%s\n", level, message)
		}
	}
}
