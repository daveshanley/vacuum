package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	vacuumreport "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"github.com/pb33f/libopenapi/index"
	"github.com/spf13/cobra"
)

type lintOutputTarget struct{ format, path string }
type lintOutputFile struct {
	source  string
	results []*model.RuleFunctionResult
}

func addLintOutputFlags(cmd *cobra.Command) {
	cmd.Flags().StringSlice("format", nil, "Report formats (repeatable): "+strings.Join(vacuumreport.Formats(), ", "))
	cmd.Flags().StringArrayP("output", "o", nil, "Report destination; use format=path for multiple formats (default: stdout)")
}
func lintOutputTargets(cmd *cobra.Command) ([]lintOutputTarget, error) {
	formats, _ := cmd.Flags().GetStringSlice("format")
	outputs, _ := cmd.Flags().GetStringArray("output")
	if len(formats) == 0 {
		if len(outputs) > 0 {
			return nil, fmt.Errorf("--output requires --format")
		}
		return nil, nil
	}
	for _, flag := range []string{"pipeline-output", "github-annotations", "details", "snippets", "silent", "show-rules", "time", "no-message", "all-results", "no-clip"} {
		if boolFlagValue(cmd, flag) {
			return nil, fmt.Errorf("--format cannot be combined with --%s", flag)
		}
	}
	targets := make([]lintOutputTarget, 0, len(formats))
	indices := map[string]int{}
	for _, format := range formats {
		if !vacuumreport.SupportsFormat(format) {
			return nil, fmt.Errorf("unknown report format %q; choose from %s", format, strings.Join(vacuumreport.Formats(), ", "))
		}
		if _, ok := indices[format]; ok {
			return nil, fmt.Errorf("duplicate report format %q", format)
		}
		indices[format] = len(targets)
		targets = append(targets, lintOutputTarget{format: format})
	}
	assigned := map[string]bool{}
	for _, output := range outputs {
		format, path, ok := strings.Cut(output, "=")
		if !ok {
			if len(targets) != 1 {
				return nil, fmt.Errorf("multiple formats require --output format=path")
			}
			format = targets[0].format
			path = output
		}
		i, ok := indices[format]
		if !ok {
			return nil, fmt.Errorf("output format %q was not selected", format)
		}
		if assigned[format] || path == "" {
			return nil, fmt.Errorf("empty or duplicate output for %s", format)
		}
		assigned[format] = true
		targets[i].path = path
	}
	stdout := 0
	paths := map[string]bool{}
	for _, t := range targets {
		if t.path == "" || t.path == "-" {
			stdout++
			continue
		}
		abs, err := reportDestinationPath(t.path)
		if err != nil {
			return nil, err
		}
		if paths[abs] {
			return nil, fmt.Errorf("report destinations must be distinct: %s", t.path)
		}
		paths[abs] = true
	}
	if stdout > 1 {
		return nil, fmt.Errorf("only one format can use stdout; assign the other formats with --output format=path")
	}
	return targets, nil
}
func validateLintOutputInputs(targets []lintOutputTarget, inputs []string) error {
	for _, t := range targets {
		if t.path == "" || t.path == "-" {
			continue
		}
		out, _ := reportDestinationPath(t.path)
		outInfo, _ := os.Stat(out)
		for _, input := range inputs {
			if input == "" || strings.Contains(input, "://") {
				continue
			}
			in, _ := reportDestinationPath(input)
			inInfo, _ := os.Stat(in)
			if out == in || outInfo != nil && inInfo != nil && os.SameFile(outInfo, inInfo) {
				return fmt.Errorf("report destination would overwrite input %s", input)
			}
		}
	}
	return nil
}
func writeLintOutputs(cmd *cobra.Command, files []lintOutputFile, flags *LintFlags) error {
	targets, err := lintOutputTargets(cmd)
	if err != nil {
		return err
	}
	needRecords := false
	for _, target := range targets {
		if target.format != "github-actions" {
			needRecords = true
		}
	}
	records := make([]vacuumreport.Diagnostic, 0)
	spanCache := make(map[*yaml.Node]cachedNodeSpan)
	for _, file := range files {
		if !needRecords {
			break
		}
		for _, result := range file.results {
			if result == nil {
				continue
			}
			if flags.ErrorsFlag && diagnosticSeverity(result) != 0 {
				continue
			}
			records = append(records, exportDiagnostic(result, file.source, spanCache))
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		a, b := records[i], records[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Range.Start.Line != b.Range.Start.Line {
			return a.Range.Start.Line < b.Range.Start.Line
		}
		if a.Range.Start.Char != b.Range.Start.Char {
			return a.Range.Start.Char < b.Range.Start.Char
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
	base, _ := os.Getwd()
	threshold := 0
	switch flags.FailSeverityFlag {
	case "warn":
		threshold = 1
	case "info":
		threshold = 2
	case "hint":
		threshold = 3
	case "none":
		threshold = -1
	}
	opts := vacuumreport.FormatOptions{Version: GetVersion(), BaseDirectory: base, FailSeverity: threshold}
	for _, target := range targets {
		render := func(w io.Writer) error {
			if target.format == "github-actions" {
				for _, file := range files {
					results := file.results
					if flags.ErrorsFlag {
						results = nil
						for _, r := range file.results {
							if r != nil && diagnosticSeverity(r) == 0 {
								results = append(results, r)
							}
						}
					}
					if err := writeGitHubAnnotations(w, results, file.source, GitHubAnnotationRenderOptions{SpectralRanges: true}); err != nil {
						return err
					}
				}
				return nil
			}
			return vacuumreport.WriteFormat(w, target.format, records, opts)
		}
		if target.path == "" || target.path == "-" {
			err = render(cmd.OutOrStdout())
		} else {
			err = writeReportFile(target.path, render)
		}
		if err != nil {
			return fmt.Errorf("write %s report: %w", target.format, err)
		}
	}
	return nil
}
func writeReportFile(path string, render func(io.Writer) error) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".vacuum-report-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = render(f); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
func diagnosticSeverity(r *model.RuleFunctionResult) int {
	s := r.RuleSeverity
	if s == "" && r.Rule != nil {
		s = r.Rule.Severity
	}
	switch s {
	case model.SeverityError:
		return 0
	case model.SeverityInfo:
		return 2
	case model.SeverityHint:
		return 3
	}
	return 1
}
func exportDiagnostic(r *model.RuleFunctionResult, source string, cache map[*yaml.Node]cachedNodeSpan) vacuumreport.Diagnostic {
	code := r.RuleId
	doc := ""
	if r.Rule != nil {
		if code == "" {
			code = r.Rule.Id
		}
		doc = r.Rule.DocumentationURL
	}
	if r.Origin != nil && r.Origin.AbsoluteLocation != "" {
		source = r.Origin.AbsoluteLocation
	}
	var start, end reports.RangeItem
	if r.Origin != nil && r.Origin.Line > 0 {
		start = reports.RangeItem{Line: r.Origin.Line, Char: r.Origin.Column}
		end = start
	} else {
		start, end = annotationRange(r, GitHubAnnotationRenderOptions{SpectralRanges: true}, cache)
	}
	start.Line = max(0, start.Line-1)
	start.Char = max(0, start.Char-1)
	end.Line = max(0, end.Line-1)
	end.Char = max(0, end.Char-1)
	if end.Line < start.Line || end.Line == start.Line && end.Char < start.Char {
		end = start
	}
	path := []any{}
	if p, err := jsonpath.NewPath(r.Path); err == nil && p.IsSingular() {
		if segments, err := p.GetSegmentInfo(); err == nil {
			for _, s := range segments {
				if s.Kind == jsonpath.SegmentKindArrayIndex {
					path = append(path, s.Index)
				} else {
					path = append(path, s.Key)
				}
			}
		}
	} else if r.Path != "" && r.Path != "$" {
		path = append(path, r.Path)
	}
	return vacuumreport.Diagnostic{Code: code, Path: path, Message: r.Message, Severity: diagnosticSeverity(r), Range: reports.Range{Start: start, End: end}, Source: source, DocumentationURL: doc}
}

// Resolve the existing parent so aliases of one directory cannot overwrite each other's reports.
func reportDestinationPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return absolute, nil
	}
	return filepath.Join(parent, filepath.Base(absolute)), nil
}

// indexedReportInputs includes loaded external documents even when they produced no findings.
func indexedReportInputs(idx *index.SpecIndex) []string {
	if idx == nil {
		return nil
	}
	paths := []string{idx.GetSpecAbsolutePath()}
	if rolodex := idx.GetRolodex(); rolodex != nil {
		for _, external := range rolodex.GetIndexes() {
			if external != nil {
				paths = append(paths, external.GetSpecAbsolutePath())
			}
		}
	}
	return paths
}
