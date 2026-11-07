// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/repair"
	"github.com/spf13/cobra"
)

func addRepairFlags(command *cobra.Command) {
	command.Flags().Bool("dry-run", false, "Verify fixes and show changes without writing files (requires --fix)")
	command.Flags().String("fix-policy", "", "Local exact-scope repair values (requires --fix)")
}

func validateRepairFlags(fix, dryRun bool, policy string) error {
	if !fix && (dryRun || policy != "") {
		return fmt.Errorf("--dry-run and --fix-policy require --fix")
	}
	return nil
}

func executeLintWithRepairs(execution *motor.RuleSetExecution, options *motor.ExecutionOptions, flags *LintFlags, ignored model.IgnoredItems) *motor.RuleSetExecutionResult {
	fail := func(err error) *motor.RuleSetExecutionResult {
		return &motor.RuleSetExecutionResult{RuleSetExecution: execution, Errors: []error{err}}
	}
	if !flags.FixFlag {
		if err := validateRepairFlags(flags.FixFlag, flags.DryRun, flags.FixPolicy); err != nil {
			return fail(err)
		}
		return motor.ApplyRulesToRuleSetWithOptions(execution, options)
	}
	var policy *repair.Policy
	if flags.FixPolicy != "" {
		data, err := os.ReadFile(flags.FixPolicy)
		if err != nil {
			return fail(err)
		}
		base := filepath.Dir(flags.FixPolicy)
		names, err := repair.PolicyCatalogNames(data, base)
		if err != nil {
			return fail(err)
		}
		catalogs := make(map[string][]byte)
		for _, name := range names {
			content, err := os.ReadFile(name)
			if err != nil {
				return fail(err)
			}
			catalogs[name] = content
		}
		policy, err = repair.ParsePolicy(data, base, catalogs)
		if err != nil {
			return fail(fmt.Errorf("invalid repair policy: %w", err))
		}
	}
	execution.RepairIgnores = ignored
	result := runReviewedCLIRepair(execution, options, flags, policy)
	writeRepairWarnings(result, flags)
	return result
}

// repairSummary describes only verified repairs. Dry-run results are projected,
// and must not be described as persisted changes.
func repairSummary(result *motor.RuleSetExecutionResult, flags *LintFlags) string {
	if flags.DryRun {
		return ""
	}
	if result == nil || !flags.FixFlag || flags.SilentFlag || flags.FormattedOutput || flags.GitHubAnnotations || len(result.Errors) > 0 {
		return ""
	}
	if flags.PipelineOutput {
		return ""
	}
	var text strings.Builder
	name := "document"
	if result.RuleSetExecution != nil && result.RuleSetExecution.SpecFileName != "" {
		name = result.RuleSetExecution.SpecFileName
	}
	fmt.Fprintf(&text, "\nREPAIRS: %s\n", name)
	fmt.Fprintf(&text, "%d verified fixes; %d findings remain\n", len(result.FixedResults), len(result.Results))
	counts := model.NewRuleResultSet(result.Results)
	fmt.Fprintf(&text, "Remaining: %d errors, %d warnings, %d info, %d hints\n", counts.GetErrorCount(), counts.GetWarnCount(), counts.GetInfoCount(), counts.GetHintCount())
	for _, fixed := range result.FixedResults {
		fmt.Fprintf(&text, "  %s at %s: %s\n", fixed.RuleId, fixed.Path, fixed.Message)
	}

	return text.String()
}

func writtenFixCount(flags *LintFlags, verified int) int {
	if flags.DryRun {
		return 0
	}
	return verified
}

// writeRepairWarnings keeps declined-repair messages out of structured reports.
func writeRepairWarnings(result *motor.RuleSetExecutionResult, flags *LintFlags) {
	if result == nil || flags.repairWarningsSilenced || flags.SilentFlag && !flags.FormattedOutput {
		return
	}
	writer := flags.RepairWarningWriter
	if writer == nil {
		writer = os.Stderr
	}
	for _, warning := range result.RepairWarnings {
		fmt.Fprintf(writer, "Warning: %s\n", warning)
	}
}
