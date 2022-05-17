package html_report

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestHtmlReport_GenerateReport(t *testing.T) {

	report := NewHTMLReport(nil, nil, nil, nil)
	assert.NotEmpty(t, report.GenerateReport(true))

}

func TestHtmlReport_GenerateReport_File(t *testing.T) {

	report := NewHTMLReport(nil, nil, nil, nil)
	generated := report.GenerateReport(false)

	tmp, _ := ioutil.TempFile("", "")
	ioutil.WriteFile(tmp.Name(), generated, 0664)
	stat, _ := os.Stat(tmp.Name())

	assert.Greater(t, int(stat.Size()), 0)

}
