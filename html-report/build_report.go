// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT
package html_report

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed templates/report-template.gohtml
var reportTemplate string

//go:embed templates/header.gohtml
var header string

//go:embed templates/footer.gohtml
var footer string

//go:embed ui/build/static/js/vacuum-report.js
var bundledJS string

type HTMLReport interface {
	GenerateReport() []byte
}

type ReportData struct {
	BundledJS string `json:"bundledJS"`
}

func NewHTMLReport() HTMLReport {
	return &htmlReport{}
}

type htmlReport struct {
}

func (html htmlReport) GenerateReport() []byte {

	var byteBuf bytes.Buffer
	t, _ := template.New("header").Parse(header)
	t.New("footer").Parse(footer)
	t.New("report").Parse(reportTemplate)

	reportData := &ReportData{bundledJS}

	t.ExecuteTemplate(&byteBuf, "report", reportData)

	return byteBuf.Bytes()
}
