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
	"net/url"
	"time"
)

func GetDashboardCommand() *cobra.Command {

	return &cobra.Command{
		Use:     "dashboard",
		Short:   "Show vacuum dashboard for linting report",
		Long:    "Interactive console dashboard to explore linting report in detail",
		Example: "vacuum dashboard my-awesome-spec.yaml",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			// check for file args
			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate a report"
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}
			baseFlag, _ := cmd.Flags().GetString("base")
			skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")

			var err error
			vacuumReport, specBytes, _ := vacuum_report.BuildVacuumReportFromFile(args[0])
			if len(specBytes) <= 0 {
				pterm.Error.Printf("Failed to read specification: %v\n\n", args[0])
				return err
			}

			var resultSet *model.RuleResultSet
			var ruleset *motor.RuleSetExecutionResult
			var specIndex *index.SpecIndex
			var specInfo *datamodel.SpecInfo

			// if we have a pre-compiled report, jump straight to the end and collect $500
			if vacuumReport == nil {

				functionsFlag, _ := cmd.Flags().GetString("functions")
				customFunctions, _ := LoadCustomFunctions(functionsFlag)

				rulesetFlag, _ := cmd.Flags().GetString("ruleset")
				resultSet, ruleset, err = BuildResultsWithDocCheckSkip(rulesetFlag, specBytes, customFunctions, baseFlag, skipCheckFlag)
				if err != nil {
					pterm.Error.Printf("Failed to render dashboard: %v\n\n", err)
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

				config := index.CreateClosedAPIIndexConfig()
				if baseFlag != "" {
					u, e := url.Parse(baseFlag)
					if e == nil && u.Scheme != "" && u.Host != "" {
						config.BaseURL = u
						config.BasePath = ""
					} else {
						config.BasePath = baseFlag
					}
					config.AllowFileLookup = true
					config.AllowRemoteLookup = true
				}

				specIndex = index.NewSpecIndexWithConfig(&rootNode, config)

				specInfo = vacuumReport.SpecInfo
				specInfo.Generated = vacuumReport.Generated
			}

			dash := cui.CreateDashboard(resultSet, specIndex, specInfo)
			dash.Version = Version
			return dash.Render()
		},
	}
}
