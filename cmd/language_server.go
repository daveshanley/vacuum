// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT
// https://pb33f.io

package cmd

import (
	languageserver "github.com/daveshanley/vacuum/language-server"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"io"
	"log/slog"
)

func GetLanguageServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		SilenceErrors: true,
		SilenceUsage:  true,
		Use:           "language-server",
		Short:         "Run a fully compliant LSP server for OpenAPI linting (Language Server Protocol)",
		Long: `Provides a fully compliant LSP backend for OpenAPI linting and validation. Connect up your favorite
IDE and start linting your OpenAPI documents in real-time.`,
		RunE: func(cmd *cobra.Command, args []string) error {

			// setup logging to be discarded, it will invalidate the LSP protocol
			handler := pterm.NewSlogHandler(&pterm.Logger{
				Writer: io.Discard,
			})
			logger := slog.New(handler)

			// extract flags
			rulesetFlag, _ := cmd.Flags().GetString("ruleset")
			functionsFlag, _ := cmd.Flags().GetString("functions")
			baseFlag, _ := cmd.Flags().GetString("base")
			skipCheckFlag, _ := cmd.Flags().GetBool("skip-check")
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			timeoutFlag, _ := cmd.Flags().GetInt("timeout")
			hardModeFlag, _ := cmd.Flags().GetBool("hard-mode")
			ignoreArrayCircleRef, _ := cmd.Flags().GetBool("ignore-array-circle-ref")
			ignorePolymorphCircleRef, _ := cmd.Flags().GetBool("ignore-array-circle-ref")
			extensionRefsFlag, _ := cmd.Flags().GetBool("ext-refs")

			defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(logger)
			selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
			customFunctions, _ := LoadCustomFunctions(functionsFlag, true)

			// HARD MODE
			if hardModeFlag {
				selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()

				// extract all OWASP Rules
				owaspRules := rulesets.GetAllOWASPRules()
				allRules := selectedRS.Rules
				for k, v := range owaspRules {
					allRules[k] = v
				}
			}

			if rulesetFlag != "" {
				remoteFlag, _ := cmd.Flags().GetBool("remote")
				var rsErr error
				selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(rulesetFlag, defaultRuleSets, remoteFlag)
				if rsErr != nil {
					return rsErr
				}
			}

			lfr := utils.LintFileRequest{
				BaseFlag:                 baseFlag,
				Remote:                   remoteFlag,
				SkipCheckFlag:            skipCheckFlag,
				DefaultRuleSets:          defaultRuleSets,
				SelectedRS:               selectedRS,
				Functions:                customFunctions,
				TimeoutFlag:              timeoutFlag,
				IgnoreArrayCircleRef:     ignoreArrayCircleRef,
				IgnorePolymorphCircleRef: ignorePolymorphCircleRef,
				Logger:                   logger,
				ExtensionRefs:            extensionRefsFlag,
			}

			return languageserver.NewServer(Version, &lfr).Run()
		},
	}
	cmd.Flags().Bool("ignore-array-circle-ref", false, "Ignore circular array references")
	cmd.Flags().Bool("ignore-polymorph-circle-ref", false, "Ignore circular polymorphic references")
	return cmd
}
