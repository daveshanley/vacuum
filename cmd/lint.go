// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/dustin/go-humanize"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
	"time"
)

func GetLintCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceErrors: true,
		SilenceUsage:  true,
		Use:           "lint",
		Short:         "Lint an OpenAPI specification",
		Long:          `Lint an OpenAPI specification, the output of the response will be in the terminal`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			detailsFlag, _ := cmd.Flags().GetBool("details")
			timeFlag, _ := cmd.Flags().GetBool("time")
			snippetsFlag, _ := cmd.Flags().GetBool("snippets")
			errorsFlag, _ := cmd.Flags().GetBool("errors")
			categoryFlag, _ := cmd.Flags().GetString("category")
			rulesetFlag, _ := cmd.Flags().GetString("ruleset")
			silent, _ := cmd.Flags().GetBool("silent")
			functionsFlag, _ := cmd.Flags().GetString("functions")
			failSeverityFlag, _ := cmd.Flags().GetString("fail-severity")
			noStyleFlag, _ := cmd.Flags().GetBool("no-style")
			baseFlag, _ := cmd.Flags().GetString("base")

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				pterm.DisableColor()
				pterm.DisableStyling()
			}

			if !silent {
				PrintBanner()
			}

			// check for file args
			if len(args) != 1 {
				pterm.Error.Println("Please supply an OpenAPI specification to lint")
				pterm.Println()
				return fmt.Errorf("no file supplied")
			}

			// read file.
			specBytes, ferr := os.ReadFile(args[0])

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

			customFunctions, _ := LoadCustomFunctions(functionsFlag)

			// if ruleset has been supplied, lets make sure it exists, then load it in
			// and see if it's valid. If so - let's go!
			if rulesetFlag != "" {

				rsBytes, rsErr := os.ReadFile(rulesetFlag)
				if rsErr != nil {
					pterm.Error.Printf("Unable to read ruleset file '%s': %s\n", rulesetFlag, rsErr.Error())
					pterm.Println()
					return rsErr
				}

				selectedRS, rsErr = BuildRuleSetFromUserSuppliedSet(rsBytes, defaultRuleSets)
				if rsErr != nil {
					return rsErr
				}
			}

			pterm.Info.Printf("Linting against %d rules: %s\n", len(selectedRS.Rules), selectedRS.DocumentationURI)
			start := time.Now()
			result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
				RuleSet:         selectedRS,
				Spec:            specBytes,
				CustomFunctions: customFunctions,
				Base:            baseFlag,
			})

			results := result.Results

			if len(result.Errors) > 0 {
				for _, err := range result.Errors {
					pterm.Error.Printf("linting error: %s", err.Error())
					pterm.Println()
				}
				return fmt.Errorf("linting failed due to %d issues", len(result.Errors))
			}

			if !silent {
				pterm.Println()
			} // Blank line

			resultSet := model.NewRuleResultSet(results)
			resultSet.SortResultsByLineNumber()
			fi, _ := os.Stat(args[0])
			duration := time.Since(start)

			warnings := resultSet.GetWarnCount()
			errors := resultSet.GetErrorCount()
			informs := resultSet.GetInfoCount()

			if !detailsFlag {
				RenderSummary(resultSet, silent)
				RenderTime(timeFlag, duration, fi)
				return CheckFailureSeverity(failSeverityFlag, errors, warnings, informs)
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
				case model.CategoryOWASP:
					cats = append(cats, model.RuleCategories[model.CategoryOWASP])
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
					if !silent {
						pterm.DefaultSection.Printf("%s Issues\n", val.Name)
					}
					processResults(categoryResults, specStringData, snippetsFlag, errorsFlag, silent)

				}

			}

			if !silent {
				pterm.Println()
			} // Blank line

			RenderSummary(resultSet, silent)
			RenderTime(timeFlag, duration, fi)
			return CheckFailureSeverity(failSeverityFlag, errors, warnings, informs)
		},
	}

	cmd.Flags().BoolP("details", "d", false, "Show full details of linting report")
	cmd.Flags().BoolP("snippets", "s", false, "Show code snippets where issues are found")
	cmd.Flags().BoolP("errors", "e", false, "Show errors only")
	cmd.Flags().StringP("category", "c", "", "Show a single category of results")
	cmd.Flags().BoolP("silent", "x", false, "Show nothing except the result.")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	cmd.Flags().StringP("fail-severity", "n", model.SeverityError, "Results of this level or above will trigger a failure exit code")

	regErr := cmd.RegisterFlagCompletionFunc("category", cobra.FixedCompletions([]string{
		model.CategoryAll,
		model.CategoryDescriptions,
		model.CategoryExamples,
		model.CategoryInfo,
		model.CategoryOperations,
		model.CategorySchemas,
		model.CategorySecurity,
		model.CategoryTags,
		model.CategoryValidation,
	}, cobra.ShellCompDirectiveNoFileComp))
	if regErr != nil {
		panic(regErr)
	}
	regErr = cmd.RegisterFlagCompletionFunc("fail-severity", cobra.FixedCompletions([]string{
		model.SeverityInfo,
		model.SeverityWarn,
		model.SeverityError,
	}, cobra.ShellCompDirectiveNoFileComp))
	if regErr != nil {
		panic(regErr)
	}

	return cmd
}

func processResults(results []*model.RuleFunctionResult, specData []string, snippets, errors bool, silent bool) {

	// if snippets are being used, we render a single table for a result and then a snippet, if not
	// we just render the entire table, all rows.
	var tableData [][]string
	if !snippets {
		tableData = [][]string{{"Line / Column", "Severity", "Message", "Rule", "Path"}}
	}
	for i, r := range results {

		if i > 200 {
			tableData = append(tableData, []string{"", "", pterm.LightRed(fmt.Sprintf("...%d "+
				"more violations not rendered.", len(results)-200)), ""})
			break
		}
		if snippets {
			tableData = [][]string{{"Line / Column", "Severity", "Message", "Rule", "Path"}}
		}
		startLine := 0
		startCol := 0
		if r.StartNode != nil {
			startLine = r.StartNode.Line
		}
		if r.StartNode != nil {
			startCol = r.StartNode.Column
		}
		start := fmt.Sprintf("(%v:%v)", startLine, startCol)
		m := r.Message
		p := r.Path
		if len(r.Path) > 60 {
			p = fmt.Sprintf("%s...", r.Path[:60])
		}

		if len(r.Message) > 180 {
			m = fmt.Sprintf("%s...", r.Message[:180])
		}

		sev := "nope"
		if r.Rule != nil {
			sev = r.Rule.Severity
		}

		switch sev {
		case model.SeverityError:
			sev = pterm.LightRed(sev)
		case model.SeverityWarn:
			sev = pterm.LightYellow("warning")
		case model.SeverityInfo:
			sev = pterm.LightBlue(sev)
		}

		if errors && r.Rule.Severity != model.SeverityError {
			continue // only show errors
		}

		tableData = append(tableData, []string{start, sev, m, r.Rule.Id, p})

		if snippets && !silent {
			_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
			renderCodeSnippet(r, specData)
		}
	}

	if !snippets && !silent {
		_ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
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

func RenderSummary(rs *model.RuleResultSet, silent bool) {

	tableData := [][]string{{"Category", pterm.LightRed("Errors"), pterm.LightYellow("Warnings"),
		pterm.LightBlue("Info")}}

	for _, cat := range model.RuleCategoriesOrdered {
		errors := rs.GetErrorsByRuleCategory(cat.Id)
		warn := rs.GetWarningsByRuleCategory(cat.Id)
		info := rs.GetInfoByRuleCategory(cat.Id)

		if len(errors) > 0 || len(warn) > 0 || len(info) > 0 {

			tableData = append(tableData, []string{cat.Name, fmt.Sprintf("%v", humanize.Comma(int64(len(errors)))),
				fmt.Sprintf("%v", humanize.Comma(int64(len(warn)))), fmt.Sprintf("%v", humanize.Comma(int64(len(info))))})
		}

	}

	if len(rs.Results) > 0 {
		if !silent {
			err := pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
			if err != nil {
				pterm.Error.Printf("error rendering table '%v'", err.Error())
			}
			pterm.Println()
			pterm.Println()
		}
	}

	errors := rs.GetErrorCount()
	warnings := rs.GetWarnCount()
	informs := rs.GetInfoCount()
	errorsHuman := humanize.Comma(int64(rs.GetErrorCount()))
	warningsHuman := humanize.Comma(int64(rs.GetWarnCount()))
	informsHuman := humanize.Comma(int64(rs.GetInfoCount()))

	if errors > 0 {
		pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgRed)).WithMargin(10).Printf(
			"Linting failed with %v errors, %v warnings and %v informs", errorsHuman, warningsHuman, informsHuman)
		return
	}
	if warnings > 0 {
		pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).WithMargin(10).Printf(
			"Linting passed, but with %v warnings and %v informs", warningsHuman, informsHuman)
		return
	}

	if informs > 0 {
		pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgGreen)).WithMargin(10).Printf(
			"Linting passed, %v informs reported", informsHuman)
	}

	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgGreen)).WithMargin(10).Println(
		"Linting passed, A perfect score! well done!")

}
