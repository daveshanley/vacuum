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
			b, ferr := ioutil.ReadFile(args[0])

			if ferr != nil {
				pterm.Error.Printf("Unable to read file '%s': %s\n", args[0], ferr.Error())
				pterm.Println()
				return ferr
			}

			// read spec and parse to dashboard.
			rs := rulesets.BuildDefaultRuleSets()
			results, _ := motor.ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), b)

			resultSet := model.NewRuleResultSet(results)
			resultSet.SortResultsByLineNumber()

			dash := cui.CreateDashboard(resultSet)
			dash.Render()
			return nil
		},
	}

}
