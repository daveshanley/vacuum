package cmd

import (
	"fmt"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/tui"
	"github.com/daveshanley/vacuum/utils"
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
	fetchConfig *utils.FetchConfig,
	ignoredItems model.IgnoredItems,
	turboFlags *TurboFlags) (*model.RuleResultSet, *motor.RuleSetExecutionResult, error) {
	return BuildResultsWithDocCheckSkip(silent, hardMode, rulesetFlag, specBytes, customFunctions, base, remote, false, timeout, lookupTimeout, httpClientConfig, fetchConfig, ignoredItems, turboFlags)
}

// TurboFlags holds turbo-related configuration for BuildResults functions.
type TurboFlags struct {
	TurboMode         bool
	SkipResolve       bool
	SkipCircularCheck bool
	SkipSchemaErrors  bool
	MaxResultsPerRule int
	MaxTotalResults   int
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
	fetchConfig *utils.FetchConfig,
	ignoredItems model.IgnoredItems,
	turboFlags *TurboFlags) (*model.RuleResultSet, *motor.RuleSetExecutionResult, error) {

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
		httpClient, clientErr := utils.CreateHTTPClientIfNeeded(httpClientConfig)
		if clientErr != nil {
			return nil, nil, fmt.Errorf("failed to create custom HTTP client: %w", clientErr)
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

	// Apply turbo mode rule filtering
	if turboFlags != nil && turboFlags.TurboMode {
		rulesets.FilterRulesForTurbo(selectedRS)
	}

	tui.RenderInfo("Linting against %d rules: %s", len(selectedRS.Rules), selectedRS.DocumentationURI)

	exec := &motor.RuleSetExecution{
		RuleSet:           selectedRS,
		Spec:              specBytes,
		CustomFunctions:   customFunctions,
		Base:              base,
		SkipDocumentCheck: skipCheck,
		AllowLookup:       remote,
		Timeout:           timeout,
		NodeLookupTimeout: lookupTimeout,
		HTTPClientConfig:  httpClientConfig,
		FetchConfig:       fetchConfig,
	}
	if turboFlags != nil {
		exec.TurboMode = turboFlags.TurboMode
		exec.SkipResolve = turboFlags.SkipResolve
		exec.SkipCircularCheck = turboFlags.SkipCircularCheck
		exec.SkipSchemaErrors = turboFlags.SkipSchemaErrors
		exec.MaxResultsPerRule = turboFlags.MaxResultsPerRule
		exec.MaxTotalResults = turboFlags.MaxTotalResults
	}

	ruleset := motor.ApplyRulesToRuleSet(exec)

	resultSet := model.NewRuleResultSet(ruleset.Results)
	resultSet.SortResultsByLineNumber()
	resultSet.Results = utils.FilterIgnoredResultsPtr(resultSet.Results, ignoredItems)
	return resultSet, ruleset, nil
}
