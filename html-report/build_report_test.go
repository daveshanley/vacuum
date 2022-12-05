package html_report

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/statistics"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestHtmlReport_GenerateReport(t *testing.T) {

	report := NewHTMLReport(nil, nil, nil, nil, false)
	assert.NotEmpty(t, report.GenerateReport(true))

}

func TestHtmlReport_GenerateReport_File(t *testing.T) {

	report := NewHTMLReport(nil, nil, nil, nil, false)
	generated := report.GenerateReport(false)

	tmp, _ := os.CreateTemp("", "")
	err := os.WriteFile(tmp.Name(), generated, 0664)
	assert.NoError(t, err)
	stat, _ := os.Stat(tmp.Name())

	assert.Greater(t, int(stat.Size()), 0)

}

func TestNewHTMLReport_FullRender_Stripe(t *testing.T) {

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

	report := NewHTMLReport(ruleset.Index, ruleset.SpecInfo, resultSet, stats, false)
	generated := report.GenerateReport(true)
	assert.True(t, len(generated) > 0)

}
