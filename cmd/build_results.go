package cmd

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"

	"github.com/daveshanley/vacuum/tui"
	"net/http"
	"time"
)

func BuildResults(
	silent bool,
	hardMode bool,
	rulesetFlag string,
	specBytes []byte,
	customFunctions map[string]model.RuleFunction,
	base string,
	remote bool,
	timeout time.Duration,
	lookupTimeout time.Duration,
	httpClientConfig utils.HTTPClientConfig,
	ignoredItems model.IgnoredItems) (*model.RuleResultSet, *motor.RuleSetExecutionResult, error) {
	return BuildResultsWithDocCheckSkip(silent, hardMode, rulesetFlag, specBytes, customFunctions, base, remote, false, timeout, lookupTimeout, httpClientConfig, ignoredItems)
}

func BuildResultsWithDocCheckSkip(
	silent bool,
	hardMode bool,
	rulesetFlag string,
	specBytes []byte,
	customFunctions map[string]model.RuleFunction,
	base string,
	remote bool,
	skipCheck bool,
	timeout time.Duration,
	lookupTimeout time.Duration,
	httpClientConfig utils.HTTPClientConfig,
	ignoredItems model.IgnoredItems) (*model.RuleResultSet, *motor.RuleSetExecutionResult, error) {

	// read spec and parse
	defaultRuleSets := rulesets.BuildDefaultRuleSets()

	// default is recommended rules, based on spectral (for now anyway)
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

	// HARD MODE
	if hardMode {
		selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()

		// extract all OWASP Rules
		owaspRules := rulesets.GetAllOWASPRules()
		allRules := selectedRS.Rules
		for k, v := range owaspRules {
			allRules[k] = v
		}
		if !silent {
			tui.RenderStyledBox(HardModeEnabled, tui.BoxTypeHard, false)
		}
	}

	// if ruleset has been supplied, lets make sure it exists, then load it in
	// and see if it's valid. If so - let's go!
	if rulesetFlag != "" {

		// Create HTTP client for remote ruleset downloads if needed
		var httpClient *http.Client
		if utils.ShouldUseCustomHTTPClient(httpClientConfig) {
			var clientErr error
			httpClient, clientErr = utils.CreateCustomHTTPClient(httpClientConfig)
			if clientErr != nil {
				return nil, nil, fmt.Errorf("failed to create custom HTTP client: %w", clientErr)
			}
		}

		var rsErr error
		selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remote, httpClient)
		if rsErr != nil {
			return nil, nil, rsErr
		}

		// Merge OWASP rules if hard mode is enabled
		if MergeOWASPRulesToRuleSet(selectedRS, hardMode) {
			if !silent {
				tui.RenderStyledBox(HardModeWithCustomRuleset, tui.BoxTypeHard, false)
			}
		}
	}

	tui.RenderInfo("Linting against %d rules: %s", len(selectedRS.Rules), selectedRS.DocumentationURI)

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:           selectedRS,
		Spec:              specBytes,
		CustomFunctions:   customFunctions,
		Base:              base,
		SkipDocumentCheck: skipCheck,
		AllowLookup:       remote,
		Timeout:           timeout,
		NodeLookupTimeout: lookupTimeout,
		HTTPClientConfig:  httpClientConfig,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	resultSet.SortResultsByLineNumber()
	resultSet.Results = utils.FilterIgnoredResultsPtr(resultSet.Results, ignoredItems)
	return resultSet, ruleset, nil
}
