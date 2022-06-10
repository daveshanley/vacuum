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
	"github.com/daveshanley/vacuum/utils"
	"strings"
	"text/template"
	"time"
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

// MaxViolations the maximum number of violations the report will render per broken rule.
// TODO: make this configurable
const MaxViolations = 100

type ReportData struct {
	BundledJS      string                    `json:"bundledJS"`
	HydrateJS      string                    `json:"hydrateJS"`
	ShoelaceJS     string                    `json:"shoelaceJS"`
	ReportCSS      string                    `json:"reportCSS"`
	Statistics     *reports.ReportStatistics `json:"reportStatistics"`
	TestMode       bool                      `json:"test"`
	RuleCategories []*model.RuleCategory     `json:"ruleCategories"`
	RuleResults    *model.RuleResultSet      `json:"ruleResults"`
	MaxViolations  int                       `json:"maxViolations"`
	Generated      time.Time                 `json:"generated"`
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

	templateFuncs := template.FuncMap{
		"renderJSON": func(data interface{}) string {
			b, _ := json.Marshal(data)
			return string(b)
		},
		"timeGenerated": func(t time.Time) string {
			return t.Format("02 Jan 2006 15:04:05 EST")
		},
		"extractResultsForCategory": func(cat string, results *model.RuleResultSet) *model.RuleResultsForCategory {
			var r *model.RuleResultsForCategory
			limit := MaxViolations

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
			iterator, _ := lexer.Tokenise(nil, html.renderCodeSnippetForResult(r, specData, 3, 3))
			b := new(strings.Builder)

			l := r.StartNode.Line
			lineRange := [][2]int{[2]int{l, l}}

			formatter := html_format.New(
				html_format.WithClasses(true),
				html_format.WithLineNumbers(true),
				html_format.BaseLineNumber(r.StartNode.Line-2),
				html_format.HighlightLines(lineRange))
			err := formatter.Format(b, style, iterator)

			if err != nil {
				return fmt.Sprintf("Oh My Stars! I cannot render the code: %v", err.Error())
			}
			return b.String()
		},
	}
	tmpl := template.New("header")
	tmpl.Funcs(templateFuncs)
	t, _ := tmpl.Parse(header)
	t.New("footer").Parse(footer)
	t.New("report").Parse(reportTemplate)

	var byteBuf bytes.Buffer

	// we need a new category here 'all'
	cats := model.RuleCategoriesOrdered
	n := []*model.RuleCategory{model.RuleCategories[model.CategoryAll]}
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
		MaxViolations:  MaxViolations,
		SpecString:     specStringData,
		Generated:      html.info.Generated,
	}
	err := t.ExecuteTemplate(&byteBuf, "report", reportData)

	if err != nil {
		return []byte(fmt.Sprintf("failed to render: %v", err.Error()))
	}

	return byteBuf.Bytes()
}

func (html htmlReport) renderCodeSnippetForResult(r *model.RuleFunctionResult, specData []string, before, after int) string {
	return utils.RenderCodeSnippet(r.StartNode, specData, before, after)
}
