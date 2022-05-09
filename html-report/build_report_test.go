package html_report

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestHtmlReport_GenerateReport(t *testing.T) {

	report := NewHTMLReport()
	assert.NotEmpty(t, report.GenerateReport())

}

func TestHtmlReport_GenerateReport_File(t *testing.T) {

	report := NewHTMLReport()
	ioutil.WriteFile("report.html", report.GenerateReport(), 0664)

}
