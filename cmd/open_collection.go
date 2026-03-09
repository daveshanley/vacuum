// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/logging"
	"github.com/daveshanley/vacuum/tui"
	"github.com/pb33f/doctor/frank"
	"github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/spf13/cobra"
)

func GetOpenCollectionCommand() *cobra.Command {
	cmd := &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,
		Use:           "open-collection <openapi-spec> <output>",
		Short:         "Convert an OpenAPI document into an OpenCollection format (opencollection.com)",
		Long: `Convert an OpenAPI document into an OpenCollection (Bruno v3+) format.

The output can be either:
  - A .yaml/.yml file path: produces a single bundled YAML file
  - A directory path: produces an exploded directory tree with individual request files`,
		Example: `  vacuum open-collection openapi.yaml collection.yaml
  vacuum open-collection openapi.yaml collection-dir/
  vacuum open-collection openapi.yaml collection.yaml --no-environments
  vacuum open-collection openapi.yaml collection.yaml --name "My API"`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: runOpenCollection,
	}
	cmd.Flags().BoolP("no-style", "q", false, "Disable styling and color output, just plain text (useful for CI/CD)")
	cmd.Flags().BoolP("no-environments", "E", false, "Skip environment generation")
	cmd.Flags().BoolP("no-descriptions", "D", false, "Skip including descriptions as docs")
	cmd.Flags().StringP("name", "n", "", "Override collection name")
	return cmd
}

func runOpenCollection(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	noStyleFlag, _ := cmd.Flags().GetBool("no-style")
	noEnvironments, _ := cmd.Flags().GetBool("no-environments")
	noDescriptions, _ := cmd.Flags().GetBool("no-descriptions")
	collectionName, _ := cmd.Flags().GetString("name")
	baseFlag, _ := cmd.Flags().GetString("base")
	remoteFlag, _ := cmd.Flags().GetBool("remote")
	timeFlag, _ := cmd.Flags().GetBool("time")
	debugFlag, _ := cmd.Flags().GetBool("debug")

	if noStyleFlag {
		color.DisableColors()
	} else {
		PrintBanner()
	}

	if len(args) < 2 {
		errText := "please supply an OpenAPI specification and an output path"
		tui.RenderErrorString("%s", errText)
		fmt.Println("Usage: vacuum open-collection <openapi-spec> <output>")
		fmt.Println()
		return errors.New(errText)
	}

	specPath := args[0]
	output := args[1]

	absSpecPath, err := filepath.Abs(specPath)
	if err != nil {
		tui.RenderErrorString("Unable to resolve spec path '%s': %s", specPath, err.Error())
		return err
	}

	specBytes, err := os.ReadFile(absSpecPath)
	if err != nil {
		tui.RenderErrorString("Unable to read file '%s': %s", specPath, err.Error())
		return err
	}

	// resolve base path: use flag if set, otherwise directory containing the spec
	basePath := filepath.Dir(absSpecPath)
	if baseFlag != "" {
		basePath = baseFlag
	}

	logLevel := logging.LogLevelWarn
	if debugFlag {
		logLevel = logging.LogLevelDebug
	}
	bufferedLogger := logging.NewBufferedLoggerWithLevel(logLevel)
	handler := logging.NewBufferedLogHandler(bufferedLogger)
	logger := slog.New(handler)

	cfg := &datamodel.DocumentConfiguration{
		BasePath:              basePath,
		AllowFileReferences:   true, // required for multi-file specs with local $ref
		AllowRemoteReferences: remoteFlag,
		Logger:                logger,
	}

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, cfg)
	if err != nil {
		tui.RenderErrorString("Failed to parse OpenAPI specification: %s", err.Error())
		return err
	}

	v3Model, errs := doc.BuildV3Model()
	if v3Model == nil {
		if doc.GetSpecInfo() != nil && doc.GetSpecInfo().SpecType == "swagger" {
			errText := "Swagger 2.x (OpenAPI 2.0) specifications are not supported, please convert to OpenAPI 3.x first"
			tui.RenderErrorString("%s", errText)
			return errors.New(errText)
		}
		errText := "failed to build OpenAPI v3 model"
		if errs != nil {
			errText = fmt.Sprintf("failed to build OpenAPI v3 model: %s", errs.Error())
		}
		tui.RenderErrorString("%s", errText)
		return errors.New(errText)
	}
	if errs != nil {
		// partial models can still produce useful collections
		tui.RenderWarning("OpenAPI model built with errors: %s", errs.Error())
	}

	drDoc := model.NewDrDocument(v3Model)
	if drDoc == nil {
		errText := "failed to build DrDocument from OpenAPI model"
		tui.RenderErrorString("%s", errText)
		return errors.New(errText)
	}

	f, err := frank.KnowWhatIMeanArry(&frank.FrankConfig{
		DrDoc:                    drDoc,
		GenerateEnvironments:     !noEnvironments,
		IncludeDescriptionAsDocs: !noDescriptions,
		Logger:                   logger,
		CollectionName:           collectionName,
	})
	if err != nil {
		tui.RenderErrorString("Failed to initialize OpenCollection generator: %s", err.Error())
		return err
	}

	result, err := f.Generate()
	if err != nil {
		tui.RenderErrorString("Failed to generate OpenCollection: %s", err.Error())
		return err
	}

	// detect output mode: .yaml/.yml → bundled, otherwise → exploded directory
	lower := strings.ToLower(output)
	bundled := strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")

	if bundled {
		data, err := frank.RenderBundled(result)
		if err != nil {
			tui.RenderErrorString("Failed to render bundled collection: %s", err.Error())
			return err
		}
		if dir := filepath.Dir(output); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				tui.RenderErrorString("Unable to create directory '%s': %s", dir, err.Error())
				return err
			}
		}
		if err := os.WriteFile(output, data, 0o644); err != nil {
			tui.RenderErrorString("Unable to write file '%s': %s", output, err.Error())
			return err
		}
		tui.RenderSuccess("Bundled OpenCollection written to '%s'", output)
	} else {
		if info, statErr := os.Stat(output); statErr == nil && !info.IsDir() {
			errText := fmt.Sprintf("output path '%s' exists as a regular file; use a .yaml/.yml extension for bundled output or a directory path for exploded output", output)
			tui.RenderErrorString("%s", errText)
			return errors.New(errText)
		}
		output = strings.TrimSuffix(output, "/")
		files, err := frank.RenderExploded(result)
		if err != nil {
			tui.RenderErrorString("Failed to render exploded collection: %s", err.Error())
			return err
		}
		if err := frank.WriteExploded(output, files); err != nil {
			tui.RenderErrorString("Unable to write to '%s': %s", output, err.Error())
			return err
		}
		tui.RenderSuccess("Exploded OpenCollection written to '%s/' (%d files)", output, len(files))
	}

	logOutput := bufferedLogger.RenderTree(noStyleFlag)
	if logOutput != "" {
		fmt.Print(logOutput)
	}

	duration := time.Since(startTime)
	RenderTime(timeFlag, duration, int64(len(specBytes)))

	return nil
}
