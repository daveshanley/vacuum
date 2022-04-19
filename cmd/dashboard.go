package cmd

import (
	"errors"
	"github.com/daveshanley/vacuum/cui"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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

			dash := new(cui.Dashboard)
			dash.Render()
			return nil
		},
	}

}
