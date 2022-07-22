// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"github.com/daveshanley/vacuum/cui"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"time"
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

			var err error
			vacuumReport, specBytes, _ := vacuum_report.BuildVacuumReportFromFile(args[0])

			var resultSet *model.RuleResultSet
			var ruleset *motor.RuleSetExecutionResult
			var specIndex *index.SpecIndex
			var specInfo *datamodel.SpecInfo

			// if we have a pre-compiled report, jump straight to the end and collect $500
			if vacuumReport == nil {

				functionsFlag, _ := cmd.Flags().GetString("functions")
				customFunctions, _ := LoadCustomFunctions(functionsFlag)

				rulesetFlag, _ := cmd.Flags().GetString("ruleset")
				resultSet, ruleset, err = BuildResults(rulesetFlag, specBytes, customFunctions)
				if err != nil {
					return err
				}
				specIndex = ruleset.Index
				specInfo = ruleset.SpecInfo
				specInfo.Generated = time.Now()

			} else {

				resultSet = model.NewRuleResultSetPointer(vacuumReport.ResultSet.Results)

				// TODO: refactor dashboard to hold state and rendering as separate entities.
				// dashboard will be slower because it needs an index
				var rootNode yaml.Node
				err = yaml.Unmarshal(*vacuumReport.SpecInfo.SpecBytes, &rootNode)
				if err != nil {
					pterm.Error.Printf("Unable to read spec bytes from report file '%s': %s\n", args[0], err.Error())
					pterm.Println()
					return err
				}
				specIndex = index.NewSpecIndex(&rootNode)
				specInfo = vacuumReport.SpecInfo
				specInfo.Generated = vacuumReport.Generated
			}

			dash := cui.CreateDashboard(resultSet, specIndex, specInfo)
			dash.Version = Version
			return dash.Render()
		},
	}
}
