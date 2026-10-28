package html_report

import (
	"html/template"
	"io"
)

// ExportRow contains a complete finding for the standalone HTML export.
// Line and Column are one-based. All strings are escaped by html/template.
type ExportRow struct {
	Source, Rule, Message, Severity, Path string
	Line, Column                          int
}

var exportTemplate = template.Must(template.New("export").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>vacuum lint report</title>
<style>body{font:16px system-ui,sans-serif;margin:2rem;color:#20242b;background:#fff}table{border-collapse:collapse;width:100%}th,td{padding:.6rem;text-align:left;border-bottom:1px solid #ddd;vertical-align:top;overflow-wrap:anywhere}th{background:#f4f5f7}td{white-space:pre-wrap}caption{text-align:left;margin-bottom:1rem}</style></head>
<body><h1>vacuum lint report</h1><table><caption>{{len .}} findings</caption><thead><tr><th>Source</th><th>Location</th><th>Severity</th><th>Rule</th><th>Message</th><th>Path</th></tr></thead><tbody>{{range .}}<tr><td>{{.Source}}</td><td>{{.Line}}:{{.Column}}</td><td>{{.Severity}}</td><td>{{.Rule}}</td><td>{{.Message}}</td><td>{{.Path}}</td></tr>{{end}}</tbody></table></body></html>`))

// WriteExport writes all findings as a standalone HTML report without UI assets or scripts.
func WriteExport(w io.Writer, rows []ExportRow) error { return exportTemplate.Execute(w, rows) }
