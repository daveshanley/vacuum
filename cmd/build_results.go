package cmd

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"os"
)

func BuildResults(
	rulesetFlag string,
	specBytes []byte,
	customFunctions map[string]model.RuleFunction,
	base string) (*model.RuleResultSet, *motor.RuleSetExecutionResult, error) {
	return BuildResultsWithDocCheckSkip(rulesetFlag, specBytes, customFunctions, base, false)
}

func BuildResultsWithDocCheckSkip(
	rulesetFlag string,
	specBytes []byte,
	customFunctions map[string]model.RuleFunction,
	base string,
	skipCheck bool) (*model.RuleResultSet, *motor.RuleSetExecutionResult, error) {

	// read spec and parse
	defaultRuleSets := rulesets.BuildDefaultRuleSets()

	// default is recommended rules, based on spectral (for now anyway)
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

	// if ruleset has been supplied, lets make sure it exists, then load it in
	// and see if it's valid. If so - let's go!
	if rulesetFlag != "" {

		rsBytes, rsErr := os.ReadFile(rulesetFlag)
		if rsErr != nil {
			return nil, nil, rsErr
		}
		selectedRS, rsErr = BuildRuleSetFromUserSuppliedSet(rsBytes, defaultRuleSets)
		if rsErr != nil {
			return nil, nil, rsErr
		}
	}

	pterm.Info.Printf("Linting against %d rules: %s\n", len(selectedRS.Rules), selectedRS.DocumentationURI)

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:           selectedRS,
		Spec:              specBytes,
		CustomFunctions:   customFunctions,
		Base:              base,
		SkipDocumentCheck: skipCheck,
		AllowLookup:       true,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	resultSet.SortResultsByLineNumber()
	return resultSet, ruleset, nil
}
