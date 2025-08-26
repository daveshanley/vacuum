package languageserver

import (
	"os"
	"strings"

	"github.com/daveshanley/vacuum/plugin"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// initializeConfig loads the vacuum configuration on language server startup
func (s *ServerState) initializeConfig() {
	// Set up environment variable support
	viper.SetEnvPrefix("VACUUM")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	
	// Try to load config file
	viper.SetConfigName("vacuum.conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath(getXdgConfigHome())
	
	// Read the config file (ignore if not found)
	_ = viper.ReadInConfig()
	
	// Update lint request with loaded configuration
	s.updateLintRequestFromConfig()
}

// getXdgConfigHome gets config directory as per the xdg basedir spec
func getXdgConfigHome() string {
	xdgConfigHome, exists := os.LookupEnv("XDG_CONFIG_HOME")
	if !exists {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		xdgConfigHome = home + "/.config"
	}
	return xdgConfigHome + "/vacuum"
}

// updateLintRequestFromConfig updates the lint request with values from viper configuration
func (s *ServerState) updateLintRequestFromConfig() {
	// Only update if not already set by command line flags
	if s.lintRequest.BaseFlag == "" {
		if base := viper.GetString("base"); base != "" {
			s.lintRequest.BaseFlag = base
		}
	}
	
	// Update other configuration values
	if viper.IsSet("remote") {
		s.lintRequest.Remote = viper.GetBool("remote")
	}
	if viper.IsSet("skip-check") {
		s.lintRequest.SkipCheckFlag = viper.GetBool("skip-check")
	}
	if viper.IsSet("timeout") {
		s.lintRequest.TimeoutFlag = viper.GetInt("timeout")
	}
	if viper.IsSet("ignore-array-circle-ref") {
		s.lintRequest.IgnoreArrayCircleRef = viper.GetBool("ignore-array-circle-ref")
	}
	if viper.IsSet("ignore-polymorph-circle-ref") {
		s.lintRequest.IgnorePolymorphCircleRef = viper.GetBool("ignore-polymorph-circle-ref")
	}
	if viper.IsSet("ext-refs") {
		s.lintRequest.ExtensionRefs = viper.GetBool("ext-refs")
	}
	
	// Handle ruleset if specified
	if rulesetFlag := viper.GetString("ruleset"); rulesetFlag != "" && s.lintRequest.SelectedRS == nil {
		s.loadRulesetFromConfig(rulesetFlag)
	}
	
	// Handle functions if specified
	if functionsFlag := viper.GetString("functions"); functionsFlag != "" && s.lintRequest.Functions == nil {
		s.loadFunctionsFromConfig(functionsFlag)
	}
}

func (s *ServerState) loadRulesetFromConfig(rulesetFlag string) {
	rsBytes, rsErr := os.ReadFile(rulesetFlag)
	if rsErr == nil {
		userRS, userErr := rulesets.CreateRuleSetFromData(rsBytes)
		if userErr == nil {
			s.lintRequest.SelectedRS = s.lintRequest.DefaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(userRS)
		}
	}
}

func (s *ServerState) loadFunctionsFromConfig(functionsFlag string) {
	pm, err := plugin.LoadFunctions(functionsFlag, true)
	if err == nil {
		s.lintRequest.Functions = pm.GetCustomFunctions()
	}
}

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
