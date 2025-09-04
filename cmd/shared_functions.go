// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
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

// ShortcodeHandler defines how to transform a shortcode into markdown
type ShortcodeHandler struct {
	// Pattern is the regex pattern to match the shortcode
	Pattern *regexp.Regexp
	// Transform is the function that converts matched shortcode to markdown
	Transform func(matches []string) string
}

// ShortcodeParser holds the configuration for parsing shortcodes
type ShortcodeParser struct {
	handlers []ShortcodeHandler
}

// NewShortcodeParser creates a new shortcode parser with default handlers
func NewShortcodeParser() *ShortcodeParser {
	return &ShortcodeParser{
		handlers: []ShortcodeHandler{
			// warn-box shortcode handler
			{
				Pattern: regexp.MustCompile(`{{<\s*warn-box\s*>}}([\s\S]*?){{</\s*warn-box\s*>}}`),
				Transform: func(matches []string) string {
					if len(matches) > 1 {
						content := strings.TrimSpace(matches[1])
						return fmt.Sprintf("**\u21e8\u21e8\u21e8 WARNING \u21e6\u21e6\u21e6** \n\n%s\n", content)
					}
					return ""
				},
			},

			// Generic shortcode with parameters (e.g., {{< shortcode param="value" >}})
			{
				Pattern: regexp.MustCompile(`{{<\s*(\w+)\s+([^>]+?)\s*>}}`),
				Transform: func(matches []string) string {
					if len(matches) > 2 {
						shortcodeName := matches[1]
						params := matches[2]
						return fmt.Sprintf("==[%s: %s]==", strings.ToUpper(shortcodeName), params)
					}
					return ""
				},
			},
			// Simple shortcode without content (e.g., {{< br >}} or {{< hr >}})
			{
				Pattern: regexp.MustCompile(`{{<\s*(br|hr)\s*>}}`),
				Transform: func(matches []string) string {
					if len(matches) > 1 {
						switch matches[1] {
						case "br":
							return "\n"
						case "hr":
							return "\n---\n"
						}
					}
					return ""
				},
			},
		},
	}
}

// AddHandler adds a custom shortcode handler to the parser
func (p *ShortcodeParser) AddHandler(pattern *regexp.Regexp, transform func([]string) string) {
	p.handlers = append(p.handlers, ShortcodeHandler{
		Pattern:   pattern,
		Transform: transform,
	})
}

// Parse processes the input text and replaces all shortcodes with their markdown equivalents
func (p *ShortcodeParser) Parse(input string) string {
	result := input

	// Process each handler in order
	for _, handler := range p.handlers {
		matches := handler.Pattern.FindAllStringSubmatch(result, -1)
		for _, match := range matches {
			replacement := handler.Transform(match)
			result = strings.Replace(result, match[0], replacement, 1)
		}
	}

	return result
}

// ConvertHugoShortcodesToMarkdown is a convenience function that converts Hugo shortcodes to markdown
// with highlight syntax using the default parser configuration
func ConvertHugoShortcodesToMarkdown(content string) string {
	parser := NewShortcodeParser()
	return parser.Parse(content)
}

// ConvertHugoShortcodesToMarkdownWithCustomHandlers allows adding custom handlers before parsing
func ConvertHugoShortcodesToMarkdownWithCustomHandlers(content string, customHandlers []ShortcodeHandler) string {
	parser := NewShortcodeParser()

	// Add custom handlers
	for _, handler := range customHandlers {
		parser.handlers = append(parser.handlers, handler)
	}

	return parser.Parse(content)
}

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

func renderEmptyState(width, height int) string {
	art := []string{
		"",
		" _|      _|     _|_|     _|_|_|_|_|   _|    _|   _|_|_|   _|      _|     _|_|_|  ",
		" _|_|    _|   _|    _|       _|       _|    _|     _|     _|_|    _|   _|        ",
		" _|  _|  _|   _|    _|       _|       _|_|_|_|     _|     _|  _|  _|   _|  _|_|  ",
		" _|    _|_|   _|    _|       _|       _|    _|     _|     _|    _|_|   _|    _|  ",
		" _|      _|     _|_|         _|       _|    _|   _|_|_|   _|      _|     _|_|_|  ",
		"",
		" _|    _|   _|_|_|_|   _|_|_|     _|_|_|_|  ",
		" _|    _|   _|         _|    _|   _|        ",
		" _|_|_|_|   _|_|_|     _|_|_|     _|_|_|    ",
		" _|    _|   _|         _|    _|   _|        ",
		" _|    _|   _|_|_|_|   _|    _|   _|_|_|_|  ",
		"",
		" Nothing to vacuum, the filters are too strict.",
		"",
		" To adjust them:",
		"",
		" > tab - cycle severity",
		" > c   - cycle categories",
		" > r   - cycle rules",
		" > esc - clear all filters",
	}

	artStr := strings.Join(art, "\n")

	maxLineWidth := 82 // width of the longest line in the art
	leftPadding := (width - maxLineWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	// add left padding to each line to center the entire block
	artLines := strings.Split(artStr, "\n")
	paddedLines := make([]string, len(artLines))
	padding := strings.Repeat(" ", leftPadding)
	for i, line := range artLines {
		if line != "" {
			paddedLines[i] = padding + line
		} else {
			paddedLines[i] = ""
		}
	}

	// calculate vertical centering
	totalLines := len(paddedLines)
	topPadding := (height - totalLines) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// build the result to exactly fill the height
	var resultLines []string
	for i := 0; i < topPadding; i++ {
		resultLines = append(resultLines, "")
	}

	// content
	resultLines = append(resultLines, paddedLines...)

	// bottom padding to exactly fill the height
	for len(resultLines) < height {
		resultLines = append(resultLines, "")
	}

	// ensure we don't exceed the height
	if len(resultLines) > height {
		resultLines = resultLines[:height]
	}

	textStyle := lipgloss.NewStyle().
		Foreground(RGBRed).
		Width(width)
	return textStyle.Render(strings.Join(resultLines, "\n"))
}

func addTableBorders(tableView string) string {
	tableStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(RGBPink).
		PaddingTop(0)

	return tableStyle.Render(tableView)
}

// fetchDocsFromDoctorAPI creates a command to fetch documentation for a rule from the doctor API.
func fetchDocsFromDoctorAPI(ruleID string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		url := fmt.Sprintf("https://localhost:9090/rules/documentation/%s?markdown", ruleID)
		resp, err := client.Get(url)
		if err != nil {
			return docsErrorMsg{ruleID: ruleID, err: err.Error(), is404: false}
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			return docsErrorMsg{ruleID: ruleID, err: "Documentation not found", is404: true}
		}

		if resp.StatusCode != 200 {
			return docsErrorMsg{ruleID: ruleID, err: fmt.Sprintf("HTTP %d", resp.StatusCode), is404: false}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return docsErrorMsg{ruleID: ruleID, err: err.Error(), is404: false}
		}

		var docResponse struct {
			RuleID   string `json:"ruleId"`
			Category string `json:"category"`
			Body     string `json:"body"`
		}

		if err := json.Unmarshal(body, &docResponse); err != nil {
			return docsErrorMsg{ruleID: ruleID, err: fmt.Sprintf("Failed to parse JSON: %s", err.Error()), is404: false}
		}

		// process shortcodes in docs
		processedContent := ConvertHugoShortcodesToMarkdown(docResponse.Body)

		return docsLoadedMsg{ruleID: ruleID, content: processedContent}
	}
}
