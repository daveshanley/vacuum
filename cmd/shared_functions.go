// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
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
