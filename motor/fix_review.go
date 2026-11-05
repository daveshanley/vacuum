package motor

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/statistics"
	"github.com/pb33f/go-yaml"
)

// FixOption describes one finding in a review. Index is stable for this review,
// independent of table sorting or filtering. Editable values are plain text,
// except contact and license values, which are JSON objects.
type FixOption struct {
	Index       int
	Result      *model.RuleFunctionResult
	Description string
	Editable    bool
	Value       string
	Selected    bool
	Reason      string
}

// FixChoice includes one finding in a preview. Value supplies edited metadata;
// nil keeps the proposed default. Omitted findings are not selected.
type FixChoice struct {
	Index int
	Value *string
}

// FixStats uses the same severity counts and quality score as ordinary lint.
type FixStats struct {
	Errors, Warnings, Info, Hints int
	Score                         int
}

// FixPreview holds a verified candidate and its final findings. Close releases
// their parser resources. It never writes files and grants no write authority.
type FixPreview struct {
	Candidate []byte
	Results   []*model.RuleFunctionResult
	Fixed     int
	Before    FixStats
	After     FixStats
	final     *RuleSetExecutionResult
}

// Close releases the preview's lint resources. It is safe to call repeatedly.
func (p *FixPreview) Close() {
	if p != nil && p.final != nil {
		p.final.ReleaseOwnedResources()
		p.final = nil
	}
}

// FixReview owns one source snapshot and baseline lint. Selection and edited
// values belong to its caller. Close the review when it is replaced or dismissed.
// Preview and Close are serialized so in-flight verification retains its inputs.
type FixReview struct {
	mu        sync.Mutex
	execution *RuleSetExecution
	options   ExecutionOptions
	baseline  *RuleSetExecutionResult
	proposal  repair.Plan
	findings  []repair.Finding
	choices   []FixOption
	fields    map[int]string
}

// NewFixReview lints one snapshot and derives supported mechanical and metadata
// fixes. It performs no writes. Unsupported findings remain visible with reasons.
func NewFixReview(execution *RuleSetExecution, options *ExecutionOptions) (*FixReview, error) {
	if execution == nil || execution.RuleSet == nil {
		return nil, fmt.Errorf("fix review requires an execution and ruleset")
	}
	r := &FixReview{execution: repairExecution(execution, execution.Spec)}
	if options != nil {
		r.options = *options
	}
	r.options.ReportFunctionErrors = true
	checked, cancel := repairRunOptions(&r.options)
	defer cancel()
	var err error
	r.proposal, r.baseline, err = PlanFixes(r.execution, checked)
	if err != nil {
		r.Close()
		return nil, err
	}
	r.findings, err = repairFindings(r.baseline.RuleSetExecution, r.baseline.Results)
	if err != nil {
		r.Close()
		return nil, err
	}
	r.fields = repair.EditableFields(r.execution.Spec, r.findings)
	// A frozen-source failure must also prevent metadata proposals.
	if _, _, err := freezeRepairSources(r.baseline); err != nil {
		r.fields = nil
	}
	available := make(map[string]repair.Target)
	declined := make(map[string]string)
	for _, t := range r.proposal.Targets {
		available[fixKey(t.RuleID, t.Pointer)] = t
	}
	for _, d := range r.proposal.Declined {
		declined[fixKey(d.Target.RuleID, d.Target.Pointer)] = d.Reason
	}
	for _, finding := range r.findings {
		result := r.baseline.Results[finding.Index]
		option := FixOption{Index: finding.Index, Result: &result, Reason: declined[fixKey(finding.RuleID, finding.Pointer)]}
		if target, ok := available[fixKey(finding.RuleID, finding.Pointer)]; ok {
			option.Selected = true
			option.Description = "Apply " + strings.ReplaceAll(target.Code, "-", " ")
			option.Reason = ""
		} else if name := r.fields[finding.Index]; name != "" {
			option.Editable = true
			option.Description = "Enter " + name
			if name == "contact" || name == "license" {
				option.Description += " as a JSON object"
			}
			option.Reason = "Value required"
		} else if option.Reason == "" {
			option.Reason = "No supported automatic repair"
		}
		r.choices = append(r.choices, option)
	}
	return r, nil
}

func fixKey(rule, pointer string) string { return rule + "\x00" + pointer }

// Options returns independent display records. Their parser nodes remain valid
// until Close; callers must not modify those nodes.
func (r *FixReview) Options() []FixOption {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := append([]FixOption(nil), r.choices...)
	for i := range out {
		result := *out[i].Result
		out[i].Result = &result
	}
	return out
}

// Close releases the original lint resources and makes further previews fail.
func (r *FixReview) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.baseline != nil {
		r.baseline.ReleaseOwnedResources()
		r.baseline = nil
		r.choices = nil
		r.findings = nil
		r.fields = nil
	}
}

// Preview applies selected fixes to private bytes and relints the candidate.
// Every selected finding must be resolved without introducing new diagnostics.
// An edited selection never reuses the candidate from an earlier preview.
func (r *FixReview) Preview(choices []FixChoice) (*FixPreview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.baseline == nil {
		return nil, fmt.Errorf("fix review is closed")
	}
	if len(choices) == 0 {
		return nil, fmt.Errorf("select at least one fix")
	}
	selected := make([]repair.Finding, 0, len(choices))
	entries := make([]repair.PolicyEntry, 0)
	seen := make(map[int]bool)
	for _, choice := range choices {
		if choice.Index < 0 || choice.Index >= len(r.choices) || seen[choice.Index] {
			return nil, fmt.Errorf("invalid or repeated fix selection %d", choice.Index)
		}
		seen[choice.Index] = true
		option := r.choices[choice.Index]
		finding := r.findings[choice.Index]
		if !option.Selected && !option.Editable {
			return nil, fmt.Errorf("fix is unavailable: %s", option.Reason)
		}
		if option.Editable {
			value := option.Value
			if choice.Value != nil {
				value = *choice.Value
			}
			node, err := fixValue(r.fields[choice.Index], value)
			if err != nil {
				return nil, err
			}
			entries = append(entries, repair.PolicyEntry{ID: fmt.Sprintf("review-%d", choice.Index), Rule: finding.RuleID, Diagnostic: r.fields[choice.Index] + "-missing", Scope: repair.PolicyScope{Source: r.execution.SpecFileName, Pointer: finding.Pointer}, Value: *node})
		} else if choice.Value != nil {
			return nil, fmt.Errorf("mechanical fix values cannot be edited")
		}
		selected = append(selected, finding)
	}
	plan, err := repair.BuiltinPlan(r.execution.SpecFileName, r.execution.Spec, selected)
	if err != nil {
		return nil, err
	}
	var policy *repair.Policy
	if len(entries) != 0 {
		// Parse the same exact-scope policy contract used by noninteractive repair.
		for i := range entries {
			entries[i].Scope.Source, err = filepath.Abs(entries[i].Scope.Source)
			if err != nil {
				return nil, err
			}
		}
		encoded, err := yaml.Marshal(struct {
			Version  int                  `yaml:"version"`
			Policies []repair.PolicyEntry `yaml:"policies"`
		}{1, entries})
		if err != nil {
			return nil, err
		}
		policy, err = repair.ParsePolicy(encoded, "", nil)
		if err != nil {
			return nil, err
		}
		plan, err = repair.ApplyPolicy(plan, r.execution.Spec, selected, policy)
		if err != nil {
			return nil, err
		}
	}
	plan.Sources = r.proposal.Sources
	targets := make(map[string]bool)
	for _, target := range plan.Targets {
		targets[fixKey(target.RuleID, target.Pointer)] = true
	}
	for _, finding := range selected {
		if !targets[fixKey(finding.RuleID, finding.Pointer)] {
			return nil, fmt.Errorf("selected fix cannot be applied together: %s at %s", finding.RuleID, finding.Pointer)
		}
	}
	options, cancel := repairRunOptions(&r.options)
	defer cancel()
	// This exact proposal was derived from the user's selected providers and
	// supplied values; no arbitrary patch operations are accepted by this API.
	verification, err := validatePreparedPlan(r.execution, plan, options, r.baseline)
	if err != nil {
		return nil, err
	}
	results := make([]*model.RuleFunctionResult, len(verification.Final.Results))
	for i := range results {
		results[i] = &verification.Final.Results[i]
	}
	return &FixPreview{Candidate: verification.Candidate, Results: results, Fixed: len(targets), Before: fixStats(r.baseline.Results), After: fixStats(verification.Final.Results), final: verification.Final}, nil
}

func fixValue(field, value string) (*yaml.Node, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("%s requires a nonempty value", field)
	}
	if len(value) > 1<<20 {
		return nil, fmt.Errorf("fix value exceeds 1 MiB")
	}
	if field != "contact" && field != "license" {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}, nil
	}
	var object map[string]string
	if !json.Valid([]byte(value)) || json.Unmarshal([]byte(value), &object) != nil || len(object) == 0 {
		return nil, fmt.Errorf("%s requires a JSON object with string values", field)
	}
	allowed := map[string]bool{"name": true, "url": true}
	if field == "contact" {
		allowed["email"] = true
	}
	for key, text := range object {
		if !allowed[key] || strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("unsupported or empty %s member %q", field, key)
		}
	}
	if field == "license" && strings.TrimSpace(object["name"]) == "" {
		return nil, fmt.Errorf("license requires a name")
	}
	doc, err := repair.ParseDocument([]byte(value))
	if err != nil {
		return nil, err
	}
	if len(doc.Content[0].Content) != 2*len(object) {
		return nil, fmt.Errorf("%s contains duplicate members", field)
	}
	return doc.Content[0], nil
}

func fixStats(results []model.RuleFunctionResult) FixStats {
	set := model.NewRuleResultSet(results)
	return FixStats{Score: statistics.CalculateQualityScore(set), Errors: set.GetErrorCount(), Warnings: set.GetWarnCount(), Info: set.GetInfoCount(), Hints: set.GetHintCount()}
}
