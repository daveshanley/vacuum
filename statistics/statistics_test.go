package statistics

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestCreateReportStatistics(t *testing.T) {

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	specBytes, _ := ioutil.ReadFile("../model/test_files/petstorev3.json")

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet: selectedRS,
		Spec:    specBytes,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	stats := CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

	assert.Equal(t, 30, stats.FilesizeKB)
	assert.Equal(t, 7, stats.References)
	assert.Equal(t, 9, stats.Parameters)
}

func TestCreateReportStatistics_Perfect(t *testing.T) {

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	specBytes, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet: selectedRS,
		Spec:    specBytes,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	stats := CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

	assert.Equal(t, 100, stats.OverallScore)

}

func TestCreateReportStatistics_BigLoadOfIssues(t *testing.T) {

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	specBytes, _ := ioutil.ReadFile("../model/test_files/api.github.com.yaml")

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet: selectedRS,
		Spec:    specBytes,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	stats := CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

	assert.Equal(t, 10, stats.OverallScore)

}
