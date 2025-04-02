// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/dustin/go-humanize"
	"github.com/pb33f/libopenapi/index"
	"os"
	"time"
)

// BuildRuleSetFromUserSuppliedSet creates a ready to run ruleset, augmented or provided by a user
// configured ruleset. This ruleset could be lifted directly from a Spectral configuration.
func BuildRuleSetFromUserSuppliedSet(rsBytes []byte, rs rulesets.RuleSets) (*rulesets.RuleSet, error) {

	// load in our user supplied ruleset and try to validate it.
	userRS, userErr := rulesets.CreateRuleSetFromData(rsBytes)
	if userErr != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse ruleset file: %s\n", userErr.Error())
		fmt.Println()
		return nil, userErr

	}
	return rs.GenerateRuleSetFromSuppliedRuleSet(userRS), nil
}

// RenderTimeAndFiles  will render out the time taken to process a specification, and the size of the file in kb.
// it will also render out how many files were processed.
func RenderTimeAndFiles(timeFlag bool, duration time.Duration, fileSize int64, totalFiles int) {
	if timeFlag {
		fmt.Println()
		l := "milliseconds"
		d := fmt.Sprintf("%d", duration.Milliseconds())
		if duration.Milliseconds() > 1000 {
			l = "seconds"
			d = humanize.FormatFloat("##.##", duration.Seconds())
		}
		fmt.Printf("vacuum took %s %s to lint %s across %d files\n", d, l,
			index.HumanFileSize(float64(fileSize)), totalFiles)
		fmt.Println()
	}
}

// RenderTime will render out the time taken to process a specification, and the size of the file in kb.
func RenderTime(timeFlag bool, duration time.Duration, fi int64) {
	if timeFlag {
		fmt.Println()
		if (fi / 1000) <= 1024 {
			fmt.Printf("vacuum took %d milliseconds to lint %dkb\n", duration.Milliseconds(), fi/1000)
		} else {
			fmt.Printf("vacuum took %d milliseconds to lint %dmb\n", duration.Milliseconds(), fi/1000000)
		}
		fmt.Println()
	}
}

func PrintBanner() {
	fmt.Println()

	banner := `
██╗   ██╗ █████╗  ██████╗██╗   ██╗██╗   ██╗███╗   ███╗
██║   ██║██╔══██╗██╔════╝██║   ██║██║   ██║████╗ ████║
██║   ██║███████║██║     ██║   ██║██║   ██║██╔████╔██║
╚██╗ ██╔╝██╔══██║██║     ██║   ██║██║   ██║██║╚██╔╝██║
 ╚████╔╝ ██║  ██║╚██████╗╚██████╔╝╚██████╔╝██║ ╚═╝ ██║
  ╚═══╝  ╚═╝  ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝ ╚═╝     ╚═╝
`

	fmt.Println(banner)
	fmt.Println()
	fmt.Printf("version: %s | compiled: %s\n", Version, Date)
	fmt.Println("🔗 https://quobix.com/vacuum | https://github.com/daveshanley/vacuum")
	fmt.Println()
	fmt.Println()
}

// LoadCustomFunctions will scan for (and load) custom functions defined as vacuum plugins.
func LoadCustomFunctions(functionsFlag string, silence bool) (map[string]model.RuleFunction, error) {
	// check custom functions
	if functionsFlag != "" {
		pm, err := plugin.LoadFunctions(functionsFlag, silence)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to open custom functions: %v\n", err)
			fmt.Println()
			return nil, err
		}
		fmt.Printf("Loaded %d custom function(s) successfully.\n", pm.LoadedFunctionCount())
		return pm.GetCustomFunctions(), nil
	}
	return nil, nil
}

func CheckFailureSeverity(failSeverityFlag string, errors int, warnings int, informs int) error {
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
