// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"io/ioutil"
)

func GetDashboardCommand() *cobra.Command {

	return &cobra.Command{
		Use:     "dashboard",
		Short:   "Show vacuum dashboard for linting report",
		Long:    "Interactive console dashboard to explore linting report in detail",
		Example: "vacuum dashboard my-awesome-spec.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {

			// check for file args
			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate a report"
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}

			// read file.
			specBytes, fileError := ioutil.ReadFile(args[0])

			if fileError != nil {
				pterm.Error.Printf("Unable to read file '%s': %s\n", args[0], fileError.Error())
				pterm.Println()
				return fileError
			}

			rulesetFlag, _ := cmd.Flags().GetString("ruleset")

			// read spec and parse to dashboard.
			defaultRuleSets := rulesets.BuildDefaultRuleSets()

			// default is recommended rules, based on spectral (for now anyway)
			selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

			// if ruleset has been supplied, lets make sure it exists, then load it in
			// and see if it's valid. If so - let's go!
			if rulesetFlag != "" {
				var rsErr error
				selectedRS, rsErr = cui.BuildRuleSetFromUserSuppliedSet(rulesetFlag, defaultRuleSets)
				if rsErr != nil {
					return rsErr
				}
			}

			pterm.Info.Printf("Running vacuum against spec '%s' against %d rules: %s\n\n%s\n", args[0],
				len(selectedRS.Rules), selectedRS.DocumentationURI, selectedRS.Description)
			pterm.Println()

			result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
				RuleSet: selectedRS,
				Spec:    specBytes,
			})

			resultSet := model.NewRuleResultSet(result.Results)
			resultSet.SortResultsByLineNumber()

			dash := cui.CreateDashboard(resultSet, result.Index, result.SpecInfo)
			dash.Render()
			return nil
		},
	}
}
