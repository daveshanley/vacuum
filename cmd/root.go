package cmd

import (
	"fmt"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"text/tabwriter"
)

var rootCmd = &cobra.Command{
	Use:   "vacuum",
	Short: "vacuum is a very fast OpenAPI linter and toolkit",
	Long:  `vacuum is a very fast OpenAPI linter and toolkit for general things and stuff.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Printf("please supply a filename to read")
			return
		}

		fmt.Printf("running vacuum against '%s'\n", args[0])

		// read file.
		b, _ := ioutil.ReadFile(args[0])
		rs := rulesets.BuildDefaultRuleSets()
		results, err := motor.ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), b)
		if err != nil {
			fmt.Printf("error: %v\n\n", err.Error())
			return
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
		//fmt.Fprintln(writer, "Start\tEnd\tMessage\tPath")
		//fmt.Fprintln(writer, "-----\t---\t-------\t----")
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
			fmt.Fprintln(writer, fmt.Sprintf("%v\t%v\t%v\t%v", start, sev, m, p))

		}
		writer.Flush()

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
