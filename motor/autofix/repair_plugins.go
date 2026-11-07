// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"fmt"

	"github.com/daveshanley/vacuum/motor"

	"github.com/daveshanley/vacuum/repair"
)

func repairWithPlugins(execution *motor.RuleSetExecution, plan repair.Plan, options *motor.ExecutionOptions, before *motor.RuleSetExecutionResult, cache *repairBaselineCache) (*RepairVerification, error) {
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
	plugins := motor.ApplyRulesToRuleSetWithOptions(e, options)
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
				result := before.Results[location.Index]
				result.Path = location.Pointer
				original[diagnosticIdentity(result)] = location
			}
		}
		correspondence, err := repair.TraceChanges(execution.Spec, candidate, plan)
		if err != nil {
			return nil, err
		}
		// Diagnostic nodes may be detached snapshots of the input or live nodes
		// changed by a callback. Resolve snapshots against the unchanged input first.
		input := &motor.RuleSetExecution{Spec: candidate, SpecFileName: execution.SpecFileName}
		pluginLocations, err := repairFindings(input, plugins.FixedResults)
		if err != nil {
			return nil, err
		}
		var liveLocations []repair.Finding
		for i := range pluginLocations {
			if pluginLocations[i].Unavailable == "" {
				continue
			}
			if liveLocations == nil {
				liveLocations, err = repairFindings(e, plugins.FixedResults)
				if err != nil {
					return nil, err
				}
			}
			pluginLocations[i] = liveLocations[i]
		}
		for i, fixed := range plugins.FixedResults {
			pointer, mapped := correspondence.OriginalPointer(pluginLocations[i].Pointer)
			if pluginLocations[i].Unavailable != "" || !mapped {
				return nil, fmt.Errorf("plugin repair has no original authored target: %s", fixed.RuleId)
			}
			fixed.Path = pointer
			fixed.Message = originalDiagnosticLines(fixed.Message, correspondence)
			location, ok := original[diagnosticIdentity(fixed)]
			if !ok {
				return nil, fmt.Errorf("plugin repair has no original authored target: %s", fixed.RuleId)
			}
			plan.Targets = append(plan.Targets, repair.Target{RuleID: location.RuleID, Code: location.Code, Pointer: location.Pointer})
		}
		candidate = plugins.ModifiedSpec
	}
	return verifyRepairCandidate(execution, plan, candidate, options, before, sources, len(hashes) > 0, cache)
}
