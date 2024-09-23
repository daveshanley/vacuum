package languageserver

import (
	"os"

	"github.com/daveshanley/vacuum/plugin"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func (s *ServerState) onConfigChange(e fsnotify.Event) {

	// extract flags
	rulesetFlag := viper.GetString("ruleset")
	functionsFlag := viper.GetString("functions")
	baseFlag := viper.GetString("base")
	skipCheckFlag := viper.GetBool("skip-check")
	remoteFlag := viper.GetBool("remote")
	timeoutFlag := viper.GetInt("timeout")
	hardModeFlag := viper.GetBool("hard-mode")
	ignoreArrayCircleRef := viper.GetBool("ignore-array-circle-ref")
	ignorePolymorphCircleRef := viper.GetBool("ignore-array-circle-ref")

	defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(s.lintRequest.Logger)
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	functions := s.lintRequest.Functions

	// FUNCTIONS
	if functionsFlag != "" {
		pm, err := plugin.LoadFunctions(functionsFlag, true)
		if err == nil {
			functions = pm.GetCustomFunctions()
		}
	}

	// HARD MODE
	if hardModeFlag {
		selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()

		// extract all OWASP Rules
		owaspRules := rulesets.GetAllOWASPRules()
		allRules := selectedRS.Rules
		for k, v := range owaspRules {
			allRules[k] = v
		}
	}

	// RULESET
	if rulesetFlag != "" {
		rsBytes, rsErr := os.ReadFile(rulesetFlag)
		if rsErr == nil {
			// load in our user supplied ruleset and try to validate it.
			userRS, userErr := rulesets.CreateRuleSetFromData(rsBytes)
			if userErr == nil {
				selectedRS = defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(userRS)
			}
		}
	}

	s.lintRequest.BaseFlag = baseFlag
	s.lintRequest.Remote = remoteFlag
	s.lintRequest.SkipCheckFlag = skipCheckFlag
	s.lintRequest.DefaultRuleSets = defaultRuleSets
	s.lintRequest.SelectedRS = selectedRS
	s.lintRequest.Functions = functions
	s.lintRequest.TimeoutFlag = timeoutFlag
	s.lintRequest.IgnoreArrayCircleRef = ignoreArrayCircleRef
	s.lintRequest.IgnorePolymorphCircleRef = ignorePolymorphCircleRef
}
