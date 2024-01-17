// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/pb33f/libopenapi/bundler"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
)

func GetBundleCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage: true,
		Use:          "bundle",
		Short:        "Bundle an OpenAPI specification with external references into a single document.",
		Long: "Bundle an OpenAPI specification with external references into a single document. All references will be resolved and " +
			"the resulting document will be a valid OpenAPI specification, containing no references. It can then be used by any tool that does not support references.",
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

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				pterm.DisableColor()
				pterm.DisableStyling()
			}

			if !stdIn && !stdOut {
				PrintBanner()
			}

			if !stdIn && len(args) == 0 {
				errText := "please supply input (unbundled) OpenAPI document, or use the -i flag to use stdin"
				pterm.Error.Println(errText)
				pterm.Println()
				pterm.Println("Usage: vacuum bundle <input-openapi-spec.yaml> <output-bundled-openapi-spec.yaml>")
				pterm.Println()
				return errors.New(errText)
			}

			// check for file args
			if !stdOut && len(args) == 1 {
				errText := "please supply output (bundled) OpenAPI document, or use the -o flag to use stdout"
				pterm.Error.Println(errText)
				pterm.Println()
				pterm.Println("Usage: vacuum bundle <input-openapi-spec.yaml> <output-bundled-openapi-spec.yaml>")
				pterm.Println()
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
				pterm.Error.Printf("Unable to read file '%s': %s\n", args[0], fileError.Error())
				pterm.Println()
				return fileError
			}
			if baseFlag == "" {
				baseFlag = "."
			}

			// setup logging
			handler := pterm.NewSlogHandler(&pterm.Logger{
				Formatter: pterm.LogFormatterColorful,
				Writer:    os.Stdout,
				Level:     pterm.LogLevelWarn,
				ShowTime:  false,
				MaxWidth:  280,
				KeyStyles: map[string]pterm.Style{
					"error":  *pterm.NewStyle(pterm.FgRed, pterm.Bold),
					"err":    *pterm.NewStyle(pterm.FgRed, pterm.Bold),
					"caller": *pterm.NewStyle(pterm.FgGray, pterm.Bold),
				},
			})
			logger := slog.New(handler)
			docConfig := &datamodel.DocumentConfiguration{
				BasePath:                baseFlag,
				ExtractRefsSequentially: true,
				Logger:                  logger,
			}

			bundled, err := bundler.BundleBytes(specBytes, docConfig)

			if err != nil {
				pterm.Error.Printf("Bundling had errors: %s\n", err.Error())
				pterm.Println()
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
				pterm.Error.Printf("Unable to write bundled file: '%s': %s\n", args[1], err.Error())
				pterm.Println()
				return err
			}

			pterm.Success.Printf("Bundled OpenAPI document written to '%s'\n", args[1])
			pterm.Println()

			return nil
		},
	}
	cmd.Flags().BoolP("stdin", "i", false, "Use stdin as input, instead of a file")
	cmd.Flags().BoolP("stdout", "o", false, "Use stdout as output, instead of a file")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	return cmd
}
