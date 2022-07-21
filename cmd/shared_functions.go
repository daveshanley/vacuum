// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"os"
	"time"
)

// BuildRuleSetFromUserSuppliedSet creates a ready to run ruleset, augmented or provided by a user
// configured ruleset. This ruleset could be lifted directly from a Spectral configuration.
func BuildRuleSetFromUserSuppliedSet(rsBytes []byte, rs rulesets.RuleSets) (*rulesets.RuleSet, error) {

	// load in our user supplied ruleset and try to validate it.
	userRS, userErr := rulesets.CreateRuleSetFromData(rsBytes)
	if userErr != nil {
		pterm.Error.Printf("Unable to parse ruleset file: %s\n", userErr.Error())
		pterm.Println()
		return nil, userErr

	}
	return rs.GenerateRuleSetFromSuppliedRuleSet(userRS), nil
}

// RenderTime will render out the time taken to process a specification, and the size of the file in kb.
func RenderTime(timeFlag bool, duration time.Duration, fi os.FileInfo) {
	if timeFlag {
		pterm.Println()
		pterm.Info.Println(fmt.Sprintf("vacuum took %d milliseconds to lint %dkb", duration.Milliseconds(), fi.Size()/1000))
		pterm.Println()
	}
}

func PrintBanner() {
	pterm.Println()
	_ = pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithRGB("vacuum", pterm.NewRGB(153, 51, 255))).Render()
	pterm.Printf("version: %s | compiled: %s\n\n", Version, Date)
	pterm.Println()
}

// LoadCustomFunctions will scan for (and load) custom functions defined as vacuum plugins.
func LoadCustomFunctions(functionsFlag string) (map[string]model.RuleFunction, error) {
	// check custom functions
	if functionsFlag != "" {
		pm, err := plugin.LoadFunctions(functionsFlag)
		if err != nil {
			pterm.Error.Printf("Unable to open custom functions: %v\n", err)
			pterm.Println()
			return nil, err
		}
		pterm.Info.Printf("Loaded %d custom function(s) successfully.\n", pm.LoadedFunctionCount())
		return pm.GetCustomFunctions(), nil
	}
	return nil, nil
}

func CheckFailureSeverity(failSeverityFlag string, errors int, warnings int, informs int) error {
	if failSeverityFlag != "error" {
		switch failSeverityFlag {
		case "warn":
			if errors > 0 && warnings > 0 {
				return fmt.Errorf("failed linting, with %d errors and %d warnings", errors, warnings)
			}
			return nil
		case "info":
			if errors > 0 && warnings > 0 && informs > 0 {
				return fmt.Errorf("failed linting, with %d errors, %d warnings and %d informs",
					errors, warnings, informs)
			}
			return nil
		}
	} else {
		if errors > 0 {
			return fmt.Errorf("failed linting, with %d errors", errors)
		}
	}
	return nil
}
