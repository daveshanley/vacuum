// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/pb33f/libopenapi/bundler"
	"github.com/pb33f/libopenapi/datamodel"

	"github.com/daveshanley/vacuum/cui"
	"github.com/spf13/cobra"
)

func GetBundleCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage: true,
		Use:          "bundle",
		Short:        "Bundle an OpenAPI specification with external references into a single document.",
		Long: "Bundle an OpenAPI specification with external references into a single document. All references will be resolved and " +
			"the resulting document will be a valid OpenAPI specification, containing no external references. It can then be used by any tool that does not support references.",
		Example: "vacuum bundle <my-exploded-openapi-spec.yaml> <bundled-openapi-spec.yaml>",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			stdIn, _ := cmd.Flags().GetBool("stdin")
			stdOut, _ := cmd.Flags().GetBool("stdout")
			noStyleFlag, _ := cmd.Flags().GetBool("no-style")
			baseFlag, _ := cmd.Flags().GetString("base")
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			extensionRefsFlag, _ := cmd.Flags().GetBool("ext-refs")
			composed, _ := cmd.Flags().GetBool("composed")
			delimiter, _ := cmd.Flags().GetString("delimiter")

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				cui.DisableColors()
			}

			if !stdIn && !stdOut {
				PrintBanner()
			}

			if !stdIn && len(args) == 0 {
				errText := "please supply input (unbundled) OpenAPI document, or use the -i flag to use stdin"
				cui.RenderErrorString("%s", errText)
				fmt.Println("Usage: vacuum bundle <input-openapi-spec.yaml> <output-bundled-openapi-spec.yaml>")
				fmt.Println()
				return errors.New(errText)
			}

			// check for file args
			if !stdOut && len(args) == 1 {
				errText := "please supply output (bundled) OpenAPI document, or use the -o flag to use stdout"
				cui.RenderErrorString("%s", errText)
				fmt.Println("Usage: vacuum bundle <input-openapi-spec.yaml> <output-bundled-openapi-spec.yaml>")
				fmt.Println()
				return errors.New(errText)
			}

			var specBytes []byte
			var fileError error

			if stdIn {
				// read file from stdin
				inputReader := cmd.InOrStdin()
				buf := &bytes.Buffer{}
				_, fileError = buf.ReadFrom(inputReader)
				specBytes = buf.Bytes()

			} else {
				// read file from filesystem
				specBytes, fileError = os.ReadFile(args[0])
			}

			if fileError != nil {
				cui.RenderErrorString("Unable to read file '%s': %s", args[0], fileError.Error())
				return fileError
			}
			if baseFlag == "" {
				baseFlag = "."
			}

			// setup logging
			bufferedLogger := cui.NewBufferedLoggerWithLevel(cui.LogLevelWarn)
			handler := cui.NewBufferedLogHandler(bufferedLogger)
			logger := slog.New(handler)
			docConfig := &datamodel.DocumentConfiguration{
				BasePath:                baseFlag,
				ExtractRefsSequentially: true,
				Logger:                  logger,
				AllowRemoteReferences:   remoteFlag,
				ExcludeExtensionRefs:    extensionRefsFlag,
			}

			var bundled []byte
			var err error
			if !composed {
				bundled, err = bundler.BundleBytes(specBytes, docConfig)
			} else {
				bundled, err = bundler.BundleBytesComposed(specBytes, docConfig, &bundler.BundleCompositionConfig{
					Delimiter: delimiter,
				})
			}

			if err != nil {
				cui.RenderError(err)
				// render any buffered logs
				logOutput := bufferedLogger.RenderTree(noStyleFlag)
				if logOutput != "" {
					fmt.Print(logOutput)
				}
				if bundled == nil {
					return err
				}
			}

			if stdOut {
				fmt.Print(string(bundled))
				return nil
			}

			err = os.WriteFile(args[1], bundled, 0664)

			if err != nil {
				cui.RenderErrorString("Unable to write bundled file: '%s': %s", args[1], err.Error())
				return err
			}

			cui.RenderSuccess("Bundled OpenAPI document written to '%s'", args[1])

			return nil
		},
	}
	cmd.Flags().BoolP("composed", "c", false, "Use composed mode, which will bundle all components into the root document, re-writing references to point to the components section.")
	cmd.Flags().StringP("delimiter", "d", "__", "Delimiter used to separate clashing names in composed mode")
	cmd.Flags().BoolP("stdin", "i", false, "Use stdin as input, instead of a file")
	cmd.Flags().BoolP("stdout", "o", false, "Use stdout as output, instead of a file")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	return cmd
}
