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
)

var rootCmd = &cobra.Command{
	Use:   "vacuum",
	Short: "vacuum is a very fast OpenAPI linter and toolkit",
	Long:  `vacuum is a very fast OpenAPI linter and toolkit for general things and stuff.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("please supply a filename to read")
		}

		fmt.Printf("running vacuum against '%s'\n", args[0])

		// read file.
		b, _ := ioutil.ReadFile(args[0])
		rs := rulesets.BuildDefaultRuleSets()
		results, err := motor.ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), b)

		resultSet := model.NewRuleResultSet(results)

		if err != nil {
			return fmt.Errorf("error: %v\n\n", err.Error())
		}

		tableData := [][]string{{"Start", "Severity", "Message", "Path"}}
		for _, r := range results {
			var start string
			if r.StartNode != nil && r.EndNode != nil {
				start = fmt.Sprintf("(%v:%v)", r.StartNode.Line, r.StartNode.Column)
				//end = fmt.Sprintf("(%v:%v)", r.EndNode.Line, r.EndNode.Column)
			} else {
				//start = "(x:x)"
				//end = "(x:x)"
			}

			m := r.Message
			p := r.Path
			if len(r.Path) > 60 {
				p = fmt.Sprintf("%s...", r.Path[:60])
			}

			if len(r.Message) > 80 {
				m = fmt.Sprintf("%s...", r.Message[:80])
			}

			//fmt.Fprintln(writer, fmt.Sprintf("%v\t%v", r.Message, p))
			sev := "nope"
			if r.Rule != nil {
				sev = r.Rule.Severity
			}

			switch sev {
			case "error":
				sev = pterm.LightRed(sev)
			case "warn":
				sev = pterm.LightYellow(sev)
			case "info":
				sev = pterm.LightBlue(sev)
			}

			tableData = append(tableData, []string{start, sev, m, p})
			//fmt.Fprintln(writer, fmt.Sprintf("%v\t%v\t%v\t%v", start, sev, m, p))

		}
		//writer.Flush()
		// TODO: build out stats

		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

		pterm.Println() // Blank line

		pterm.Printf("Errors: %d\n", resultSet.GetErrorCount())
		pterm.Printf("Warnings: %d\n", resultSet.GetWarnCount())
		pterm.Printf("Info: %d\n", resultSet.GetInfoCount())

		if resultSet.GetErrorCount() > 0 {
			return fmt.Errorf("there are %d errors in this contract", resultSet.GetErrorCount())
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
