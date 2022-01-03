package cmd

import (
	"fmt"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
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
			fmt.Printf("error: %v", err.Error())
			return
		}
		for _, r := range results {
			fmt.Printf("%s (%v:%v) - (%v:%v)\n", r.Message,
				r.StartNode.Line, r.StartNode.Column,
				r.EndNode.Line, r.EndNode.Column)
		}

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
