// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	asyncapiContext "github.com/daveshanley/vacuum/asyncapi"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/logging"
	"github.com/pb33f/libasyncapi"
	asyncapiBundler "github.com/pb33f/libasyncapi/bundler"
	"github.com/pb33f/libopenapi/bundler"
	"github.com/pb33f/libopenapi/datamodel"
	libopenapijson "github.com/pb33f/libopenapi/json"

	"github.com/daveshanley/vacuum/tui"
	"github.com/pb33f/go-yaml"
	"github.com/spf13/cobra"
)

const (
	bundleOutputFormatYAML = "yaml"
	bundleOutputFormatJSON = "json"
)

func GetBundleCommand() *cobra.Command {

	cmd := &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,
		Use:           "bundle",
		Short:         "Bundle an OpenAPI or AsyncAPI 3 specification into a single document.",
		Long: "Bundle an OpenAPI or AsyncAPI 3 specification with external references into a single document. " +
			"Local references are preserved. AsyncAPI fields that require references keep local references to their targets.",
		Example: "vacuum bundle <input-api.yaml> <bundled-api.yaml>",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			stdIn, _ := cmd.Flags().GetBool("stdin")
			stdOut, _ := cmd.Flags().GetBool("stdout")
			formatFlag, _ := cmd.Flags().GetString("format")
			noStyleFlag, _ := cmd.Flags().GetBool("no-style")
			baseFlag, _ := cmd.Flags().GetString("base")
			remoteFlag, _ := cmd.Flags().GetBool("remote")
			extensionRefsFlag, _ := cmd.Flags().GetBool("ext-refs")
			composed, _ := cmd.Flags().GetBool("composed")
			delimiter, _ := cmd.Flags().GetString("delimiter")

			// disable color and styling, for CI/CD use.
			// https://github.com/daveshanley/vacuum/issues/234
			if noStyleFlag {
				color.DisableColors()
			}

			if !stdIn && !stdOut {
				PrintBanner()
			}

			if !stdIn && len(args) == 0 {
				err := NewInputError("please supply an input API document, or use --stdin")
				fmt.Fprintln(os.Stderr, err)
				return err
			}

			// check for file args
			outputIndex := 1
			if stdIn && len(args) < 2 {
				outputIndex = 0
			}
			// Preserve the legacy --stdin <ignored-input> <output> form.
			if len(args) > 2 {
				err := NewInputError("bundle accepts at most an input and an output document")
				fmt.Fprintln(os.Stderr, err)
				return err
			}
			if !stdOut && len(args) <= outputIndex {
				err := NewInputError("please supply an output API document, or use --stdout")
				fmt.Fprintln(os.Stderr, err)
				return err
			}

			formatArgs := args
			if stdIn && outputIndex == 0 {
				formatArgs = append([]string{"stdin"}, args...)
			}
			outputFormat, err := resolveBundleOutputFormat(formatFlag, stdOut, formatArgs)
			if err != nil {
				tui.RenderErrorString("%s", err.Error())
				return err
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
				fmt.Fprintln(os.Stderr, fileError)
				return fileError
			}
			isAsyncAPI, err := asyncapiContext.IsDocument(specBytes)
			if err != nil && asyncapiContext.HasMarker(specBytes) {
				fmt.Fprintln(os.Stderr, err)
				return NewInputError("%v", err)
			}
			if baseFlag == "" {
				baseFlag = "."
				if isAsyncAPI && !stdIn {
					baseFlag = filepath.Dir(args[0])
				}
			}

			// setup logging
			bufferedLogger := logging.NewBufferedLoggerWithLevel(logging.LogLevelWarn)
			handler := logging.NewBufferedLogHandler(bufferedLogger)
			logger := slog.New(handler)
			docConfig := &datamodel.DocumentConfiguration{
				BasePath:                baseFlag,
				ExtractRefsSequentially: true,
				Logger:                  logger,
				AllowRemoteReferences:   remoteFlag,
				ExcludeExtensionRefs:    !extensionRefsFlag,
			}

			var bundled []byte
			if isAsyncAPI {
				config := &libasyncapi.DocumentConfiguration{
					AllowFileReferences: true, AllowRemoteReferences: remoteFlag,
					ExcludeExtensionRefs: !extensionRefsFlag, Logger: logger,
				}
				if strings.HasPrefix(baseFlag, "http://") || strings.HasPrefix(baseFlag, "https://") {
					config.BaseURL, err = url.Parse(baseFlag)
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						return NewInputError("invalid base URL: %v", err)
					}
				} else {
					config.BasePath = baseFlag
				}
				if !stdIn {
					config.SpecFilePath, err = filepath.Abs(args[0])
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						return err
					}
				}
				if composed {
					bundled, err = asyncapiBundler.BundleBytesComposed(specBytes, config, &asyncapiBundler.BundleCompositionConfig{Delimiter: delimiter})
				} else {
					bundled, err = asyncapiBundler.BundleBytes(specBytes, config)
				}
			} else if !composed {
				bundled, err = bundler.BundleBytes(specBytes, docConfig)
			} else {
				bundled, err = bundler.BundleBytesComposed(specBytes, docConfig, &bundler.BundleCompositionConfig{
					Delimiter: delimiter,
				})
			}

			if err != nil {
				if isAsyncAPI {
					fmt.Fprintln(os.Stderr, err)
				} else {
					tui.RenderError(err)
				}
				// render any buffered logs
				logOutput := bufferedLogger.RenderTree(noStyleFlag)
				if logOutput != "" {
					if isAsyncAPI {
						fmt.Fprint(os.Stderr, logOutput)
					} else {
						fmt.Print(logOutput)
					}
				}
				if isAsyncAPI {
					return NewInputError("%v", err)
				}
				if bundled == nil {
					return err
				}
			}

			bundled, err = renderBundledOutput(bundled, outputFormat)
			if err != nil {
				tui.RenderErrorString("Unable to render bundled output as %s: %s", outputFormat, err.Error())
				return err
			}

			if stdOut {
				fmt.Print(string(bundled))
				return nil
			}

			err = os.WriteFile(args[outputIndex], bundled, 0664)

			if err != nil {
				tui.RenderErrorString("Unable to write bundled file: '%s': %s", args[outputIndex], err.Error())
				return err
			}

			tui.RenderSuccess("Bundled API document written to '%s'", args[outputIndex])

			return nil
		},
	}
	cmd.Flags().BoolP("composed", "c", false, "Use composed mode, which will bundle all components into the root document, re-writing references to point to the components section.")
	cmd.Flags().StringP("delimiter", "d", "__", "Delimiter used to separate clashing names in composed mode")
	cmd.Flags().String("format", "", "Output format for the bundled document. Supported values: yaml, json. Defaults to the output file extension or yaml for stdout.")
	cmd.Flags().BoolP("stdin", "i", false, "Use stdin as input, instead of a file")
	cmd.Flags().BoolP("stdout", "o", false, "Use stdout as output, instead of a file")
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	_ = cmd.RegisterFlagCompletionFunc("format", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{bundleOutputFormatYAML, bundleOutputFormatJSON}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func resolveBundleOutputFormat(formatFlag string, stdOut bool, args []string) (string, error) {
	if formatFlag != "" {
		switch strings.ToLower(formatFlag) {
		case bundleOutputFormatYAML, "yml":
			return bundleOutputFormatYAML, nil
		case bundleOutputFormatJSON:
			return bundleOutputFormatJSON, nil
		default:
			return "", fmt.Errorf("invalid bundle output format %q, expected yaml or json", formatFlag)
		}
	}

	if stdOut || len(args) < 2 {
		return bundleOutputFormatYAML, nil
	}

	switch strings.ToLower(filepath.Ext(args[1])) {
	case ".json":
		return bundleOutputFormatJSON, nil
	case ".yaml", ".yml":
		return bundleOutputFormatYAML, nil
	default:
		return bundleOutputFormatYAML, nil
	}
}

func renderBundledOutput(bundled []byte, format string) ([]byte, error) {
	switch format {
	case bundleOutputFormatYAML:
		return bundled, nil
	case bundleOutputFormatJSON:
		var node yaml.Node
		if err := yaml.Unmarshal(bundled, &node); err != nil {
			return nil, err
		}
		return libopenapijson.YAMLNodeToJSON(&node, "  ")
	default:
		return nil, fmt.Errorf("unsupported bundle output format %q", format)
	}
}
