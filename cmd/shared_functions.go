// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/dustin/go-humanize"
	"github.com/pb33f/libopenapi/index"
	"github.com/pterm/pterm"
)

// Hard mode message constants
const (
	HardModeEnabled           = "🚨 HARD MODE ENABLED 🚨"
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

func formatFileLocation(r *model.RuleFunctionResult, fileName string) string {
	startLine := 0
	startCol := 0
	f := fileName

	if r.StartNode != nil {
		startLine = r.StartNode.Line
		startCol = r.StartNode.Column
	}

	if r.Origin != nil {
		f = r.Origin.AbsoluteLocation
		startLine = r.Origin.Line
		startCol = r.Origin.Column
	}

	// Make path relative
	if absPath, err := filepath.Abs(f); err == nil {
		if cwd, err := os.Getwd(); err == nil {
			if relPath, err := filepath.Rel(cwd, absPath); err == nil {
				f = relPath
			}
		}
	}

	return fmt.Sprintf("%s:%d:%d", f, startLine, startCol)
}

func getRuleSeverity(r *model.RuleFunctionResult) string {
	if r.Rule != nil {
		switch r.Rule.Severity {
		case model.SeverityError:
			return "✗ error"
		case model.SeverityWarn:
			return "▲ warning"
		default:
			return "● info"
		}
	}
	return "● info"
}

func getLintingFilterName(state FilterState) string {
	switch state {
	case FilterAll:
		return "All"
	case FilterErrors:
		return "Errors"
	case FilterWarnings:
		return "Warnings"
	case FilterInfo:
		return "Info"
	default:
		return "All"
	}
}

func extractCategories(results []*model.RuleFunctionResult) []string {
	categoryMap := make(map[string]bool)
	for _, r := range results {
		if r.Rule != nil && r.Rule.RuleCategory != nil {
			categoryMap[r.Rule.RuleCategory.Name] = true
		}
	}

	categories := make([]string, 0, len(categoryMap))
	for cat := range categoryMap {
		categories = append(categories, cat)
	}

	for i := 0; i < len(categories); i++ {
		for j := i + 1; j < len(categories); j++ {
			if categories[i] > categories[j] {
				categories[i], categories[j] = categories[j], categories[i]
			}
		}
	}

	return categories
}

func extractRules(results []*model.RuleFunctionResult) []string {
	ruleMap := make(map[string]bool)
	for _, r := range results {
		if r.Rule != nil && r.Rule.Id != "" {
			ruleMap[r.Rule.Id] = true
		}
	}

	rules := make([]string, 0, len(ruleMap))
	for rule := range ruleMap {
		rules = append(rules, rule)
	}

	// sort rules
	for i := 0; i < len(rules); i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i] > rules[j] {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}

	return rules
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) <= width {
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}
