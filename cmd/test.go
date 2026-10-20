package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/daveshanley/vacuum/harness"
	"github.com/spf13/cobra"
)

// GetTestCommand creates the agent test harness command. Execution runs in a
// child process so plugin stdout cannot corrupt the parent JSON result stream.
func GetTestCommand() *cobra.Command {
	var schema, bundle, workerResult, evidence, format string
	var cases []string
	var sampleLimit int
	command := &cobra.Command{
		Use:          "test <suite.yaml>",
		Short:        "Test rules and document changes with explicit assertions and execution evidence",
		Long:         "Run a versioned YAML or JSON test suite through vacuum. Results are JSON; logs go to stderr. Suite paths are relative to the suite file. Ambient vacuum configuration is not loaded.",
		SilenceUsage: true, SilenceErrors: true,
		// The suite is the configuration contract. Do not inherit root config/update checks.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "json" {
				return writeHarnessError(cmd, fmt.Errorf("--format must be json"))
			}
			if schema != "" {
				if len(args) != 0 {
					return writeHarnessError(cmd, fmt.Errorf("--schema does not accept a suite"))
				}
				data, err := harness.Schema(schema)
				if err != nil {
					return writeHarnessError(cmd, err)
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return err
			}
			if len(args) != 1 {
				return writeHarnessError(cmd, fmt.Errorf("provide one suite file"))
			}
			if evidence != "summary" && evidence != "detailed" {
				return writeHarnessError(cmd, fmt.Errorf("--evidence must be summary or detailed"))
			}
			if sampleLimit < 0 || sampleLimit > 100 {
				return writeHarnessError(cmd, fmt.Errorf("--sample-limit must be between 0 and 100"))
			}
			samples := 0
			if evidence == "detailed" {
				samples = sampleLimit
			}
			if workerResult != "" {
				report := harness.RunFile(args[0], harness.RunOptions{Context: cmd.Context(), CaseIDs: cases, SampleLimit: samples, Version: GetVersion()})
				data, err := json.Marshal(report)
				if err != nil {
					return err
				}
				return os.WriteFile(workerResult, data, 0600)
			}
			// A result file avoids OS-specific inherited-descriptor protocols. Worker
			// stdout and stderr both remain logs; only the parent writes JSON to stdout.
			dir, err := os.MkdirTemp("", "vacuum-test-")
			if err != nil {
				return writeHarnessError(cmd, err)
			}
			defer os.RemoveAll(dir)
			resultPath := filepath.Join(dir, "result.json")
			executable, err := os.Executable()
			if err != nil {
				return writeHarnessError(cmd, err)
			}
			workerArgs := []string{"test", "--worker-result", resultPath, "--evidence", evidence, "--sample-limit", fmt.Sprint(sampleLimit)}
			for _, id := range cases {
				workerArgs = append(workerArgs, "--case", id)
			}
			workerArgs = append(workerArgs, "--", args[0])
			child := exec.CommandContext(cmd.Context(), executable, workerArgs...)
			child.Stdout = cmd.ErrOrStderr()
			child.Stderr = cmd.ErrOrStderr()
			if err = child.Run(); err != nil {
				return writeHarnessError(cmd, fmt.Errorf("test worker failed: %w", err))
			}
			data, err := os.ReadFile(resultPath)
			if err != nil {
				return writeHarnessError(cmd, err)
			}
			var report harness.Report
			if err = json.Unmarshal(data, &report); err != nil {
				return writeHarnessError(cmd, fmt.Errorf("invalid worker result: %w", err))
			}
			if bundle != "" {
				if err = harness.WriteBundle(&report, args[0], bundle); err != nil {
					report.Status = "error"
					report.ExitCode = 2
					report.Errors = append(report.Errors, "bundle: "+err.Error())
				}
			}
			return writeHarnessReport(cmd, &report)
		},
	}
	command.Flags().StringVar(&format, "format", "json", "Result format (json)")
	command.Flags().StringVar(&schema, "schema", "", "Print the suite or result JSON Schema")
	command.Flags().StringSliceVar(&cases, "case", nil, "Run selected case IDs (repeat or comma-separate)")
	command.Flags().StringVar(&evidence, "evidence", "summary", "Evidence detail: summary or detailed")
	command.Flags().IntVar(&sampleLimit, "sample-limit", 10, "Maximum matched source samples per selector (0-100)")
	command.Flags().StringVar(&bundle, "bundle", "", "Write verified local inputs and results to a new directory")
	command.Flags().StringVar(&workerResult, "worker-result", "", "Internal result channel")
	_ = command.Flags().MarkHidden("worker-result")
	return command
}
func writeHarnessError(cmd *cobra.Command, err error) error {
	return writeHarnessReport(cmd, &harness.Report{Version: 1, Status: "error", ExitCode: 2, Cases: []harness.CaseResult{}, Errors: []string{err.Error()}})
}
func writeHarnessReport(cmd *cobra.Command, report *harness.Report) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(report); err != nil {
		return err
	}
	if report.ExitCode != 0 {
		return &ExitError{Code: report.ExitCode, Message: "test suite " + report.Status}
	}
	return nil
}
