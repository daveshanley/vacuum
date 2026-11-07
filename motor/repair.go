// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

// RepairVerification owns the final lint result and the verified candidate.
// Release Final's owned resources when done. Candidate is nil on failure.
type RepairVerification struct {
	Candidate []byte                  `json:"candidate,omitempty"`
	Verified  []repair.Target         `json:"verified,omitempty"`
	Final     *RuleSetExecutionResult `json:"-"`
}

// remainingRepairTargets identifies unresolved selected targets after all other
// verification gates pass. It never represents newly introduced diagnostics.
type remainingRepairTargets struct {
	targets []repair.Target
}

func (e *remainingRepairTargets) Error() string {
	first := e.targets[0]
	message := fmt.Sprintf("repair target remains: %s at %s", first.RuleID, first.Pointer)
	if len(e.targets) > 1 {
		message += fmt.Sprintf(" (and %d more)", len(e.targets)-1)
	}
	return message
}

// PlanFixes lints the source snapshot and proposes built-in repairs. The caller
// owns the returned lint result, including when a proposal cannot be created.
func PlanFixes(execution *RuleSetExecution, options *ExecutionOptions) (repair.Plan, *RuleSetExecutionResult, error) {
	return PlanFixesWithPolicy(execution, options, nil)
}

// PlanFixesWithPolicy includes exact caller-supplied policy values.
func PlanFixesWithPolicy(execution *RuleSetExecution, options *ExecutionOptions, policy *repair.Policy) (repair.Plan, *RuleSetExecutionResult, error) {
	if execution == nil || execution.RuleSet == nil {
		return repair.Plan{}, nil, fmt.Errorf("repair execution and ruleset are required")
	}
	if len(execution.Spec) > 64<<20 {
		return repair.Plan{}, nil, fmt.Errorf("repair source exceeds 64 MiB")
	}
	var checked ExecutionOptions
	if options != nil {
		checked = *options
	}
	checked.ReportFunctionErrors = true
	options = &checked
	e := repairExecution(execution, execution.Spec)
	baseline := ApplyRulesToRuleSetWithOptions(e, options)
	plan := repair.NewPlan(execution.SpecFileName, execution.Spec)
	if len(baseline.Errors) > 0 {
		return plan, baseline, fmt.Errorf("cannot plan repairs after incomplete lint: %w", baseline.Errors[0])
	}
	baseline.Results = utils.FilterIgnoredResultsWithOptions(baseline.Results, e.RepairIgnores, utils.IgnoreMatcherOptions{RootNode: e.CanonicalDocument, SpecBytes: e.Spec, LookupTimeout: e.NodeLookupTimeout})
	baseline.RepairIgnoresApplied = true
	findings, err := baselineRepairFindings(baseline)
	if err != nil {
		return plan, baseline, err
	}
	_, sourceHashes, sourceErr := freezeRepairSources(baseline)
	if sourceErr != nil {
		for _, f := range findings {
			plan.Declined = append(plan.Declined, repair.Declined{Target: repair.Target{RuleID: f.RuleID, Pointer: f.Pointer}, Reason: sourceErr.Error()})
		}
		return plan, baseline, nil
	}
	plan, err = repair.BuiltinPlan(execution.SpecFileName, execution.Spec, findings)
	if err == nil {
		plan, err = repair.ApplyPolicy(plan, execution.Spec, findings, policy)
	}
	plan.Sources = sourceHashes
	return plan, baseline, err
}

// validatePreparedPlan verifies a proposal produced by the built-in review flow.
func validatePreparedPlan(execution *RuleSetExecution, plan repair.Plan, options *ExecutionOptions, before *RuleSetExecutionResult) (*RepairVerification, error) {
	return validatePreparedPlanOutput(execution, plan, options, before, FixOutputOptions{})
}

func validatePreparedPlanOutput(execution *RuleSetExecution, plan repair.Plan, options *ExecutionOptions, before *RuleSetExecutionResult, output FixOutputOptions) (*RepairVerification, error) {
	if err := repairCanceled(options); err != nil {
		return nil, err
	}
	sources, hashes, err := freezeRepairSources(before)
	if err != nil {
		return nil, err
	}
	a, _ := json.Marshal(hashes)
	b, _ := json.Marshal(plan.Sources)
	if len(hashes) > 0 && !bytes.Equal(a, b) {
		return nil, fmt.Errorf("repair dependency snapshots changed")
	}
	candidate, err := repair.Apply(execution.SpecFileName, execution.Spec, plan)
	if err != nil {
		return nil, err
	}
	candidate, err = prepareRepairOutput(execution, candidate, options, before, sources, output)
	if err != nil {
		return nil, err
	}
	return verifyRepairCandidate(execution, plan, candidate, options, before, sources, len(hashes) > 0)
}

func verifyRepairCandidate(execution *RuleSetExecution, plan repair.Plan, candidate []byte, options *ExecutionOptions, before *RuleSetExecutionResult, sources *repairSources, hasExternal bool) (*RepairVerification, error) {
	if err := repairCanceled(options); err != nil {
		return nil, err
	}
	if err := repair.ValidateReviewedExamples(candidate, plan); err != nil {
		return nil, err
	}
	parsed, err := repair.ParseDocument(candidate)
	if err != nil {
		return nil, err
	}
	correspondence, err := repair.TraceChanges(execution.Spec, candidate, plan)
	if err != nil {
		return nil, err
	}
	originalIgnores, _ := json.Marshal(repairIgnoreScopes(before.RuleSetExecution.CanonicalDocument))
	candidateScopes := make(map[string]map[string]bool)
	for path, scope := range repairIgnoreScopes(parsed) {
		origin, ok := correspondence.OriginalPointer(path)
		if !ok {
			return nil, fmt.Errorf("repair cannot introduce inline ignore directives")
		}
		candidateScopes[origin] = scope
	}
	candidateIgnores, _ := json.Marshal(candidateScopes)
	if !bytes.Equal(originalIgnores, candidateIgnores) {
		return nil, fmt.Errorf("repair cannot alter inline ignore directives")
	}
	e := repairExecution(execution, candidate)
	e.RolodexFS = sources
	// Arazzo replays captured retrievals through its own resolver. Keep its
	// concrete snapshot type so validation cannot select a live resolver.
	if before.RuleSetExecution.Arazzo == nil {
		indexedSources, err := newRepairIndexedSources(sources)
		if err != nil {
			return nil, err
		}
		e.RolodexFS = indexedSources
	}
	if !hasExternal && externalRepairBytes(candidate) {
		return nil, fmt.Errorf("repair cannot introduce external inputs")
	}
	after := ApplyRulesToRuleSetWithOptions(e, options)
	fail := func(err error) (*RepairVerification, error) { after.ReleaseOwnedResources(); return nil, err }
	if len(after.Errors) > 0 {
		return fail(fmt.Errorf("repair validation did not complete: %w", after.Errors[0]))
	}
	if err := verifyRepairIntegrity(before, after, correspondence, options); err != nil {
		return fail(err)
	}
	filterRepairCandidateIgnores(before, after, correspondence)
	beforeFindings, err := baselineRepairFindings(before)
	if err != nil {
		return fail(err)
	}
	afterFindings, err := repairFindings(e, after.Results)
	if err != nil {
		return fail(err)
	}
	baseline := make(map[string]int)
	for i, r := range before.Results {
		if beforeFindings[i].Unavailable == "" {
			r.Path = beforeFindings[i].Pointer
		}
		baseline[diagnosticIdentity(r)]++
	}
	for i, r := range after.Results {
		if afterFindings[i].Unavailable == "" {
			origin, ok := correspondence.OriginalPointer(afterFindings[i].Pointer)
			if !ok {
				return fail(fmt.Errorf("repair introduces diagnostic %s at %s", r.RuleId, r.Path))
			}
			r.Path = origin
			r.Message = originalDiagnosticLines(r.Message, correspondence)
		}
		k := diagnosticIdentity(r)
		if baseline[k] == 0 {
			return fail(fmt.Errorf("repair introduces diagnostic %s at %s", r.RuleId, r.Path))
		}
		baseline[k]--
	}
	original := make(map[string]bool)
	remaining := make(map[string]bool)
	for _, f := range beforeFindings {
		if f.Unavailable == "" {
			original[f.RuleID+"\x00"+f.Pointer] = true
		}
	}
	for _, f := range afterFindings {
		if f.Unavailable == "" {
			if origin, ok := correspondence.OriginalPointer(f.Pointer); ok {
				remaining[f.RuleID+"\x00"+origin] = true
			}
		}
	}
	var unresolved []repair.Target
	for _, target := range plan.Targets {
		k := target.RuleID + "\x00" + target.Pointer
		if !original[k] {
			return fail(fmt.Errorf("repair target is not an original diagnostic: %s", target.RuleID))
		}
		if remaining[k] {
			unresolved = append(unresolved, target)
		}
	}
	if err := repairCanceled(options); err != nil {
		return fail(err)
	}
	if len(unresolved) > 0 {
		return fail(&remainingRepairTargets{targets: unresolved})
	}
	return &RepairVerification{Candidate: candidate, Verified: plan.Targets, Final: after}, nil
}

func repairExecution(source *RuleSetExecution, data []byte) *RuleSetExecution {
	e := source.CopyConfig()
	e.captureRepairSources = true
	e.Spec = append([]byte(nil), data...)
	e.ApplyAutoFixes = false
	return e
}

func diagnosticIdentity(r model.RuleFunctionResult) string {
	message := r.Message
	if r.RuleId == "resolving-references" {
		message = referenceDiagnosticMessage(message)
	}
	if r.DiagnosticCode != "" {
		message = r.DiagnosticCode
	}
	return r.RuleId + "\x00" + r.Path + "\x00" + message + "\x00" + r.RuleSeverity
}
func repairFindings(execution *RuleSetExecution, results []model.RuleFunctionResult) ([]repair.Finding, error) {
	root := execution.CanonicalDocument
	if root == nil {
		var err error
		root, err = repair.ParseDocument(execution.Spec)
		if err != nil {
			return nil, err
		}
	}
	locations := repair.IndexLocations(root)
	var findings []repair.Finding
	for i, r := range results {
		id := r.RuleId
		if id == "" && r.Rule != nil {
			id = r.Rule.Id
		}
		finding := repair.Finding{Index: i, RuleID: id, Code: r.DiagnosticCode}
		if r.Origin != nil && r.Origin.AbsoluteLocation != "" && execution.SpecFileName != "" {
			a, _ := filepath.Abs(execution.SpecFileName)
			b, _ := filepath.Abs(r.Origin.AbsoluteLocation)
			if a != b {
				finding.Unavailable = "diagnostic belongs to an external source"
				findings = append(findings, finding)
				continue
			}
		}
		node := r.RepairNode
		if node == nil {
			node = r.StartNode
		}
		// Repair owners can come from a separate resolved parse. Verify that
		// owner's document before allowing a position match into the source.
		local := r.RepairNode == nil
		if _, exact := locations.FindExact(node); exact {
			local = true
		}
		if !local && execution.Document != nil && execution.Document.GetRolodex() != nil {
			rolo := execution.Document.GetRolodex()
			origin := rolo.FindNodeOrigin(node)
			local = origin != nil && origin.Index == rolo.GetRootIndex()
			for _, idx := range rolo.GetIndexes() {
				if idx == rolo.GetRootIndex() {
					continue
				}
				if origin := idx.FindNodeOrigin(node); origin != nil && origin.Node == node {
					local = false
					break
				}
			}
		}
		if pointer, ok := locations.Find(node); local && ok {
			finding.Pointer = pointer
		} else {
			finding.Unavailable = "diagnostic has no unique authored source location"
		}
		findings = append(findings, finding)
	}
	return findings, nil
}
func externalRepairBytes(data []byte) bool {
	var root yaml.Node
	if yaml.Unmarshal(data, &root) != nil {
		return true
	}
	return externalRepairInputs(&root)
}
func externalRepairInputs(root *yaml.Node) bool {
	seen := make(map[*yaml.Node]bool)
	var walk func(*yaml.Node) bool
	walk = func(n *yaml.Node) bool {
		if n == nil || seen[n] {
			return false
		}
		seen[n] = true
		if n.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(n.Content); i += 2 {
				key, value := n.Content[i].Value, n.Content[i+1].Value
				if (key == "$ref" || key == "$dynamicRef" || key == "$recursiveRef") && value != "" && !strings.HasPrefix(value, "#") {
					return true
				}
				if key == "sourceDescriptions" && (n.Content[i+1].Kind != yaml.SequenceNode || len(n.Content[i+1].Content) != 0) {
					return true
				}
			}
		}
		for _, c := range n.Content {
			if walk(c) {
				return true
			}
		}
		return false
	}
	return walk(root)
}

// RunRepairs plans and optionally verifies built-in/policy repairs with one
// baseline lint and one verification lint. Enabled legacy callbacks add a pass.
// The returned plan describes built-in/policy edits; callback effects are in
// ModifiedSpec and FixedResults. The result remains caller-owned.
func RunRepairs(execution *RuleSetExecution, options *ExecutionOptions, policy *repair.Policy, apply bool) (repair.Plan, *RuleSetExecutionResult) {
	return runRepairs(execution, options, policy, apply, true)
}

// RunPluginRepairs preserves named plugin callbacks while verifying their final
// document. Plugin callbacks are trusted code and may require a third lint pass.
func RunPluginRepairs(execution *RuleSetExecution, options *ExecutionOptions) *RuleSetExecutionResult {
	_, result := runRepairs(execution, options, nil, true, false)
	return result
}

func runRepairs(execution *RuleSetExecution, options *ExecutionOptions, policy *repair.Policy, apply, builtins bool) (repair.Plan, *RuleSetExecutionResult) {
	originalOptions := options
	strict := options != nil && options.ReportFunctionErrors || execution != nil && execution.RuleSet != nil && execution.RuleSet.Executable != nil
	options, cancel := repairRunOptions(options)
	defer cancel()
	plan, before, err := PlanFixesWithPolicy(execution, options, policy)
	if !builtins {
		plan.Operations = nil
		plan.KeyTags = nil
		plan.KeyRemovals = nil
		plan.Targets = nil
	}
	if before == nil {
		if err == nil {
			err = fmt.Errorf("repair baseline is unavailable")
		}
		return plan, &RuleSetExecutionResult{RuleSetExecution: execution, Errors: []error{err}}
	}
	if err != nil {
		// Repeat only an incomplete strict lint with the caller's settings.
		// This preserves all ordinary errors without guessing their kinds.
		if !strict && execution != nil && len(before.Errors) > 0 {
			before.ReleaseOwnedResources()
			before = ApplyRulesToRuleSetWithOptions(repairExecution(execution, execution.Spec), originalOptions)
			e := before.RuleSetExecution
			before.Results = utils.FilterIgnoredResultsWithOptions(before.Results, e.RepairIgnores, utils.IgnoreMatcherOptions{RootNode: e.CanonicalDocument, SpecBytes: e.Spec, LookupTimeout: e.NodeLookupTimeout})
			before.RepairIgnoresApplied = true
		}
		before.RepairWarnings = append(before.RepairWarnings, "repairs declined: "+err.Error())
		return plan, before
	}
	plugins := apply && execution.ApplyAutoFixes && len(execution.AutoFixFunctions) > 0
	if !apply || (!plugins && len(plan.Operations) == 0 && len(plan.KeyTags) == 0 && len(plan.KeyRemovals) == 0) {
		return plan, before
	}
	var verified *RepairVerification
	if plugins {
		verified, err = repairWithPlugins(execution, plan, options, before)
	} else {
		verified, err = validatePreparedPlan(execution, plan, options, before)
	}
	if err != nil {
		before.RepairWarnings = append(before.RepairWarnings, "repairs declined: "+err.Error())
		return plan, before
	}
	selected := make(map[string]bool)
	for _, target := range verified.Verified {
		selected[target.RuleID+"\x00"+target.Pointer] = true
	}
	locations, _ := baselineRepairFindings(before)
	for _, location := range locations {
		if location.Unavailable == "" && selected[location.RuleID+"\x00"+location.Pointer] {
			result := before.Results[location.Index]
			result.AutoFixed = true
			verified.Final.FixedResults = append(verified.Final.FixedResults, result)
		}
	}

	if !bytes.Equal(execution.Spec, verified.Candidate) {
		verified.Final.ModifiedSpec = verified.Candidate
	}
	verified.Final.RuleSetExecution.Spec = execution.Spec
	before.ReleaseOwnedResources()
	return plan, verified.Final
}

func repairRunOptions(options *ExecutionOptions) (*ExecutionOptions, context.CancelFunc) {
	var copy ExecutionOptions
	if options != nil {
		copy = *options
	}
	if copy.Context == nil {
		copy.Context = context.Background()
	}
	var cancel context.CancelFunc
	if copy.RunTimeout > 0 {
		copy.Context, cancel = context.WithTimeout(copy.Context, copy.RunTimeout)
	} else {
		copy.Context, cancel = context.WithCancel(copy.Context)
	}
	copy.RunTimeout = 0
	copy.ReportFunctionErrors = true
	return &copy, cancel
}
func repairCanceled(options *ExecutionOptions) error {
	if options != nil && options.Context != nil {
		return options.Context.Err()
	}
	return nil
}

func repairIgnoreScopes(root *yaml.Node) map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	ignores := buildInlineIgnoreIndex(root)
	if ignores == nil {
		return result
	}
	locations := repair.IndexLocations(root)
	for node, rules := range ignores.nodeIgnores {
		if path, ok := locations.Find(node); ok {
			result[path] = rules
		}
	}
	return result
}

// Only explicit line references are translated. Numbers that describe API
// constraints or values remain part of the diagnostic identity.
var diagnosticLineReference = regexp.MustCompile("(?i)(\\bline[ :]+[`'\"]?)([0-9]+)")

func originalDiagnosticLines(message string, correspondence *repair.Correspondence) string {
	return diagnosticLineReference.ReplaceAllStringFunc(message, func(part string) string {
		match := diagnosticLineReference.FindStringSubmatch(part)
		n, _ := strconv.Atoi(match[2])
		if original, ok := correspondence.OriginalLine(n); ok {
			return match[1] + strconv.Itoa(original)
		}
		return part
	})
}

// Resolver coordinates are presentation data. Canonical pointer and error text
// identify the defect when rendering moves an unchanged reference to a new line.
var resolverPosition = regexp.MustCompile(` \[([0-9]+):([0-9]+)\]$`)

func referenceDiagnosticMessage(message string) string {
	return resolverPosition.ReplaceAllString(message, "")
}
