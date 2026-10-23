package html_report

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/testify/require"
)

func TestHTMLReportCategoryScores(t *testing.T) {
	for _, score := range []int{0, 73, 100} {
		stats := &reports.ReportStatistics{OverallScore: score, CategoryStatistics: []*reports.CategoryStatistic{{CategoryName: "Schema <checks>", Score: 42, NumIssues: 3}}}
		report := NewHTMLReport(nil, nil, model.NewRuleResultSet(nil), stats, false)
		for _, testMode := range []bool{false, true} {
			output := string(report.GenerateReport(testMode, "test"))
			require.Contains(t, output, "<category-piechart></category-piechart>")
			require.Contains(t, output, "Schema &lt;checks&gt;</th><td>42%</td><td>3</td>")
			require.Contains(t, output, "window.statistics =")
			require.Contains(t, output, fmt.Sprintf("<th scope=\"row\">Overall</th><td>%d%%</td>", score))
		}
	}
}
