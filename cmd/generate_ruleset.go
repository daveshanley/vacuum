// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
	"os"
)

func GetGenerateRulesetCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,
		Use:           "generate-ruleset",
		Short:         "Generate a vacuum RuleSet",
		Long:          "Generate a YAML ruleset containing 'all', or 'recommended' rules",
		Example:       "vacuum generate-ruleset recommended | all <ruleset-output-name>",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0:
				return []string{"recommended", "all"}, cobra.ShellCompDirectiveNoFileComp
			case 1:
				return []string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt
			default:
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			PrintBanner()

			// check for file args
			if len(args) < 1 {
				errText := "please supply 'recommended', 'owasp' or 'all' and a file path to output the ruleset"
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}

			if args[0] != "recommended" && args[0] != "all" && args[0] != "owasp" {
				errText := fmt.Sprintf("please use 'all', 'owasp' or 'recommended' your choice '%s' is not valid", args[0])
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}

			extension := ".yaml"
			reportOutput := "ruleset"

			if len(args) == 2 {
				reportOutput = args[1]
			}

			// read spec and parse to dashboard.
			defaultRuleSets := rulesets.BuildDefaultRuleSets()

			var selectedRuleSet *rulesets.RuleSet

			// default is recommended rules, based on spectral (for now anyway)
			selectedRuleSet = defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

			if args[0] == "all" {
				selectedRuleSet = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
			}

			if args[0] == "owasp" {
				selectedRuleSet = rulesets.GenerateOWASPOpenAPIRuleSet()
			}

			// this bit needs a re-think, but it works for now.
			// because Spectral has an ass backwards schema design, this disco dance here
			// is to re-encode from rules to ruleDefinitions (which is a proxy property)
			encoded, _ := json.Marshal(selectedRuleSet.Rules)
			encodedMap := make(map[string]interface{})
			err := json.Unmarshal(encoded, &encodedMap)
			if err != nil {
				return err
			}

			selectedRuleSet.RuleDefinitions = encodedMap

			pterm.Info.Printf("Generating RuleSet rules: %s", selectedRuleSet.DocumentationURI)
			pterm.Println()

			yamlBytes, _ := yaml.Marshal(selectedRuleSet)

			reportOutputName := fmt.Sprintf("%s-%s%s", reportOutput, args[0], extension)

			err = os.WriteFile(reportOutputName, yamlBytes, 0664)

			if err != nil {
				pterm.Error.Printf("Unable to write RuleSet file: '%s': %s\n", reportOutputName, err.Error())
				pterm.Println()
				return err
			}

			pterm.Success.Printf("RuleSet generated for '%s', written to '%s'\n", args[0], reportOutputName)
			pterm.Println()

			return nil
		},
	}
	return cmd
}
