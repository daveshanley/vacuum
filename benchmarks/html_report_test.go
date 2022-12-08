package benchmarks

import (
	"crypto/sha256"
	"fmt"
	html_report "github.com/daveshanley/vacuum/html-report"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/statistics"
	"os"
	"testing"
)

func BenchmarkHtmlReport_GenerateReport(b *testing.B) {

	specBytes, _ := os.ReadFile("../model/test_files/stripe.yaml")
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
		report := html_report.NewHTMLReport(ruleset.Index, ruleset.SpecInfo, resultSet, stats, false)
		report.GenerateReport(true)

	}

}

func BenchmarkHtmlReport_GenerateReportIdentical(b *testing.B) {
	specBytes, _ := os.ReadFile("../model/test_files/pegel-online-api.yaml")
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
		// generate html reports and compare hash
		reportA := html_report.NewHTMLReport(ruleset.Index, ruleset.SpecInfo, resultSet, stats, true)
		reportABytes := reportA.GenerateReport(false)

		reportB := html_report.NewHTMLReport(ruleset.Index, ruleset.SpecInfo, resultSet, stats, true)
		reportBBytes := reportB.GenerateReport(false)

		hashA := sha256.Sum256(reportABytes)
		hashB := sha256.Sum256(reportBBytes)

		if hashA != hashB {
			panic("failed identical check")
		}
	}
}

func TestHtmlReport_GenerateReportIdenticalRun200(t *testing.T) {
	specBytes, _ := os.ReadFile("../model/test_files/pegel-online-api.yaml")
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

	reportZero := html_report.NewHTMLReport(ruleset.Index, ruleset.SpecInfo, resultSet, stats, true)
	reportZeroBytes := reportZero.GenerateReport(false)
	hashZero := sha256.Sum256(reportZeroBytes)

	for n := 0; n < 200; n++ {
		// generate html reports and compare hash
		reportA := html_report.NewHTMLReport(ruleset.Index, ruleset.SpecInfo, resultSet, stats, true)
		reportABytes := reportA.GenerateReport(false)

		reportB := html_report.NewHTMLReport(ruleset.Index, ruleset.SpecInfo, resultSet, stats, true)
		reportBBytes := reportB.GenerateReport(false)

		hashA := sha256.Sum256(reportABytes)
		hashB := sha256.Sum256(reportBBytes)

		if hashA != hashB {
			panic("failed identical check")
		}

		if hashA != hashZero {
			panic("failed identical check")
		}

		if hashB != hashZero {
			panic("failed identical check")
		}

	}
	fmt.Print("done")
}
