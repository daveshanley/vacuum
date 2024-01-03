package statistics

import (
	"context"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestCreateReportStatistics(t *testing.T) {

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	specBytes, _ := os.ReadFile("../model/test_files/petstorev3.json")

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

func TestCreateReportStatistics_AlmostPerfect(t *testing.T) {

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	specBytes, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet: selectedRS,
		Spec:    specBytes,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	stats := CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

	//assert.Equal(t, 100, stats.OverallScore)
	// new missing examples function is now strict / correct
	assert.Equal(t, 95, stats.OverallScore)

}

func TestCreateReportStatistics_BigLoadOfIssues(t *testing.T) {

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	specBytes, _ := os.ReadFile("../model/test_files/api.github.com.yaml")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	d := make(chan bool)
	go func(f chan bool) {

		ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
			RuleSet:     selectedRS,
			Spec:        specBytes,
			AllowLookup: true,
		})
		resultSet := model.NewRuleResultSet(ruleset.Results)
		stats := CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

		assert.Equal(t, 10, stats.OverallScore)
		f <- true
	}(d)

	select {
	case <-ctx.Done():
		assert.Fail(t, "Timed out, we have an issue that needs fixing")
	case <-d:
		break
	}

}
