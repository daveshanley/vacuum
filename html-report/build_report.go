// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT
package html_report

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
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
			if cat == "all" {
				// todo, replace this with something not wrong.
				return results.GetRuleResultsForCategory("schemas")
			}
			return results.GetRuleResultsForCategory(cat)
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

	reportData := &ReportData{
		BundledJS:      bundledJS,
		HydrateJS:      hydrateJS,
		ShoelaceJS:     shoelaceJS,
		ReportCSS:      reportCSS,
		Statistics:     html.stats,
		RuleCategories: cats,
		TestMode:       test,
		RuleResults:    html.results,
	}
	t.ExecuteTemplate(&byteBuf, "report", reportData)

	return byteBuf.Bytes()
}
