package benchmarks

import (
	html_report "github.com/daveshanley/vacuum/html-report"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/statistics"
	"io/ioutil"
	"testing"
)

func BenchmarkHtmlReport_GenerateReport(b *testing.B) {

	specBytes, _ := ioutil.ReadFile("../model/test_files/stripe.yaml")
	defaultRuleSets := rulesets.BuildDefaultRuleSets()

	// default is recommended rules, based on spectral (for now anyway)
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet: selectedRS,
		Spec:    specBytes,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	resultSet.SortResultsByLineNumber()

	// generate statistics
	stats := statistics.CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

	for n := 0; n < b.N; n++ {
		// generate html report
		report := html_report.NewHTMLReport(ruleset.Index, ruleset.SpecInfo, resultSet, stats)
		report.GenerateReport(true)

	}

}
