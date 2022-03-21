package cmd

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var (
	// Used for flags.
	cfgFile     string
	userLicense string

	rootCmd = &cobra.Command{
		Use:   "vacuum <your-openapi-file.yaml>",
		Short: "vacuum is a very fast OpenAPI linter",
		Long:  `vacuum is a very fast OpenAPI linter. It will suck all the lint off your spec in milliseconds`,
		RunE: func(cmd *cobra.Command, args []string) error {

			detailsFlag, _ := cmd.Flags().GetBool("details")
			timeFlag, _ := cmd.Flags().GetBool("time")

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
			b, ferr := ioutil.ReadFile(args[0])

			if ferr != nil {

				pterm.Error.Printf("Unable to read file '%s': %s\n", args[0], ferr.Error())
				pterm.Println()
				return ferr

			}

			pterm.Info.Printf("Running vacuum against spec '%s'\n", args[0])
			pterm.Println()

			rs := rulesets.BuildDefaultRuleSets()
			results, err := motor.ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), b)

			resultSet := model.NewRuleResultSet(results)
			resultSet.SortResultsByLineNumber()
			fi, _ := os.Stat(args[0])
			duration := time.Since(start)

			if !detailsFlag {
				RenderSummary(resultSet, args)
				renderTime(timeFlag, duration, fi)

				return nil
			}

			if err != nil {
				return fmt.Errorf("error: %v\n\n", err.Error())
			}

			//writer.Flush()
			// TODO: build out stats

			pterm.Println() // Blank line

			//positiveBars := pterm.Bars{
			//	pterm.Bar{
			//		Label: "Errors",
			//		Value: resultSet.GetErrorCount(),
			//		Style: pterm.NewStyle(pterm.FgLightRed),
			//	},
			//	pterm.Bar{
			//		Label: "Warnings",
			//		Value: resultSet.GetWarnCount(),
			//		Style: pterm.NewStyle(pterm.FgLightYellow),
			//	},
			//	pterm.Bar{
			//		Label: "Info",
			//		Value: resultSet.GetInfoCount(),
			//		Style: pterm.NewStyle(pterm.FgLightBlue),
			//	},
			//}
			//
			//_ = pterm.DefaultBarChart.WithHorizontal().WithBars(positiveBars).Render()

			//pterm.Printf("Errors: %d\n", resultSet.GetErrorCount())
			//pterm.Printf("Warnings: %d\n", resultSet.GetWarnCount())
			//pterm.Printf("Info: %d\n\n", resultSet.GetInfoCount())

			// try a category print out.
			for key, _ := range model.RuleCategories {

				categoryResults := resultSet.GetResultsByRuleCategory(key)

				tableData := processResults(categoryResults)

				if len(categoryResults) > 0 {
					pterm.DefaultSection.Printf("%s Results\n", strings.Title(key))
					pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
				}
			}

			pterm.Println() // Blank line

			renderTime(timeFlag, duration, fi)

			return nil
		},
	}
)

func renderTime(timeFlag bool, duration time.Duration, fi os.FileInfo) {
	if timeFlag {
		pterm.Println()
		pterm.Info.Println(fmt.Sprintf("Vacuum took %d milliseconds to lint %dkb", duration.Milliseconds(), fi.Size()/1000))
		pterm.Println()
	}
}

func processResults(results []*model.RuleFunctionResult) [][]string {
	tableData := [][]string{{"Line / Column", "Severity", "Message", "Path"}}
	for _, r := range results {
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

		tableData = append(tableData, []string{start, sev, m, p})
	}
	return tableData
}

func RenderSummary(rs *model.RuleResultSet, args []string) {

	tableData := [][]string{{"Category", pterm.LightRed("Errors"), pterm.LightYellow("Warnings"),
		pterm.LightBlue("Info")}}

	for key, cat := range model.RuleCategories {
		errors := rs.GetErrorsByRuleCategory(key)
		warn := rs.GetWarningsByRuleCategory(key)
		info := rs.GetInfoByRuleCategory(key)

		if len(errors) > 0 || len(warn) > 0 || len(info) > 0 {

			tableData = append(tableData, []string{cat.Name, fmt.Sprintf("%d", len(errors)),
				fmt.Sprintf("%d", len(warn)), fmt.Sprintf("%d", len(info))})
		}

	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	pterm.Println()
	pterm.Printf(">> run 'vacuum %s -d' to see full details", args[0])
	pterm.Println()
	pterm.Println()

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
		"Linting passed, great job!", rs.GetWarnCount())

}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolP("details", "d", false, "Show full details of linting report")
	rootCmd.PersistentFlags().BoolP("time", "t", false, "Show how long vacuum took to run")

}

func initConfig() {

	// do something with this later, we don't need any configuration files right now

	//if cfgFile != "" {
	//	// Use config file from the flag.
	//	viper.SetConfigFile(cfgFile)
	//} else {
	//	// Find home directory.
	//	home, err := os.UserHomeDir()
	//	cobra.CheckErr(err)
	//
	//	// Search config in home directory with name ".cobra" (without extension).
	//	viper.AddConfigPath(home)
	//	viper.SetConfigType("yaml")
	//	viper.SetConfigName(".cobra")
	//}
	//
	//viper.AutomaticEnv()
	//
	//if err := viper.ReadInConfig(); err == nil {
	//	fmt.Println("Using config file:", viper.ConfigFileUsed())
	//}
}
