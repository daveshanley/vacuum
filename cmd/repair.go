package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/repair"
	"github.com/spf13/cobra"
)

func addRepairFlags(command *cobra.Command) {
	command.Flags().Bool("fix-builtins", false, "Enable verified built-in repairs with --fix")
	command.Flags().String("fix-policy", "", "Local exact-scope repair policy (requires --fix-builtins)")
	command.Flags().String("fix-plan", "", "Write a repair plan as JSON without changing the input (new file)")
}

func executeLintWithRepairs(execution *motor.RuleSetExecution, options *motor.ExecutionOptions, flags *LintFlags, ignored model.IgnoredItems) *motor.RuleSetExecutionResult {
	if !flags.FixBuiltins && flags.FixPolicy == "" && flags.FixPlan == "" {
		if flags.FixFlag && len(execution.AutoFixFunctions) > 0 {
			execution.RepairIgnores = ignored
			return motor.RunPluginRepairs(execution, options)
		}
		return motor.ApplyRulesToRuleSetWithOptions(execution, options)
	}
	fail := func(err error) *motor.RuleSetExecutionResult {
		return &motor.RuleSetExecutionResult{RuleSetExecution: execution, Errors: []error{err}}
	}
	if flags.FixPolicy != "" && !flags.FixBuiltins {
		return fail(fmt.Errorf("--fix-policy requires --fix-builtins"))
	}
	if !flags.FixFlag && flags.FixPlan == "" {
		return fail(fmt.Errorf("--fix-builtins requires --fix or --fix-plan"))
	}
	if flags.FixFlag && flags.FixPlan != "" {
		return fail(fmt.Errorf("--fix-plan is a preview and cannot be combined with --fix"))
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
	plan, result := motor.RunRepairs(execution, options, policy, flags.FixFlag)
	if len(result.Errors) > 0 {
		return result
	}
	if flags.FixPlan != "" {
		data, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			result.Errors = append(result.Errors, err)
			return result
		}
		// Exclusive creation cannot overwrite a source, policy, catalog or symlink.
		output, err := os.OpenFile(flags.FixPlan, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("create repair plan: %w", err))
			return result
		}
		_, writeErr := output.Write(append(data, '\n'))
		closeErr := output.Close()
		if writeErr != nil {
			result.Errors = append(result.Errors, writeErr)
		} else if closeErr != nil {
			result.Errors = append(result.Errors, closeErr)
		}
	}
	return result
}
