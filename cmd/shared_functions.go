// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/dustin/go-humanize"
	"github.com/pb33f/libopenapi/index"
	"github.com/pterm/pterm"
	"net/http"
	"os"
	"strings"
	"time"
)

// Hard mode message constants
const (
	HardModeEnabled = "🚨 HARD MODE ENABLED 🚨"
	HardModeWithCustomRuleset = "🚨 OWASP Rules added to custom ruleset 🚨"
)

// BuildRuleSetFromUserSuppliedSet creates a ready to run ruleset, augmented or provided by a user
// configured ruleset. This ruleset could be lifted directly from a Spectral configuration.
func BuildRuleSetFromUserSuppliedSet(rsBytes []byte, rs rulesets.RuleSets) (*rulesets.RuleSet, error) {
	return BuildRuleSetFromUserSuppliedSetWithHTTPClient(rsBytes, rs, nil)
}

// BuildRuleSetFromUserSuppliedSetWithHTTPClient creates a ready to run ruleset, augmented or provided by a user
// configured ruleset with HTTP client support for certificate authentication.
func BuildRuleSetFromUserSuppliedSetWithHTTPClient(rsBytes []byte, rs rulesets.RuleSets, httpClient *http.Client) (*rulesets.RuleSet, error) {

	// load in our user supplied ruleset and try to validate it.
	userRS, userErr := rulesets.CreateRuleSetFromData(rsBytes)
	if userErr != nil {
		pterm.Error.Printf("Unable to parse ruleset file: %s\n", userErr.Error())
		pterm.Println()
		return nil, userErr

	}
	return rs.GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(userRS, httpClient), nil
}

// BuildRuleSetFromUserSuppliedLocation creates a ready to run ruleset from a location (file path or URL)
func BuildRuleSetFromUserSuppliedLocation(rulesetFlag string, rs rulesets.RuleSets, remote bool, httpClient *http.Client) (*rulesets.RuleSet, error) {
	if strings.HasPrefix(rulesetFlag, "http") {
		// Handle remote ruleset URL directly
		if !remote {
			return nil, fmt.Errorf("remote ruleset specified but remote flag is disabled (use --remote=true or -u=true)")
		}
		downloadedRS, rsErr := rulesets.DownloadRemoteRuleSet(context.Background(), rulesetFlag, httpClient)
		if rsErr != nil {
			return nil, rsErr
		}
		return rs.GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(downloadedRS, httpClient), nil
	} else {
		// Handle local ruleset file
		rsBytes, rsErr := os.ReadFile(rulesetFlag)
		if rsErr != nil {
			return nil, rsErr
		}
		return BuildRuleSetFromUserSuppliedSetWithHTTPClient(rsBytes, rs, httpClient)
	}
}

// MergeOWASPRulesToRuleSet merges OWASP rules into the provided ruleset when hard mode is enabled.
// This fixes issue #552 where -z flag was ignored when using -r flag.
// Returns true if OWASP rules were merged, false otherwise.
func MergeOWASPRulesToRuleSet(selectedRS *rulesets.RuleSet, hardModeFlag bool) bool {
	if !hardModeFlag || selectedRS == nil {
		return false
	}
	
	owaspRules := rulesets.GetAllOWASPRules()
	if selectedRS.Rules == nil {
		selectedRS.Rules = make(map[string]*model.Rule)
	}
	
	for k, v := range owaspRules {
		// Add OWASP rule if it doesn't already exist in the custom ruleset
		if selectedRS.Rules[k] == nil {
			selectedRS.Rules[k] = v
		}
	}
	
	return true
}

// RenderTimeAndFiles  will render out the time taken to process a specification, and the size of the file in kb.
// it will also render out how many files were processed.
func RenderTimeAndFiles(timeFlag bool, duration time.Duration, fileSize int64, totalFiles int) {
	if timeFlag {
		pterm.Println()
		l := "milliseconds"
		d := fmt.Sprintf("%d", duration.Milliseconds())
		if duration.Milliseconds() > 1000 {
			l = "seconds"
			d = humanize.FormatFloat("##.##", duration.Seconds())
		}
		pterm.Info.Println(fmt.Sprintf("vacuum took %s %s to lint %s across %d files", d, l,
			index.HumanFileSize(float64(fileSize)), totalFiles))
		pterm.Println()
	}
}

// RenderTime will render out the time taken to process a specification, and the size of the file in kb.
func RenderTime(timeFlag bool, duration time.Duration, fi int64) {
	if timeFlag {
		pterm.Println()
		if (fi / 1000) <= 1024 {
			pterm.Info.Println(fmt.Sprintf("vacuum took %d milliseconds to lint %dkb", duration.Milliseconds(), fi/1000))
		} else {
			pterm.Info.Println(fmt.Sprintf("vacuum took %d milliseconds to lint %dmb", duration.Milliseconds(), fi/1000000))
		}
		pterm.Println()
	}
}

func PrintBanner() {
	pterm.Println()

	//_ = pterm.DefaultBigText.WithLetters(
	//	putils.LettersFromString(pterm.LightMagenta("vacuum"))).Render()
	banner := `
██╗   ██╗ █████╗  ██████╗██╗   ██╗██╗   ██╗███╗   ███╗
██║   ██║██╔══██╗██╔════╝██║   ██║██║   ██║████╗ ████║
██║   ██║███████║██║     ██║   ██║██║   ██║██╔████╔██║
╚██╗ ██╔╝██╔══██║██║     ██║   ██║██║   ██║██║╚██╔╝██║
 ╚████╔╝ ██║  ██║╚██████╗╚██████╔╝╚██████╔╝██║ ╚═╝ ██║
  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝ ╚═╝     ╚═╝
`

	pterm.Println(pterm.LightMagenta(banner))
	pterm.Println()
	pterm.Printf("version: %s | compiled: %s\n", pterm.LightGreen(Version), pterm.LightGreen(Date))
	pterm.Println(pterm.Cyan("🔗 https://quobix.com/vacuum | https://github.com/daveshanley/vacuum"))
	pterm.Println()
	pterm.Println()
}

// LoadCustomFunctions will scan for (and load) custom functions defined as vacuum plugins.
func LoadCustomFunctions(functionsFlag string, silence bool) (map[string]model.RuleFunction, error) {
	// check custom functions
	if functionsFlag != "" {
		pm, err := plugin.LoadFunctions(functionsFlag, silence)
		if err != nil {
			pterm.Error.Printf("Unable to open custom functions: %v\n", err)
			pterm.Println()
			return nil, err
		}
		
		customFunctions := pm.GetCustomFunctions()
		pterm.Info.Printf("Loaded %d custom function(s) successfully.\n", pm.LoadedFunctionCount())
		
		if !silence && len(customFunctions) > 0 {
			pterm.Info.Println("Available custom functions:")
			for funcName := range customFunctions {
				pterm.Printf("  - %s\n", pterm.LightCyan(funcName))
			}
			pterm.Println()
		}
		
		return customFunctions, nil
	}
	return nil, nil
}

func CheckFailureSeverity(failSeverityFlag string, errors int, warnings int, informs int) error {
	if failSeverityFlag == model.SeverityNone {
		return nil
	}
	if failSeverityFlag != model.SeverityError {
		switch failSeverityFlag {
		case model.SeverityWarn:
			if warnings > 0 || errors > 0 {
				return fmt.Errorf("failed with %d errors and %d warnings", errors, warnings)
			}
		case model.SeverityInfo:
			if informs > 0 || warnings > 0 || errors > 0 {
				return fmt.Errorf("failed with %d errors, %d warnings and %d informs",
					errors, warnings, informs)
			}
			return nil
		}
	} else {
		if errors > 0 {
			return fmt.Errorf("failed with %d errors", errors)
		}
	}
	return nil
}
