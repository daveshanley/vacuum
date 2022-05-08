package cui

import (
	"fmt"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"io/ioutil"
	"os"
	"time"
)

// BuildRuleSetFromUserSuppliedSet creates a ready to run ruleset, augmented or provided by a user
// configured ruleset. This ruleset could be lifted directly from a Spectral configuration.
func BuildRuleSetFromUserSuppliedSet(rulesetFilePath string, rs rulesets.RuleSets) (*rulesets.RuleSet, error) {
	rsBytes, rsErr := ioutil.ReadFile(rulesetFilePath)

	if rsErr != nil {

		pterm.Error.Printf("Unable to read ruleset file '%s': %s\n", rulesetFilePath, rsErr.Error())
		pterm.Println()
		return nil, rsErr
	}

	// load in our user supplied ruleset and try to validate it.
	userRS, userErr := rulesets.CreateRuleSetFromData(rsBytes)
	if userErr != nil {
		pterm.Error.Printf("Unable to parse ruleset file '%s': %s\n", rulesetFilePath, userErr.Error())
		pterm.Println()
		return nil, userErr

	}
	return rs.GenerateRuleSetFromSuppliedRuleSet(userRS), nil
}

// RenderTime will render out the time taken to process a specification, and the size of the file in kb.
func RenderTime(timeFlag bool, duration time.Duration, fi os.FileInfo) {
	if timeFlag {
		pterm.Println()
		pterm.Info.Println(fmt.Sprintf("Vacuum took %d milliseconds to lint %dkb", duration.Milliseconds(), fi.Size()/1000))
		pterm.Println()
	}
}
