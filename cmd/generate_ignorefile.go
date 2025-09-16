// Copyright 2025 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/daveshanley/vacuum/cui"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
	"io"
	"log"
	"os"
)

func GetGenerateIgnoreFileCommand() *cobra.Command {

	// LintResult represents a single linting result
	type LintResult struct {
		Path   string `json:"path"`
		RuleID string `json:"ruleId"`
	}

	type LintReport struct {
		ResultSet struct {
			Results []LintResult `json:"results"`
		} `json:"resultSet"`
	}

	// RulePathsMap is the structure for YAML output
	type RulePathsMap map[string][]string

	cmd := &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,
		Use:           "generate-ignorefile",
		Short:         "Generate an ignorefile from a lint report",
		Long:          "Generate an ignorefile from a lint report. An ignorefile is used to ignore specific errors from the lint results.",
		Example:       "vacuum generate-ignorefile <report-file-name.json> <output-ignore-file-name.yaml>",
		RunE: func(cmd *cobra.Command, args []string) error {

			PrintBanner()

			// check for report file args
			if len(args) < 1 {
				errText := "please supply the lint report file"
				tui.RenderErrorString("%s", errText)
				return errors.New(errText)
			}

			lintReportPath := args[0]
			_, err := os.Stat(lintReportPath)
			if os.IsNotExist(err) {
				errText := fmt.Sprintf("cannot find lint report file at '%s'", lintReportPath)
				tui.RenderErrorString("%s", errText)
				return errors.New(errText)
			}

			outputFile := "ignorefile.yaml"
			if len(args) == 2 {
				outputFile = args[1]
			}

			tui.RenderInfo("Generating Ignorefile from lint errors in: %s", lintReportPath)

			// Read JSON file
			jsonFile, err := os.Open(lintReportPath)
			if err != nil {
				log.Fatalf("Failed to open JSON file: %v", err)
			}
			defer jsonFile.Close()

			byteValue, err := io.ReadAll(jsonFile)
			if err != nil {
				log.Fatalf("Failed to read JSON file: %v", err)
			}

			// Parse JSON into a slice of LintResult
			var lintReport LintReport
			if err := json.Unmarshal(byteValue, &lintReport); err != nil {
				log.Fatalf("Failed to parse JSON: %v", err)
			}

			// Organize paths by rule ID
			rulePaths := make(RulePathsMap)
			for _, result := range lintReport.ResultSet.Results {
				rulePaths[result.RuleID] = append(rulePaths[result.RuleID], result.Path)
			}

			// Convert to YAML format
			yamlData, err := yaml.Marshal(rulePaths)
			if err != nil {
				log.Fatalf("Failed to convert to YAML: %v", err)
			}

			outFile, err := os.Create(outputFile)
			if err != nil {
				log.Fatalf("Failed to open write file: %v", err)
			}
			defer outFile.Close()

			// Write YAML data to file
			_, err = outFile.Write(yamlData)
			if err != nil {
				log.Fatalf("Failed to write YAML file: %v", err)
			}

			tui.RenderSuccess("Ingorefile generated at '%s'", outputFile)

			return nil
		},
	}
	return cmd
}
