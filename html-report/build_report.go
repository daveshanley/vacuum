// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT
package html_report

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/alecthomas/chroma"
	html_format "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"strings"
	"text/template"
)

//go:embed templates/report-template.gohtml
var reportTemplate string

//go:embed templates/header.gohtml
var header string

//go:embed templates/footer.gohtml
var footer string

//go:embed ui/build/static/js/vacuumReport.js
var bundledJS string

//go:embed ui/build/static/js/hydrate.js
var hydrateJS string

//go:embed ui/build/static/js/shoelace.js
var shoelaceJS string

//go:embed ui/src/css/report.css
var reportCSS string

type HTMLReport interface {
	GenerateReport(testMode bool) []byte
}

type ReportData struct {
	BundledJS      string                    `json:"bundledJS"`
	HydrateJS      string                    `json:"hydrateJS"`
	ShoelaceJS     string                    `json:"shoelaceJS"`
	ReportCSS      string                    `json:"reportCSS"`
	Statistics     *reports.ReportStatistics `json:"reportStatistics"`
	TestMode       bool                      `json:"test"`
	RuleCategories []*model.RuleCategory     `json:"ruleCategories"`
	RuleResults    *model.RuleResultSet      `json:"ruleResults"`
	SpecString     []string                  `json:"-"`
}

func NewHTMLReport(
	index *model.SpecIndex,
	info *model.SpecInfo,
	results *model.RuleResultSet,
	stats *reports.ReportStatistics) HTMLReport {
	return &htmlReport{index, info, results, stats}
}

type htmlReport struct {
	index   *model.SpecIndex
	info    *model.SpecInfo
	results *model.RuleResultSet
	stats   *reports.ReportStatistics
}

func (html htmlReport) GenerateReport(test bool) []byte {

	tmpl := template.New("header")
	templateFuncs := template.FuncMap{
		"renderJSON": func(data interface{}) string {
			b, _ := json.Marshal(data)
			return string(b)
		},
		"extractResultsForCategory": func(cat string, results *model.RuleResultSet) *model.RuleResultsForCategory {
			var r *model.RuleResultsForCategory
			// todo: make this configurable.
			limit := 100

			if cat == "all" {
				// todo, replace this with something not wrong.
				r = results.GetResultsForCategoryWithLimit("schemas", limit)
				return r
			}
			r = results.GetResultsForCategoryWithLimit(cat, limit)
			return r
		},
		"ruleSeverityIcon": func(sev string) string {
			switch sev {
			case "error":
				return "❌"
			case "warn":
				return "⚠️"
			case "info":
				return "🔵"
			case "hint":
				return "💠"
			}
			return ""
		},
		"renderSource": func(r *model.RuleFunctionResult, specData []string) string {

			// let's go chroma!
			lexer := lexers.Get("yaml")
			lexer = chroma.Coalesce(lexer)

			style := styles.Get("swapoff")
			if style == nil {
				style = styles.Fallback
			}

			iterator, _ := lexer.Tokenise(nil, html.renderCodeSnippetForResult(r, specData, 8, 8))
			b := new(strings.Builder)

			lineRange := [][2]int{[2]int{r.StartNode.Line, r.EndNode.Line}}

			formatter := html_format.New(
				html_format.WithClasses(true),
				html_format.WithLineNumbers(true),
				html_format.BaseLineNumber(r.StartNode.Line-8),
				html_format.HighlightLines(lineRange))
			err := formatter.Format(b, style, iterator)

			if err != nil {
				return fmt.Sprintf("Oh My Stars! I cannot render the code: %v", err.Error())
			}
			return b.String()
		},
	}
	tmpl.Funcs(templateFuncs)

	var byteBuf bytes.Buffer
	t, _ := tmpl.Parse(header)
	t.New("footer").Parse(footer)
	t.New("report").Parse(reportTemplate)

	// we need a new category here 'all'
	cats := model.RuleCategoriesOrdered
	allCat := model.RuleCategory{
		Id:          "all",
		Name:        "All Categories",
		Description: "View everything from all categories",
	}

	n := []*model.RuleCategory{&allCat}
	cats = append(n, cats...)

	var specStringData []string

	if html.info != nil {
		specStringData = strings.Split(string(*html.info.SpecBytes), "\n")
	}

	reportData := &ReportData{
		BundledJS:      bundledJS,
		HydrateJS:      hydrateJS,
		ShoelaceJS:     shoelaceJS,
		ReportCSS:      reportCSS,
		Statistics:     html.stats,
		RuleCategories: cats,
		TestMode:       test,
		RuleResults:    html.results,
		SpecString:     specStringData,
	}
	err := t.ExecuteTemplate(&byteBuf, "report", reportData)

	if err != nil {
		return []byte(fmt.Sprintf("failed to render: %v", err.Error()))
	}

	return byteBuf.Bytes()
}

func (html htmlReport) renderCodeSnippetForResult(r *model.RuleFunctionResult, specData []string, before, after int) string {

	buf := new(strings.Builder)

	startLine := r.StartNode.Line - 1
	endLine := r.StartNode.Line

	if startLine-before < 0 {
		startLine = before - ((startLine - before) * -1)
	} else {
		startLine = startLine - before
	}

	if r.StartNode.Line+after >= len(specData)-1 {
		endLine = len(specData) - 1
	} else {
		endLine = r.StartNode.Line - 1 + after
	}

	firstDelta := (r.StartNode.Line - 1) - startLine
	secondDelta := endLine - r.StartNode.Line
	for i := 0; i < firstDelta; i++ {
		line := specData[startLine+i]
		buf.WriteString(fmt.Sprintf("%s\n", line))
	}

	// todo, fix this.
	line := specData[r.StartNode.Line-1]
	buf.WriteString(fmt.Sprintf("%s\n", line))

	for i := 0; i < secondDelta; i++ {
		line = specData[r.StartNode.Line+i]
		buf.WriteString(fmt.Sprintf("%s\n", line))
	}

	return buf.String()
}
