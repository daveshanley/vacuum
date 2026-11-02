package motor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

// RepairApproval supplies caller authority independently of a submitted plan.
// An empty ApprovedSHA256 permits only the exact built-in plan generated from
// the current findings. Explicitly reviewed plans require their full JSON hash.
type RepairApproval struct {
	ApprovedSHA256 string
	Policy         *repair.Policy
}

// RepairVerification owns the final lint result and the verified candidate.
// Release Final's owned resources when done. Candidate is nil on failure.
type RepairVerification struct {
	Candidate []byte                  `json:"candidate,omitempty"`
	Verified  []repair.Target         `json:"verified,omitempty"`
	Final     *RuleSetExecutionResult `json:"-"`
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
	if _, err := repair.ParseDocument(execution.Spec); err != nil {
		return repair.Plan{}, nil, err
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
	findings, err := repairFindings(e, baseline.Results)
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

// ValidateFixPlan applies a plan privately and relints it with the same input
// settings. It rejects unapproved edits, incomplete runs and new diagnostics.
// A plan without target findings can validate but never counts as a repair.
func ValidateFixPlan(execution *RuleSetExecution, plan repair.Plan, approval RepairApproval, options *ExecutionOptions) (*RepairVerification, error) {
	options, cancel := repairRunOptions(options)
	defer cancel()
	expected, before, err := PlanFixesWithPolicy(execution, options, approval.Policy)
	if before != nil {
		defer before.ReleaseOwnedResources()
	}
	if err != nil {
		return nil, err
	}
	return validatePreparedPlan(execution, plan, approval, options, expected, before)
}

func validatePreparedPlan(execution *RuleSetExecution, plan repair.Plan, approval RepairApproval, options *ExecutionOptions, expected repair.Plan, before *RuleSetExecutionResult) (*RepairVerification, error) {
	if err := repairCanceled(options); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	if approval.ApprovedSHA256 == "" {
		allowed, err := json.Marshal(expected)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(encoded, allowed) {
			return nil, fmt.Errorf("plan differs from the authorized built-in repairs")
		}
	} else if approval.ApprovedSHA256 != repair.Hash(encoded) {
		return nil, fmt.Errorf("reviewed plan hash does not match")
	}
	if approval.ApprovedSHA256 != "" {
		allowedCodes := make(map[string]string)
		for _, target := range expected.Targets {
			allowedCodes[target.RuleID+"\x00"+target.Pointer] = target.Code
		}
		findings, err := repairFindings(before.RuleSetExecution, before.Results)
		if err != nil {
			return nil, err
		}
		for _, target := range plan.Targets {
			valid := allowedCodes[target.RuleID+"\x00"+target.Pointer] == target.Code && target.Code != ""
			for _, finding := range findings {
				if finding.Unavailable == "" && finding.RuleID == target.RuleID && finding.Pointer == target.Pointer {
					code := finding.Code
					if code == "" {
						code = "reviewed-edit"
					}
					valid = valid || target.Code == code
				}
			}
			if !valid {
				return nil, fmt.Errorf("repair target code is not supported by the original diagnostic: %s", target.Code)
			}
		}
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
	return verifyRepairCandidate(execution, plan, candidate, options, before, sources, len(hashes) > 0)
}

func verifyRepairCandidate(execution *RuleSetExecution, plan repair.Plan, candidate []byte, options *ExecutionOptions, before *RuleSetExecutionResult, sources *repairSources, hasExternal bool) (*RepairVerification, error) {
	if err := repairCanceled(options); err != nil {
		return nil, err
	}
	parsed, err := repair.ParseDocument(candidate)
	if err != nil {
		return nil, err
	}
	originalIgnores, _ := json.Marshal(repairIgnoreScopes(before.RuleSetExecution.CanonicalDocument))
	candidateIgnores, _ := json.Marshal(repairIgnoreScopes(parsed))
	if !bytes.Equal(originalIgnores, candidateIgnores) {
		return nil, fmt.Errorf("repair cannot alter inline ignore directives")
	}
	e := repairExecution(execution, candidate)
	e.RolodexFS = sources
	if !hasExternal && externalRepairBytes(candidate) {
		return nil, fmt.Errorf("repair cannot introduce external inputs")
	}
	after := ApplyRulesToRuleSetWithOptions(e, options)
	fail := func(err error) (*RepairVerification, error) { after.ReleaseOwnedResources(); return nil, err }
	if len(after.Errors) > 0 {
		return fail(fmt.Errorf("repair validation did not complete: %w", after.Errors[0]))
	}
	after.Results = utils.FilterIgnoredResultsWithOptions(after.Results, e.RepairIgnores, utils.IgnoreMatcherOptions{RootNode: e.CanonicalDocument, SpecBytes: e.Spec, LookupTimeout: e.NodeLookupTimeout})
	baseline := make(map[string]int)
	for _, r := range before.Results {
		baseline[diagnosticIdentity(r)]++
	}
	for _, r := range after.Results {
		k := diagnosticIdentity(r)
		if baseline[k] == 0 {
			return fail(fmt.Errorf("repair introduces diagnostic %s at %s", r.RuleId, r.Path))
		}
		baseline[k]--
	}
	beforeFindings, err := repairFindings(before.RuleSetExecution, before.Results)
	if err != nil {
		return fail(err)
	}
	afterFindings, err := repairFindings(e, after.Results)
	if err != nil {
		return fail(err)
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
			remaining[f.RuleID+"\x00"+f.Pointer] = true
		}
	}
	for _, target := range plan.Targets {
		k := target.RuleID + "\x00" + target.Pointer
		if !original[k] {
			return fail(fmt.Errorf("repair target is not an original diagnostic: %s", target.RuleID))
		}
		if remaining[k] {
			return fail(fmt.Errorf("repair target remains: %s at %s", target.RuleID, target.Pointer))
		}
	}
	if err := repairCanceled(options); err != nil {
		return fail(err)
	}
	return &RepairVerification{Candidate: candidate, Verified: plan.Targets, Final: after}, nil
}

func repairExecution(source *RuleSetExecution, data []byte) *RuleSetExecution {
	e := *source
	e.Spec = append([]byte(nil), data...)
	e.SpecInfo = nil
	e.IndexUnresolved = nil
	e.IndexResolved = nil
	e.Document = nil
	e.DrDocument = nil
	e.Arazzo = nil
	e.AsyncAPI = nil
	e.CanonicalDocument = nil
	e.ApplyAutoFixes = false
	e.RuleSet = source.RuleSet.CopyForExecution()
	return &e
}
func diagnosticIdentity(r model.RuleFunctionResult) string {
	message := r.Message
	if r.DiagnosticCode != "" {
		message = r.DiagnosticCode
	}
	return r.RuleId + "\x00" + r.Path + "\x00" + message + "\x00" + r.RuleSeverity
}
func repairFindings(execution *RuleSetExecution, results []model.RuleFunctionResult) ([]repair.Finding, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(execution.Spec, &doc); err != nil {
		return nil, err
	}
	locations := repair.IndexLocations(&doc)
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
		if pointer, ok := locations.Find(node); ok {
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
				if key == "sourceDescriptions" {
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
		before = &RuleSetExecutionResult{RuleSetExecution: execution}
	}
	if err != nil {
		before.Errors = append(before.Errors, err)
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
		verified, err = validatePreparedPlan(execution, plan, RepairApproval{Policy: policy}, options, plan, before)
	}
	if err != nil {
		before.Errors = append(before.Errors, err)
		return plan, before
	}
	selected := make(map[string]bool)
	for _, target := range verified.Verified {
		selected[target.RuleID+"\x00"+target.Pointer] = true
	}
	locations, _ := repairFindings(before.RuleSetExecution, before.Results)
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
