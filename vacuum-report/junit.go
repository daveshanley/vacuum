// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package vacuum_report

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"strings"
	"text/template"
	"time"
)

type TestSuites struct {
	XMLName    xml.Name     `xml:"testsuites"`
	TestSuites []*TestSuite `xml:"testsuite"`
	Tests      int          `xml:"tests,attr"`
	Failures   int          `xml:"failures,attr"`
	Time       float64      `xml:"time,attr"`
}

type TestSuite struct {
	XMLName   xml.Name    `xml:"testsuite"`
	Name      string      `xml:"name,attr"`
	Tests     int         `xml:"tests,attr"`
	Failures  int         `xml:"failures,attr"`
	TestCases []*TestCase `xml:"testcase"`
}

type Properties struct {
	xml.Name   `xml:"properties"`
	Properties []*Property `xml:"property"`
}

type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type TestCase struct {
	Name       string      `xml:"name,attr"`
	ClassName  string      `xml:"classname,attr"`
	Line       int         `xml:"line,attr,omitempty"`
	Failure    *Failure    `xml:"failure,omitempty"`
	Properties *Properties `xml:"properties,omitempty"`
	File       string      `xml:"file,attr,omitempty"`
}

type Failure struct {
	Message  string `xml:"message,attr,omitempty"`
	Type     string `xml:"type,attr,omitempty"`
	File     string `xml:"file,attr,omitempty"`
	Contents string `xml:",innerxml"`
}

// JUnitConfig controls how JUnit reports are generated
type JUnitConfig struct {
	// FailOnWarn treats warnings as failures (default: false, only errors are failures)
	FailOnWarn bool
}

// severityToUppercase converts severity to uppercase without allocation for known values
func severityToUppercase(severity string) string {
	switch severity {
	case model.SeverityError:
		return "ERROR"
	case model.SeverityWarn:
		return "WARN"
	case model.SeverityInfo:
		return "INFO"
	case model.SeverityHint:
		return "HINT"
	default:
		return strings.ToUpper(severity)
	}
}

// BuildJUnitReport generates a JUnit XML report from linting results.
// By default, only errors create failure elements. Use config.FailOnWarn to include warnings.
// Info and hint severities always create passing test cases.
func BuildJUnitReport(resultSet *model.RuleResultSet, t time.Time, args []string) []byte {
	return BuildJUnitReportWithConfig(resultSet, t, args, JUnitConfig{FailOnWarn: true})
}

// BuildJUnitReportWithConfig generates a JUnit XML report with configurable failure behavior.
func BuildJUnitReportWithConfig(resultSet *model.RuleResultSet, t time.Time, args []string, config JUnitConfig) []byte {

	since := time.Since(t)
	var suites []*TestSuite

	var cats = model.RuleCategoriesOrdered

	tmpl := `
	{{ .Message }}
	
    JSON Path: {{ .Path }}
	Rule: {{ .Rule.Id }}
	Severity: {{ .Rule.Severity }}
	Line: {{ .StartNode.Line }}`

	parsedTemplate, _ := template.New("failure").Parse(tmpl)

	gf, gtc := 0, 0 // global failure count, global test cases count.

	// isFailure determines if a severity level should be treated as a failure
	isFailure := func(severity string) bool {
		if severity == model.SeverityError {
			return true
		}
		if severity == model.SeverityWarn && config.FailOnWarn {
			return true
		}
		return false
	}

	for _, val := range cats {
		categoryResults := resultSet.GetResultsByRuleCategory(val.Id)

		f := 0
		var tc []*TestCase

		for _, r := range categoryResults {
			var sb bytes.Buffer
			_ = parsedTemplate.Execute(&sb, r)

			treatAsFailure := isFailure(r.Rule.Severity)
			if treatAsFailure {
				f++
				gf++
			}

			line := 1
			if r.StartNode != nil {
				line = r.StartNode.Line
			}

			tCase := &TestCase{
				Line:      line,
				Name:      fmt.Sprintf("%s", val.Name),
				ClassName: r.Rule.Id,
				Properties: &Properties{
					Properties: []*Property{
						{
							Name:  "path",
							Value: r.Path,
						},
						{
							Name:  "rule",
							Value: r.Rule.Id,
						},
						{
							Name:  "severity",
							Value: r.Rule.Severity,
						},
						{
							Name:  "line",
							Value: fmt.Sprintf("%d", line),
						},
					},
				},
			}

			// only create failure element for errors (and warnings if configured)
			// info and hint severities become passing test cases
			if treatAsFailure {
				tCase.Failure = &Failure{
					Message:  r.Message,
					Type:     severityToUppercase(r.Rule.Severity),
					Contents: sb.String(),
				}
			}

			// determine file path once and apply to all locations
			filePath := ""
			if r.Origin != nil && r.Origin.AbsoluteLocation != "" {
				filePath = r.Origin.AbsoluteLocation
			} else if len(args) > 0 {
				filePath = args[0]
			}

			if filePath != "" {
				tCase.File = filePath
				tCase.Properties.Properties = append(tCase.Properties.Properties, &Property{
					Name:  "file",
					Value: filePath,
				})
				if tCase.Failure != nil {
					tCase.Failure.File = filePath
				}
			}

			tc = append(tc, tCase)
		}

		if len(tc) > 0 {
			ts := &TestSuite{
				Name:      fmt.Sprintf("%s", val.Name),
				Tests:     len(categoryResults),
				Failures:  f,
				TestCases: tc,
			}

			suites = append(suites, ts)
		}
		gtc += len(tc)
	}

	allSuites := &TestSuites{
		TestSuites: suites,
		Tests:      gtc,
		Failures:   gf,
		Time:       since.Seconds(),
	}

	b, _ := xml.MarshalIndent(allSuites, "", " ")
	return b

}
