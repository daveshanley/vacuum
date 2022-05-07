package cui

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

func GetLintCommand() *cobra.Command {

	return &cobra.Command{
		Use:   "lint",
		Short: "vacuum is a very, very fast OpenAPI linter",
		Long:  `vacuum is a very, very fast OpenAPI linter. It will suck all the lint off your spec in milliseconds`,
		RunE: func(cmd *cobra.Command, args []string) error {

			detailsFlag, _ := cmd.Flags().GetBool("details")
			timeFlag, _ := cmd.Flags().GetBool("time")
			snippetsFlag, _ := cmd.Flags().GetBool("snippets")
			errorsFlag, _ := cmd.Flags().GetBool("errors")
			categoryFlag, _ := cmd.Flags().GetString("category")
			rulesetFlag, _ := cmd.Flags().GetString("ruleset")

			pterm.Println()

			pterm.DefaultBigText.WithLetters(
				pterm.NewLettersFromStringWithRGB("vacuum", pterm.NewRGB(153, 51, 255))).
				Render()

			pterm.Println()

			// check for file args
			if len(args) != 1 {
				pterm.Error.Println("please supply OpenAPI specification(s) to lint")
				pterm.Println()
				return fmt.Errorf("no files supplied")
			}

			start := time.Now()

			// read file.
			specBytes, ferr := ioutil.ReadFile(args[0])

			// split up file into an array with lines.
			specStringData := strings.Split(string(specBytes), "\n")

			if ferr != nil {

				pterm.Error.Printf("Unable to read file '%s': %s\n", args[0], ferr.Error())
				pterm.Println()
				return ferr

			}

			defaultRuleSets := rulesets.BuildDefaultRuleSets()

			// default is recommended rules, based on spectral (for now anyway)
			selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

			// if ruleset has been supplied, lets make sure it exists, then load it in
			// and see if it's valid. If so - let's go!
			if rulesetFlag != "" {

				// read spec
				rsBytes, rsErr := ioutil.ReadFile(rulesetFlag)

				if rsErr != nil {

					pterm.Error.Printf("Unable to read ruleset file '%s': %s\n", rulesetFlag, rsErr.Error())
					pterm.Println()
					return ferr
				}

				// load in our user supplied ruleset and try to validate it.
				userRS, userErr := rulesets.CreateRuleSetFromData(rsBytes)
				if userErr != nil {

					pterm.Error.Printf("Unable to parse ruleset file '%s': %s\n", rulesetFlag, userErr.Error())
					pterm.Println()
					return ferr

				}
				selectedRS = defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(userRS)

			}

			pterm.Info.Printf("Running vacuum against spec '%s' against %d rules: %s\n\n%s\n", args[0],
				len(selectedRS.Rules), selectedRS.DocumentationURI, selectedRS.Description)
			pterm.Println()

			result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
				RuleSet: selectedRS,
				Spec:    specBytes,
			})

			results := result.Results

			if len(result.Errors) > 0 {
				for _, err := range result.Errors {
					pterm.Error.Printf("linting error: %s", err.Error())
					pterm.Println()
				}
				return fmt.Errorf("linting failed due to %d issues", len(result.Errors))
			}

			pterm.Println() // Blank line

			resultSet := model.NewRuleResultSet(results)
			resultSet.SortResultsByLineNumber()
			fi, _ := os.Stat(args[0])
			duration := time.Since(start)

			if !detailsFlag {
				RenderSummary(resultSet, args)
				renderTime(timeFlag, duration, fi)
				return nil
			}

			var cats []*model.RuleCategory

			if categoryFlag != "" {
				switch categoryFlag {
				case model.CategoryDescriptions:
					cats = append(cats, model.RuleCategories[model.CategoryDescriptions])
				case model.CategoryExamples:
					cats = append(cats, model.RuleCategories[model.CategoryExamples])
				case model.CategoryInfo:
					cats = append(cats, model.RuleCategories[model.CategoryInfo])
				case model.CategorySchemas:
					cats = append(cats, model.RuleCategories[model.CategorySchemas])
				case model.CategorySecurity:
					cats = append(cats, model.RuleCategories[model.CategorySecurity])
				case model.CategoryValidation:
					cats = append(cats, model.RuleCategories[model.CategoryValidation])
				case model.CategoryOperations:
					cats = append(cats, model.RuleCategories[model.CategoryOperations])
				case model.CategoryTags:
					cats = append(cats, model.RuleCategories[model.CategoryTags])
				default:
					cats = model.RuleCategoriesOrdered
				}
			} else {
				cats = model.RuleCategoriesOrdered
			}

			// try a category print out.
			for _, val := range cats {

				categoryResults := resultSet.GetResultsByRuleCategory(val.Id)

				if len(categoryResults) > 0 {
					pterm.DefaultSection.Printf("%s Issues\n", val.Name)
					processResults(categoryResults, specStringData, snippetsFlag, errorsFlag, val.Name)

				}

			}

			pterm.Println() // Blank line

			renderTime(timeFlag, duration, fi)

			return nil
		},
	}
}

func renderTime(timeFlag bool, duration time.Duration, fi os.FileInfo) {
	if timeFlag {
		pterm.Println()
		pterm.Info.Println(fmt.Sprintf("Vacuum took %d milliseconds to lint %dkb", duration.Milliseconds(), fi.Size()/1000))
		pterm.Println()
	}
}

func processResults(results []*model.RuleFunctionResult, specData []string, snippets, errors bool, cat string) {

	// if snippets are being used, we render a single table for a result and then a snippet, if not
	// we just render the entire table, all rows.
	var tableData [][]string
	if !snippets {
		tableData = [][]string{{"Line / Column", "Severity", "Message", "Path"}}
	}
	for _, r := range results {

		if snippets {
			tableData = [][]string{{"Line / Column", "Severity", "Message", "Path"}}
		}
		start := fmt.Sprintf("(%v:%v)", r.StartNode.Line, r.StartNode.Column)

		m := r.Message
		p := r.Path
		if len(r.Path) > 60 {
			p = fmt.Sprintf("%s...", r.Path[:60])
		}

		if len(r.Message) > 100 {
			m = fmt.Sprintf("%s...", r.Message[:100])
		}

		sev := "nope"
		if r.Rule != nil {
			sev = r.Rule.Severity
		}

		switch sev {
		case "error":
			sev = pterm.LightRed(sev)
		case "warn":
			sev = pterm.LightYellow("warning")
		case "info":
			sev = pterm.LightBlue(sev)
		}

		if errors && r.Rule.Severity != "error" {
			continue // only show errors
		}

		tableData = append(tableData, []string{start, sev, m, p})

		if snippets {
			pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
			renderCodeSnippet(r, specData)
		}
	}

	if !snippets {
		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	}

}

func renderCodeSnippet(r *model.RuleFunctionResult, specData []string) {
	// render out code snippet

	if r.StartNode.Line-3 >= 0 {
		pterm.Printf("\n\n%s %s %s\n", pterm.Gray(r.StartNode.Line-3), pterm.Gray("|"), specData[r.StartNode.Line-3])
	} else {
		pterm.Printf("\n\n")
	}

	if r.StartNode.Line-2 >= 1 {
		pterm.Printf("%s %s %s\n", pterm.Gray(r.StartNode.Line-2), pterm.Gray("|"), specData[r.StartNode.Line-2])
	}
	if r.StartNode.Line-1 >= 2 {
		pterm.Printf("%s %s %s\n", pterm.LightRed(strconv.Itoa(r.StartNode.Line-1)),
			pterm.Gray("|"), pterm.LightRed(specData[r.StartNode.Line-1]))
	}
	pterm.Printf("%s %s %s\n", pterm.Gray(r.StartNode.Line), pterm.Gray("|"), specData[r.StartNode.Line])

	if r.StartNode.Line+1 <= len(specData) {
		pterm.Printf("%s %s %s\n\n\n", pterm.Gray(r.StartNode.Line+1), pterm.Gray("|"), specData[r.StartNode.Line+1])
	}
}

func RenderSummary(rs *model.RuleResultSet, args []string) {

	tableData := [][]string{{"Category", pterm.LightRed("Errors"), pterm.LightYellow("Warnings"),
		pterm.LightBlue("Info")}}

	for _, cat := range model.RuleCategoriesOrdered {
		errors := rs.GetErrorsByRuleCategory(cat.Id)
		warn := rs.GetWarningsByRuleCategory(cat.Id)
		info := rs.GetInfoByRuleCategory(cat.Id)

		if len(errors) > 0 || len(warn) > 0 || len(info) > 0 {

			tableData = append(tableData, []string{cat.Name, fmt.Sprintf("%d", len(errors)),
				fmt.Sprintf("%d", len(warn)), fmt.Sprintf("%d", len(info))})
		}

	}

	if len(rs.Results) > 0 {
		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
		pterm.Println()
		pterm.Printf(">> run 'vacuum %s -d' to see full details", args[0])
		pterm.Println()
		pterm.Println()
	}

	if rs.GetErrorCount() > 0 {
		pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgRed)).WithMargin(10).Printf(
			"Linting failed with %d errors", rs.GetErrorCount())
		return
	}
	if rs.GetWarnCount() > 0 {
		pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).WithMargin(10).Printf(
			"Linting passed, but with %d warnings", rs.GetWarnCount())
		return
	}

	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgGreen)).WithMargin(10).Println(
		"Linting passed, great job!")

}
