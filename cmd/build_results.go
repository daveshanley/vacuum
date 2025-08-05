package cmd

import (
	"context"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pterm/pterm"
	"net/http"
	"os"
	"strings"
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
	httpClientConfig utils.HTTPClientConfig) (*model.RuleResultSet, *motor.RuleSetExecutionResult, error) {
	return BuildResultsWithDocCheckSkip(silent, hardMode, rulesetFlag, specBytes, customFunctions, base, remote, false, timeout, httpClientConfig)
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
	httpClientConfig utils.HTTPClientConfig) (*model.RuleResultSet, *motor.RuleSetExecutionResult, error) {

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
			box := pterm.DefaultBox.WithLeftPadding(5).WithRightPadding(5)
			box.BoxStyle = pterm.NewStyle(pterm.FgLightRed)
			box.Println(pterm.LightRed("🚨 HARD MODE ENABLED 🚨"))
			pterm.Println()
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

		if strings.HasPrefix(rulesetFlag, "http") {
			// Handle remote ruleset URL
			if !remote {
				return nil, nil, fmt.Errorf("remote ruleset specified but remote flag is disabled (use --remote=true or -u=true)")
			}
			
			downloadedRS, rsErr := rulesets.DownloadRemoteRuleSet(context.Background(), rulesetFlag, httpClient)
			if rsErr != nil {
				return nil, nil, rsErr
			}
			selectedRS = defaultRuleSets.GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(downloadedRS, httpClient)
		} else {
			// Handle local ruleset file
			rsBytes, rsErr := os.ReadFile(rulesetFlag)
			if rsErr != nil {
				return nil, nil, rsErr
			}
			selectedRS, rsErr = BuildRuleSetFromUserSuppliedSetWithHTTPClient(rsBytes, defaultRuleSets, httpClient)
			if rsErr != nil {
				return nil, nil, rsErr
			}
		}
	}

	pterm.Info.Printf("Linting against %d rules: %s\n", len(selectedRS.Rules), selectedRS.DocumentationURI)

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:           selectedRS,
		Spec:              specBytes,
		CustomFunctions:   customFunctions,
		Base:              base,
		SkipDocumentCheck: skipCheck,
		AllowLookup:       remote,
		Timeout:           timeout,
		HTTPClientConfig:  httpClientConfig,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	resultSet.SortResultsByLineNumber()
	return resultSet, ruleset, nil
}
