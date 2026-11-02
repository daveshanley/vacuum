package motor

import (
	"fmt"

	"github.com/daveshanley/vacuum/repair"
)

func repairWithPlugins(execution *RuleSetExecution, plan repair.Plan, options *ExecutionOptions, before *RuleSetExecutionResult) (*RepairVerification, error) {
	plan.Targets = append([]repair.Target(nil), plan.Targets...)
	sources, hashes, err := freezeRepairSources(before)
	if err != nil {
		return nil, err
	}
	candidate := append([]byte(nil), execution.Spec...)
	if len(plan.Operations) > 0 || len(plan.KeyTags) > 0 || len(plan.KeyRemovals) > 0 {
		candidate, err = repair.Apply(execution.SpecFileName, execution.Spec, plan)
		if err != nil {
			return nil, err
		}
	}
	e := repairExecution(execution, candidate)
	e.RolodexFS = sources
	e.ApplyAutoFixes = true
	plugins := ApplyRulesToRuleSetWithOptions(e, options)
	defer plugins.ReleaseOwnedResources()
	if len(plugins.Errors) > 0 {
		return nil, fmt.Errorf("plugin repair failed: %w", plugins.Errors[0])
	}
	if len(plugins.ModifiedSpec) > 0 {
		locations, err := repairFindings(before.RuleSetExecution, before.Results)
		if err != nil {
			return nil, err
		}
		original := make(map[string]repair.Finding)
		for _, location := range locations {
			if location.Unavailable == "" {
				original[diagnosticIdentity(before.Results[location.Index])] = location
			}
		}
		for _, fixed := range plugins.FixedResults {
			location, ok := original[diagnosticIdentity(fixed)]
			if !ok {
				return nil, fmt.Errorf("plugin repair has no original authored target: %s", fixed.RuleId)
			}
			plan.Targets = append(plan.Targets, repair.Target{RuleID: location.RuleID, Code: location.Code, Pointer: location.Pointer})
		}
		candidate = plugins.ModifiedSpec
	}
	return verifyRepairCandidate(execution, plan, candidate, options, before, sources, len(hashes) > 0)
}
