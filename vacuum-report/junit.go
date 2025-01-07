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

func BuildJUnitReport(resultSet *model.RuleResultSet, t time.Time, args []string) []byte {

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

	// try a category print out.
	for _, val := range cats {
		categoryResults := resultSet.GetResultsByRuleCategory(val.Id)

		f := 0
		var tc []*TestCase

		for _, r := range categoryResults {
			var sb bytes.Buffer
			_ = parsedTemplate.Execute(&sb, r)
			if r.Rule.Severity == model.SeverityError || r.Rule.Severity == model.SeverityWarn {
				f++
				gf++
			}

			tCase := &TestCase{
				Line:      r.StartNode.Line,
				Name:      fmt.Sprintf("%s", val.Name),
				ClassName: r.Rule.Id,
				Failure: &Failure{
					Message:  r.Message,
					Type:     strings.ToUpper(r.Rule.Severity),
					Contents: sb.String(),
				},
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
							Value: fmt.Sprintf("%d", r.StartNode.Line),
						},
					},
				},
			}

			if r.Origin != nil && r.Origin.AbsoluteLocation != "" {
				tCase.File = r.Origin.AbsoluteLocation
				tCase.Properties.Properties = append(tCase.Properties.Properties, &Property{
					Name:  "file",
					Value: r.Origin.AbsoluteLocation,
				})
				tCase.Failure.File = r.Origin.AbsoluteLocation
			} else {
				if len(args) > 0 {
					tCase.File = args[0]
					tCase.Properties.Properties = append(tCase.Properties.Properties, &Property{
						Name:  "file",
						Value: args[0],
					})
					tCase.Failure.File = args[0]
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
