// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT
// https://pb33f.io

package cmd

import (
	"fmt"
	"log/slog"
	"net/http"

	languageserver "github.com/daveshanley/vacuum/language-server"
	"github.com/daveshanley/vacuum/logging"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/spf13/cobra"
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
			// use discard logger to prevent memory accumulation
			bufferedLogger := logging.NewDiscardLogger()
			handler := logging.NewBufferedLogHandler(bufferedLogger)
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
			ignorePolymorphCircleRef, _ := cmd.Flags().GetBool("ignore-polymorph-circle-ref")
			extensionRefsFlag, _ := cmd.Flags().GetBool("ext-refs")
			ignoreFile, _ := cmd.Flags().GetString("ignore-file")

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

				// Certificate/TLS configuration for language server
				certFile, _ := cmd.Flags().GetString("cert-file")
				keyFile, _ := cmd.Flags().GetString("key-file")
				caFile, _ := cmd.Flags().GetString("ca-file")
				insecure, _ := cmd.Flags().GetBool("insecure")

				// Create HTTP client for remote ruleset downloads if needed
				var httpClient *http.Client
				resolvedCertFile, err := ResolveConfigPath(certFile)
				if err != nil {
					return fmt.Errorf("failed to resolve cert file path: %w", err)
				}
				resolvedKeyFile, err := ResolveConfigPath(keyFile)
				if err != nil {
					return fmt.Errorf("failed to resolve key file path: %w", err)
				}
				resolvedCAFile, err := ResolveConfigPath(caFile)
				if err != nil {
					return fmt.Errorf("failed to resolve CA file path: %w", err)
				}
				httpClientConfig := utils.HTTPClientConfig{
					CertFile: resolvedCertFile,
					KeyFile:  resolvedKeyFile,
					CAFile:   resolvedCAFile,
					Insecure: insecure,
				}
				if utils.ShouldUseCustomHTTPClient(httpClientConfig) {
					var clientErr error
					httpClient, clientErr = utils.CreateCustomHTTPClient(httpClientConfig)
					if clientErr != nil {
						return clientErr
					}
				}

				resolvedRulesetPath, resolveErr := ResolveConfigPath(rulesetFlag)
				if resolveErr != nil {
					return resolveErr
				}

				var rsErr error
				selectedRS, rsErr = BuildRuleSetFromUserSuppliedLocation(resolvedRulesetPath, defaultRuleSets, remoteFlag, httpClient)
				if rsErr != nil {
					return rsErr
				}
			}

			ignoredItems, err := LoadIgnoreFile(ignoreFile, true, false, false)
			if err != nil {
				return err
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
				IgnoredResults:           ignoredItems,
			}

			return languageserver.NewServer(GetVersion(), &lfr).Run()
		},
	}
	cmd.Flags().Bool("ignore-array-circle-ref", false, "Ignore circular array references")
	cmd.Flags().Bool("ignore-polymorph-circle-ref", false, "Ignore circular polymorphic references")
	cmd.Flags().String("ignore-file", "", "Path to ignore file")
	return cmd
}
