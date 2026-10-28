package vacuum_report

import (
	"fmt"
	"io"
	"strings"

	"github.com/daveshanley/vacuum/model/reports"
)

var escapeGitHubPropertyReplacer = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")

// EscapeGitHubProperty escapes a workflow command property.
func EscapeGitHubProperty(s string) string {
	return escapeGitHubPropertyReplacer.Replace(s)
}

var escapeGitHubMessageReplacer = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")

// EscapeGitHubMessage escapes a workflow command message.
func EscapeGitHubMessage(s string) string {
	return escapeGitHubMessageReplacer.Replace(s)
}

// WriteGitHubAnnotation writes one annotation using one-based coordinates.
// Empty file/title and zero coordinates are omitted. Message is unescaped input.
func WriteGitHubAnnotation(w io.Writer, level, title, file, message string, start, end reports.RangeItem) error {
	props := make([]string, 0, 6)
	if title != "" {
		props = append(props, "title="+EscapeGitHubProperty(title))
	}
	if file != "" {
		props = append(props, "file="+EscapeGitHubProperty(file))
	}
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
	prefix := ""
	if len(props) > 0 {
		prefix = " " + strings.Join(props, ",")
	}
	_, err := fmt.Fprintf(w, "::%s%s::%s\n", level, prefix, EscapeGitHubMessage(message))
	return err
}
