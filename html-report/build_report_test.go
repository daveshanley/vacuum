package html_report

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestHtmlReport_GenerateReport(t *testing.T) {

	report := NewHTMLReport()
	assert.NotEmpty(t, report.GenerateReport(true))

}

func TestHtmlReport_GenerateReport_File(t *testing.T) {

	report := NewHTMLReport()
	generated := report.GenerateReport(true)
	ioutil.WriteFile("report.html", generated, 0664)

}
