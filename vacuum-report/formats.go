package vacuum_report

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"text/tabwriter"

	htmlreport "github.com/daveshanley/vacuum/html-report"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/tui"
)

// Diagnostic is a portable lint finding. Ranges use zero-based, end-exclusive coordinates.
// Path segments retain numeric array indexes and literal mapping keys.
type Diagnostic struct {
	Code             string        `json:"code"`
	Path             []any         `json:"path"`
	Message          string        `json:"message"`
	Severity         int           `json:"severity"`
	Range            reports.Range `json:"range"`
	Source           string        `json:"source"`
	DocumentationURL string        `json:"documentationUrl,omitempty"`
}

// FormatOptions supplies report metadata and the JUnit failure threshold.
type FormatOptions struct {
	Version       string
	BaseDirectory string
	FailSeverity  int // -1 disables failures; 0..3 means error through hint.
}

// Formats returns the supported export names in a stable order.
func Formats() []string {
	return []string{"json", "stylish", "junit", "html", "text", "teamcity", "pretty", "github-actions", "sarif", "code-climate", "gitlab", "markdown"}
}

// SupportsFormat reports whether name is a supported report format.
func SupportsFormat(name string) bool {
	for _, f := range Formats() {
		if f == name {
			return true
		}
	}
	return false
}

// WriteFormat writes a complete report without changing the supplied diagnostics.
// All formats use the same finding data and propagate writer errors.
func WriteFormat(w io.Writer, format string, results []Diagnostic, opts FormatOptions) error {
	if results == nil {
		results = []Diagnostic{}
	}
	switch format {
	case "github-actions":
		for _, r := range results {
			level := "notice"
			if r.Severity == 0 {
				level = "error"
			} else if r.Severity == 1 {
				level = "warning"
			}
			source := relativeSource(r.Source, opts.BaseDirectory)
			if source == "stdin" || strings.Contains(source, "://") {
				source = ""
			}
			message := r.Message
			if r.DocumentationURL != "" {
				message += "\nDocumentation: " + r.DocumentationURL
			}
			start, end := r.Range.Start, r.Range.End
			start.Line++
			start.Char++
			end.Line++
			end.Char++
			if err := WriteGitHubAnnotation(w, level, r.Code, source, message, start, end); err != nil {
				return err
			}
		}
		return nil
	case "json":
		return writeJSON(w, results)
	case "sarif":
		return writeSARIF(w, results, opts)
	case "code-climate", "gitlab":
		return writeCodeClimate(w, results, opts)
	case "junit":
		return writeJUnit(w, results, opts)
	case "html":
		rows := make([]htmlreport.ExportRow, 0, len(results))
		for _, r := range results {
			rows = append(rows, htmlreport.ExportRow{Source: r.Source, Rule: r.Code, Message: r.Message, Severity: severityName(r.Severity), Path: pathText(r.Path), Line: r.Range.Start.Line + 1, Column: r.Range.Start.Char + 1})
		}
		return htmlreport.WriteExport(w, rows)
	case "markdown":
		rows := make([][]string, 0, len(results))
		for _, r := range results {
			row := []string{r.Code, pathText(r.Path), r.Message, severityName(r.Severity), fmt.Sprintf("%d:%d", r.Range.Start.Line+1, r.Range.Start.Char+1), r.Source}
			for i := range row {
				row[i] = markdownCell(row[i])
			}
			rows = append(rows, row)
		}
		_, err := io.WriteString(w, tui.RenderMarkdownTable([]string{"Code", "Path", "Message", "Severity", "Location", "Source"}, rows))
		return err
	case "text", "stylish", "pretty", "teamcity":
		return writeText(w, format, results)
	default:
		return fmt.Errorf("unsupported report format %q", format)
	}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
func severityName(s int) string {
	if s < 0 || s > 3 {
		return "warn"
	}
	return []string{"error", "warn", "info", "hint"}[s]
}
func pathText(p []any) string { b, _ := json.Marshal(p); return string(b) }
func markdownCell(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\\", "\\\\", "|", "\\|", "\r", "", "\n", "<br>", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]").Replace(s)
}
func singleLine(s string) string {
	return strings.NewReplacer("\r", "\\r", "\n", "\\n", "\t", "\\t", "\x1b", "\\x1b").Replace(s)
}
func escapeTeamCity(s string) string {
	return strings.NewReplacer("|", "||", "'", "|'", "\n", "|n", "\r", "|r", "[", "|[", "]", "|]", "\u0085", "|x", "\u2028", "|l", "\u2029", "|p").Replace(s)
}
func writeText(w io.Writer, format string, results []Diagnostic) error {
	if format == "teamcity" {
		for _, r := range results {
			if _, err := fmt.Fprintf(w, "##teamcity[inspectionType category='openapi' id='%s' name='%s' description='%s']\n##teamcity[inspection typeId='%s' file='%s' line='%d' message='%s']\n", escapeTeamCity(r.Code), escapeTeamCity(r.Code), escapeTeamCity(r.Message), escapeTeamCity(r.Code), escapeTeamCity(r.Source), r.Range.Start.Line+1, escapeTeamCity(severityName(r.Severity)+" -- "+r.Message)); err != nil {
				return err
			}
		}
		return nil
	}
	if format == "text" {
		for _, r := range results {
			if _, err := fmt.Fprintf(w, "%s:%d:%d %s %s %q\n", singleLine(r.Source), r.Range.Start.Line+1, r.Range.Start.Char+1, severityName(r.Severity), singleLine(r.Code), r.Message); err != nil {
				return err
			}
		}
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	last := ""
	for _, r := range results {
		if last != r.Source {
			if _, err := fmt.Fprintf(tw, "\n%s\n", singleLine(r.Source)); err != nil {
				return err
			}
			last = r.Source
		}
		if _, err := fmt.Fprintf(tw, "  %d:%d\t%s\t%s\t%s\t%s\n", r.Range.Start.Line+1, r.Range.Start.Char+1, severityName(r.Severity), singleLine(r.Code), singleLine(r.Message), singleLine(pathText(r.Path))); err != nil {
			return err
		}
		if format == "pretty" {
			if _, err := fmt.Fprintln(tw); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(tw, "\n%d findings\n", len(results)); err != nil {
		return err
	}
	return tw.Flush()
}

func relativeSource(source, base string) string {
	if strings.Contains(source, "://") || source == "stdin" {
		return source
	}
	if filepath.IsAbs(source) && base != "" {
		if rel, err := filepath.Rel(base, source); err == nil {
			source = rel
		}
	}
	return filepath.ToSlash(source)
}
func sourceURI(source, base string) string {
	s := relativeSource(source, base)
	if strings.Contains(s, "://") {
		return s
	}
	return (&url.URL{Path: s}).String()
}

type qualityLocation struct {
	Path  string       `json:"path"`
	Lines qualityLines `json:"lines"`
}
type qualityLines struct {
	Begin int `json:"begin"`
	End   int `json:"end"`
}
type qualityIssue struct {
	Type        string          `json:"type"`
	CheckName   string          `json:"check_name"`
	Description string          `json:"description"`
	Categories  []string        `json:"categories"`
	Location    qualityLocation `json:"location"`
	Severity    string          `json:"severity"`
	Fingerprint string          `json:"fingerprint"`
}

func writeCodeClimate(w io.Writer, results []Diagnostic, opts FormatOptions) error {
	issues := make([]qualityIssue, 0, len(results))
	for _, r := range results {
		source := relativeSource(r.Source, opts.BaseDirectory)
		key, _ := json.Marshal([]any{source, r.Path, r.Code, r.Message})
		hash := sha256.Sum256(key)
		severity := "major"
		if r.Severity >= 0 && r.Severity < 4 {
			severity = []string{"critical", "major", "minor", "info"}[r.Severity]
		}
		issues = append(issues, qualityIssue{"issue", r.Code, r.Message, []string{"Style"}, qualityLocation{source, qualityLines{r.Range.Start.Line + 1, r.Range.End.Line + 1}}, severity, hex.EncodeToString(hash[:])})
	}
	return writeJSON(w, issues)
}
func writeJUnit(w io.Writer, results []Diagnostic, opts FormatOptions) error {
	suites := TestSuites{}
	bySource := map[string]*TestSuite{}
	for _, r := range results {
		s := bySource[r.Source]
		if s == nil {
			s = &TestSuite{Name: r.Source}
			bySource[r.Source] = s
			suites.TestSuites = append(suites.TestSuites, s)
		}
		tc := &TestCase{Name: r.Code + " " + pathText(r.Path), ClassName: r.Code, File: r.Source, Line: r.Range.Start.Line + 1}
		if r.Severity <= opts.FailSeverity {
			tc.Failure = &Failure{Message: r.Message, Type: severityName(r.Severity), File: r.Source}
			s.Failures++
			suites.Failures++
		}
		s.TestCases = append(s.TestCases, tc)
		s.Tests++
		suites.Tests++
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(suites)
}

type sarifDocument struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}
type sarifRun struct {
	ColumnKind string        `json:"columnKind"`
	Tool       sarifTool     `json:"tool"`
	Results    []sarifResult `json:"results"`
}
type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}
type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}
type sarifRule struct {
	ID      string `json:"id"`
	HelpURI string `json:"helpUri,omitempty"`
}
type sarifMessage struct {
	Text string `json:"text"`
}
type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	RuleIndex int             `json:"ruleIndex"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}
type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}
type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           sarifRegion   `json:"region"`
}
type sarifArtifact struct {
	URI string `json:"uri"`
}
type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
}

func writeSARIF(w io.Writer, results []Diagnostic, opts FormatOptions) error {
	driver := sarifDriver{Name: "vacuum", Version: opts.Version, InformationURI: "https://quobix.com/vacuum/", Rules: []sarifRule{}}
	items := make([]sarifResult, 0, len(results))
	indexes := map[string]int{}
	for _, r := range results {
		i, ok := indexes[r.Code]
		if !ok {
			i = len(driver.Rules)
			indexes[r.Code] = i
			driver.Rules = append(driver.Rules, sarifRule{r.Code, r.DocumentationURL})
		}
		level := "note"
		if r.Severity == 0 {
			level = "error"
		} else if r.Severity == 1 {
			level = "warning"
		}
		items = append(items, sarifResult{r.Code, i, level, sarifMessage{r.Message}, []sarifLocation{{sarifPhysical{sarifArtifact{sourceURI(r.Source, opts.BaseDirectory)}, sarifRegion{r.Range.Start.Line + 1, r.Range.Start.Char + 1, r.Range.End.Line + 1, r.Range.End.Char + 1}}}}})
	}
	return writeJSON(w, sarifDocument{"https://json.schemastore.org/sarif-2.1.0.json", "2.1.0", []sarifRun{{"unicodeCodePoints", sarifTool{driver}, items}}})
}
