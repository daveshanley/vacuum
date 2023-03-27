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
	Time      float64     `xml:"time,attr"`
	TestCases []*TestCase `xml:"testcase"`
}

type TestCase struct {
	Name      string   `xml:"name,attr"`
	ClassName string   `xml:"classname,attr"`
	Time      float64  `xml:"time,attr"`
	Failure   *Failure `xml:"failure,omitempty"`
}

type Failure struct {
	Message  string `xml:"message,attr,omitempty"`
	Type     string `xml:"type,attr,omitempty"`
	Contents string `xml:",innerxml"`
}

func BuildJUnitReport(resultSet *model.RuleResultSet, t time.Time) []byte {

	since := time.Since(t)

	var suites []*TestSuite

	var cats = model.RuleCategoriesOrdered

	tmpl := `
	{{ .Message }}
	JSON Path: {{ .Path }}
	Rule: {{ .Rule.Id }}
	Severity: {{ .Rule.Severity }}
	Start Line: {{ .StartNode.Line }}
	End Line: {{ .EndNode.Line }}`

	parsedTemplate, _ := template.New("failure").Parse(tmpl)

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
			}
			tc = append(tc, &TestCase{
				Name:      fmt.Sprintf("Category: %s", val.Id),
				ClassName: r.Rule.Id,
				Time:      since.Seconds(),
				Failure: &Failure{
					Message:  r.Message,
					Type:     strings.ToUpper(r.Rule.Severity),
					Contents: sb.String(),
				},
			})
		}

		if len(tc) > 0 {
			ts := &TestSuite{
				Name:      fmt.Sprintf("Category: %s", val.Id),
				Tests:     len(categoryResults),
				Failures:  f,
				Time:      since.Seconds(),
				TestCases: tc,
			}

			suites = append(suites, ts)
		}
	}

	b, _ := xml.MarshalIndent(suites, "", " ")
	return b

}
