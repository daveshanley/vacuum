package motor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/statistics"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

// FixOption describes one finding in a review. Index is stable for this review,
// independent of table sorting or filtering. Text metadata uses plain text;
// structured proposals use JSON. Compound transformations are not editable.
type FixOption struct {
	Index       int
	Result      *model.RuleFunctionResult
	Description string
	Available   bool // Available is independent of the caller's selection.
	Editable    bool
	Value       string
	Selected    bool
	Generated   bool // Value is a contextual suggestion that requires explicit selection.
	Reason      string
}

// FixChoice includes one finding in a preview. Value supplies an edited value;
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
	Impact    RepairImpact
	Candidate []byte
	Results   []*model.RuleFunctionResult
	Fixed     int
	Before    FixStats
	After     FixStats
	original  []byte
	fixed     []model.RuleFunctionResult
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
	policy    *repair.Policy
	proposal  repair.Plan
	findings  []repair.Finding
	choices   []FixOption
	edits     map[int]repair.Operation
	fields    map[int]string
}

// NewFixReview lints one snapshot and derives supported repair proposals.
// It performs no writes. Unsupported findings remain visible with reasons.
func NewFixReview(execution *RuleSetExecution, options *ExecutionOptions) (*FixReview, error) {
	return NewFixReviewWithPolicy(execution, options, nil)
}

// NewFixReviewWithPolicy includes exact caller-supplied values in the review.
func NewFixReviewWithPolicy(execution *RuleSetExecution, options *ExecutionOptions, policy *repair.Policy) (*FixReview, error) {
	if execution == nil || execution.RuleSet == nil {
		return nil, fmt.Errorf("fix review requires an execution and ruleset")
	}
	r := &FixReview{execution: repairExecution(execution, execution.Spec), policy: policy}
	if options != nil {
		r.options = *options
	}
	r.options.ReportFunctionErrors = true
	checked, cancel := repairRunOptions(&r.options)
	defer cancel()
	var err error
	r.proposal, r.baseline, err = PlanFixesWithPolicy(r.execution, checked, policy)
	if err != nil {
		r.Close()
		return nil, err
	}
	r.findings, err = repairFindings(r.baseline.RuleSetExecution, r.baseline.Results)
	if err != nil {
		r.Close()
		return nil, err
	}
	originalProposal := r.proposal
	reviewPlan, err := repair.ReviewedPlanWithSources(r.execution.SpecFileName, r.execution.Spec, r.findings, r.reviewSourceNodes())
	if err != nil {
		r.Close()
		return nil, err
	}
	reviewPlan, err = repair.ApplyPolicy(reviewPlan, r.execution.Spec, r.findings, policy)
	if err != nil {
		r.Close()
		return nil, err
	}
	reviewPlan.Sources = r.proposal.Sources
	r.proposal = reviewPlan
	r.edits = make(map[int]repair.Operation)
	r.fields = repair.EditableFields(r.execution.Spec, r.findings)
	// A frozen-source failure must also prevent metadata proposals.
	if _, _, err := freezeRepairSources(r.baseline); err != nil {
		r.fields = nil
		r.proposal = originalProposal
	}
	if r.fields == nil {
		r.fields = make(map[int]string)
	}
	available := make(map[string]repair.Target)
	declined := make(map[string]string)
	for _, finding := range r.findings {
		if len(r.proposal.ReviewedProposals[fixKey(finding.RuleID, finding.Pointer)]) > 0 {
			available[fixKey(finding.RuleID, finding.Pointer)] = repair.Target{RuleID: finding.RuleID, Pointer: finding.Pointer, Code: "reviewed-proposal"}
		}
	}
	for _, t := range r.proposal.Targets {
		available[fixKey(t.RuleID, t.Pointer)] = t
	}
	for _, d := range r.proposal.Declined {
		declined[fixKey(d.Target.RuleID, d.Target.Pointer)] = d.Reason
	}
	root := r.baseline.RuleSetExecution.CanonicalDocument
	var usedIDs map[string]bool
	var usedDescriptions map[string]bool
	for _, name := range r.fields {
		if name == "operationId" {
			usedIDs = fixOperationIDs(root)
			break
		}
	}
	for _, finding := range r.findings {
		result := r.baseline.Results[finding.Index]
		option := FixOption{Index: finding.Index, Result: &result, Reason: declined[fixKey(finding.RuleID, finding.Pointer)]}
		if target, ok := available[fixKey(finding.RuleID, finding.Pointer)]; ok {
			option.Available = true
			option.Generated = target.Code == "reviewed-proposal"
			option.Description = "Apply " + strings.ReplaceAll(target.Code, "-", " ")
			option.Reason = ""
			if op, ok := editableProposalOperation(r.proposal, finding.RuleID, finding.Pointer, r.fields[finding.Index]); ok {
				option.Editable = true
				option.Generated = true
				option.Value = string(op.Value)
				option.Description = "Edit proposed JSON value"
				r.fields[finding.Index] = "$proposal-json"
				var text string
				if len(op.Value) > 0 && op.Value[0] == '"' && json.Unmarshal(op.Value, &text) == nil {
					option.Value = text
					option.Description = "Edit proposed value"
					r.fields[finding.Index] = "$proposal-string"
				}
				r.edits[finding.Index] = op
			}
		} else if name := r.fields[finding.Index]; name != "" {
			option.Available = true
			option.Editable = true
			option.Description = "Enter " + name
			if name == "contact" || name == "license" {
				option.Description += " as a JSON object"
			}
			option.Reason = "Value required"
			option.Value = suggestedFixValue(name, finding.Pointer, fixNodeAt(root, finding.Pointer), usedIDs)
			if name == "description" || name == "summary" {
				if usedDescriptions == nil {
					usedDescriptions = fixDescriptionValues(root)
				}
				option.Value = uniqueFixDescription(option.Value, name, usedDescriptions)
			}
			if option.Value != "" {
				option.Generated = true
				option.Description = "Add " + name
				option.Reason = "Generated suggestion; review before selecting"
			}
		} else if option.Reason == "" {
			option.Reason = "No supported automatic repair"
		}
		if !option.Available && finding.RuleID == "component-description" &&
			strings.HasPrefix(fixScalar(root, "openapi"), "3.0.") &&
			fixNodeAt(fixNodeAt(root, finding.Pointer), "/$ref") != nil {
			option.Reason = "OpenAPI 3.0 reference metadata must be edited at the referenced source"
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
	plan, err := r.planChoices(choices)
	if err != nil {
		return nil, err
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
	targets := make(map[string]bool, len(plan.Targets))
	for _, target := range plan.Targets {
		targets[fixKey(target.RuleID, target.Pointer)] = true
	}
	var fixed []model.RuleFunctionResult
	for _, f := range r.findings {
		if targets[fixKey(f.RuleID, f.Pointer)] {
			result := r.baseline.Results[f.Index]
			result.AutoFixed = true
			fixed = append(fixed, result)
		}
	}
	return &FixPreview{original: append([]byte(nil), r.execution.Spec...), fixed: fixed, Impact: assessRepairImpact(r.baseline, verification.Final, options), Candidate: verification.Candidate, Results: results, Fixed: len(targets), Before: fixStats(r.baseline.Results), After: fixStats(verification.Final.Results), final: verification.Final}, nil
}

// planChoices derives providers and policy values from the current selection.
// The caller holds r.mu and must verify the result before any write.
func (r *FixReview) planChoices(choices []FixChoice) (repair.Plan, error) {
	if len(choices) == 0 {
		return repair.Plan{}, fmt.Errorf("select at least one fix")
	}
	selected := make([]repair.Finding, 0, len(choices))
	entries := make([]repair.PolicyEntry, 0)
	overrides := make(map[string]json.RawMessage)
	seen := make(map[int]bool)
	for _, choice := range choices {
		if choice.Index < 0 || choice.Index >= len(r.choices) || seen[choice.Index] {
			return repair.Plan{}, fmt.Errorf("invalid or repeated fix selection %d", choice.Index)
		}
		seen[choice.Index] = true
		option := r.choices[choice.Index]
		finding := r.findings[choice.Index]
		if !option.Available {
			return repair.Plan{}, fmt.Errorf("fix is unavailable: %s", option.Reason)
		}
		if option.Editable {
			value := option.Value
			if choice.Value != nil {
				value = *choice.Value
			}
			if op, ok := r.edits[choice.Index]; ok {
				encoded := json.RawMessage(value)
				if r.fields[choice.Index] == "$proposal-string" {
					encoded, _ = json.Marshal(value)
				}
				if !json.Valid(encoded) {
					return repair.Plan{}, fmt.Errorf("proposed value must be valid JSON")
				}
				if old, exists := overrides[op.Path]; exists && string(old) != string(encoded) {
					return repair.Plan{}, fmt.Errorf("conflicting proposed values at %s", op.Path)
				}
				overrides[op.Path] = encoded
				selected = append(selected, finding)
				continue
			}
			node, err := fixValue(r.fields[choice.Index], value)
			if err != nil {
				return repair.Plan{}, err
			}
			entries = append(entries, repair.PolicyEntry{ID: fmt.Sprintf("review-%d", choice.Index), Rule: finding.RuleID, Diagnostic: r.fields[choice.Index] + "-missing", Scope: repair.PolicyScope{Source: r.execution.SpecFileName, Pointer: finding.Pointer}, Value: *node})
		} else if choice.Value != nil {
			return repair.Plan{}, fmt.Errorf("mechanical fix values cannot be edited")
		}
		selected = append(selected, finding)
	}
	plan, err := repair.ReviewedPlanWithSources(r.execution.SpecFileName, r.execution.Spec, selected, r.reviewSourceNodes())
	if err != nil {
		return repair.Plan{}, err
	}
	plan, err = repair.ApplyPolicy(plan, r.execution.Spec, selected, r.policy)
	if err != nil {
		return repair.Plan{}, err
	}
	var policy *repair.Policy
	if len(entries) != 0 {
		// Parse the same exact-scope policy contract used by noninteractive repair.
		for i := range entries {
			entries[i].Scope.Source, err = filepath.Abs(entries[i].Scope.Source)
			if err != nil {
				return repair.Plan{}, err
			}
		}
		encoded, err := yaml.Marshal(struct {
			Version  int                  `yaml:"version"`
			Policies []repair.PolicyEntry `yaml:"policies"`
		}{1, entries})
		if err != nil {
			return repair.Plan{}, err
		}
		policy, err = repair.ParsePolicy(encoded, "", nil)
		if err != nil {
			return repair.Plan{}, err
		}
		plan, err = repair.ApplyPolicy(plan, r.execution.Spec, selected, policy)
		if err != nil {
			return repair.Plan{}, err
		}
	}
	for i := range plan.Operations {
		if value, ok := overrides[plan.Operations[i].Path]; ok && (plan.Operations[i].Op == "add" || plan.Operations[i].Op == "replace") {
			plan.Operations[i].Value = value
		}
	}
	plan.Sources = r.proposal.Sources
	targets := make(map[string]bool)
	for _, target := range plan.Targets {
		targets[fixKey(target.RuleID, target.Pointer)] = true
	}
	for _, finding := range selected {
		if !targets[fixKey(finding.RuleID, finding.Pointer)] {
			return repair.Plan{}, fmt.Errorf("selected fix cannot be applied together: %s at %s", finding.RuleID, finding.Pointer)
		}
	}
	return plan, nil
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

// FixDraft shows one proposed edit. It is not verified and cannot be applied;
// Preview must verify the complete selection before its candidate is written.
// Before and After are bounded, normalized excerpts of the affected owner.
// YAML and JSON retain their source format but may use normalized indentation.
type FixDraft struct {
	Before, After string
	Pointer       string
	Generated     bool
	NeedsValue    bool // The editable field has no proposed value; After is empty.
}

// Draft applies one selected provider in memory without running another lint.
// Call it when opening fix details or changing a value, not while rendering.
func (r *FixReview) Draft(choice FixChoice) (*FixDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.baseline == nil {
		return nil, fmt.Errorf("fix review is closed")
	}

	if choice.Index < 0 || choice.Index >= len(r.choices) {
		return nil, fmt.Errorf("invalid fix selection %d", choice.Index)
	}
	option := r.choices[choice.Index]
	value := option.Value
	if choice.Value != nil {
		value = *choice.Value
	}
	pointer := r.findings[choice.Index].Pointer
	root := r.baseline.RuleSetExecution.CanonicalDocument
	isJSON := json.Valid(r.execution.Spec)
	format := utils.CaptureRepairYAMLFormat(root)
	scope := fixDraftOwnerScope(root, pointer)
	if option.Editable && strings.TrimSpace(value) == "" {
		before, err := renderFixOwner(root, scope, isJSON, r.execution.Spec, format)
		if err != nil {
			return nil, err
		}
		excerpt, _ := fixDraftExcerpts(before, nil, 1)
		return &FixDraft{Before: excerpt, Pointer: scope, NeedsValue: true}, nil
	}

	plan, err := r.planChoices([]FixChoice{choice})
	if err != nil {
		return nil, err
	}
	candidate, err := repair.Apply(r.execution.SpecFileName, r.execution.Spec, plan)
	if err != nil {
		return nil, err
	}
	next, err := repair.ParseDocument(candidate)
	if err != nil {
		return nil, err
	}
	scope = fixDraftOwnerScope(root, fixDraftScope(pointer, plan))
	before, err := renderFixOwner(root, scope, isJSON, r.execution.Spec, format)
	if err != nil {
		return nil, err
	}
	after, err := renderFixOwner(next, scope, isJSON, candidate, format)
	if err != nil {
		return nil, err
	}
	a, b := fixDraftExcerpts(before, after, 1)
	return &FixDraft{Before: a, After: b, Pointer: scope, Generated: option.Generated && (choice.Value == nil || *choice.Value == option.Value)}, nil
}

// Include every changed path in the displayed owner. A removed or moved child
// cannot serve as the after-location because its previous sibling index may move.
func fixDraftScope(pointer string, plan repair.Plan) string {
	common := func(a, b string) string {
		for a != "" && b != a && !strings.HasPrefix(b, a+"/") {
			a = a[:strings.LastIndex(a, "/")]
		}
		return a
	}
	for _, op := range plan.Operations {
		path := op.Path
		if (op.Op == "remove" || op.Op == "move") && path != "" {
			path = path[:strings.LastIndex(path, "/")]
		}
		pointer = common(pointer, path)
		if op.Op == "move" || op.Op == "copy" {
			pointer = common(pointer, op.From)
		}
	}
	for _, edit := range plan.KeyRemovals {
		pointer = common(pointer, edit.Parent)
	}
	for _, edit := range plan.KeyTags {
		pointer = common(pointer, edit.Parent)
	}
	return pointer
}

// A sequence element has no mapping key of its own. Show its containing
// sequence instead of inventing a numeric key that is absent from the source.
func fixDraftOwnerScope(root *yaml.Node, pointer string) string {
	for pointer != "" {
		i := strings.LastIndex(pointer, "/")
		parent := fixNodeAt(root, pointer[:i])
		if parent == nil || parent.Kind != yaml.SequenceNode {
			break
		}
		pointer = pointer[:i]
	}
	return pointer
}

func renderFixOwner(root *yaml.Node, pointer string, isJSON bool, source []byte, format utils.RepairYAMLFormat) ([]byte, error) {
	owner := fixNodeAt(root, pointer)
	if owner == nil {
		return nil, fmt.Errorf("fix has no source context at %s", pointer)
	}
	if i := strings.LastIndex(pointer, "/"); i >= 0 {
		name := strings.ReplaceAll(strings.ReplaceAll(pointer[i+1:], "~1", "/"), "~0", "~")
		key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}
		parent := fixNodeAt(root, pointer[:i])
		if parent != nil && parent.Kind == yaml.MappingNode {
			for j := 0; j+1 < len(parent.Content); j += 2 {
				if parent.Content[j].Value == name {
					copy := *parent.Content[j]
					key = &copy
					break
				}
			}
		}
		owner = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{key, owner}}
	}
	if isJSON {
		// Duplicate-path repair must show the duplicated original members. Keep
		// this display-only path separate from the strict repaired JSON renderer.
		if pointer == "/paths" {
			var document struct {
				Paths json.RawMessage `json:"paths"`
			}
			if err := json.Unmarshal(source, &document); err != nil {
				return nil, err
			}
			raw := append([]byte("{\"paths\":"), document.Paths...)
			raw = append(raw, '}')
			var out bytes.Buffer
			if err := json.Indent(&out, raw, "", "  "); err != nil {
				return nil, err
			}
			return out.Bytes(), nil
		}
		return utils.RenderRepairJSON(owner, []byte("{\n  \"value\": null\n}\n"))
	}
	return utils.RenderRepairYAML(owner, nil, format)
}

func fixNodeAt(root *yaml.Node, pointer string) *yaml.Node {
	if root == nil {
		return nil
	}
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) != 1 {
			return nil
		}
		root = root.Content[0]
	}
	if pointer == "" {
		return root
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil
	}
	for _, part := range strings.Split(pointer[1:], "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		switch root.Kind {
		case yaml.MappingNode:
			var found *yaml.Node
			for i := 0; i+1 < len(root.Content); i += 2 {
				if root.Content[i].Value == part {
					found = root.Content[i+1]
					break
				}
			}
			if found == nil {
				return nil
			}
			root = found
		case yaml.SequenceNode:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(root.Content) {
				return nil
			}
			root = root.Content[index]
		default:
			return nil
		}
	}
	return root
}

func fixScalar(root *yaml.Node, name string) string {
	if n := fixNodeAt(root, "/"+repair.EscapeToken(name)); n != nil && n.Kind == yaml.ScalarNode && n.Tag == "!!str" {
		return strings.TrimSpace(n.Value)
	}
	return ""
}

func fixOperationIDs(root *yaml.Node) map[string]bool {
	used := make(map[string]bool)
	var walk func(*yaml.Node)
	walk = func(n *yaml.Node) {
		if n == nil {
			return
		}
		if n.Kind == yaml.MappingNode {
			if id := fixScalar(n, "operationId"); id != "" {
				used[id] = true
			}
		}
		for _, child := range n.Content {
			walk(child)
		}
	}
	walk(root)
	return used
}

func suggestedFixValue(field, pointer string, owner *yaml.Node, usedIDs map[string]bool) string {
	if field == "contact" {
		name := fixScalar(owner, "title")
		if name == "" {
			name = "API"
		}
		value, _ := json.Marshal(map[string]string{"name": name + " support"})
		return string(value)
	}
	if field == "license" {
		// Do not infer a legal grant from the API's metadata.
		return `{"name":"Unspecified"}`
	}
	parts := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	for i := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(parts[i], "~1", "/"), "~0", "~")
	}
	if field == "operationId" {
		id := suggestedOperationID(pointer)
		if id == "" {
			id = "operation"
		}
		base := id
		for suffix := 2; usedIDs[id]; suffix++ {
			id = base + strconv.Itoa(suffix)
		}
		usedIDs[id] = true
		return id
	}
	if field != "description" && field != "summary" {
		return ""
	}
	if len(parts) >= 3 && parts[0] == "paths" {
		// A path-level parameter applies to all methods. An operation parameter
		// has a method owner; both need context because names recur across paths.
		if name, location := fixScalar(owner, "name"), fixScalar(owner, "in"); name != "" && location != "" {
			context := parts[1]
			if parts[2] != "parameters" {
				context = strings.ToUpper(parts[2]) + " " + context
			}
			return name + " " + location + " parameter for " + context + "."
		}
		operation := strings.ToUpper(parts[2]) + " " + parts[1]
		if len(parts) == 3 {
			return operation + "."
		}
		if parts[len(parts)-1] == "requestBody" {
			if content := fixNodeAt(owner, "/content"); content != nil && content.Kind == yaml.MappingNode && len(content.Content) == 2 {
				if schema := fixNodeAt(content.Content[1], "/schema"); fixScalar(schema, "format") == "binary" {
					return "Binary request body for " + operation + "."
				}
			}
			return "Request body for " + operation + "."
		}
		if len(parts) == 5 && parts[3] == "responses" {
			return "Response " + parts[4] + " for " + operation + "."
		}
	}
	if len(parts) > 0 && parts[len(parts)-1] == "info" {
		if title := fixScalar(owner, "title"); title != "" {
			return title + " API."
		}
	}
	name := fixScalar(owner, "name")
	if name == "" {
		name = fixScalar(owner, "workflowId")
	}
	if name == "" {
		name = fixScalar(owner, "stepId")
	}
	if len(parts) == 3 && parts[0] == "components" {
		return parts[2] + " " + strings.TrimSuffix(parts[1], "s") + "."
	}
	if name != "" {
		if len(parts) > 1 && parts[len(parts)-2] == "tags" {
			return "Operations tagged " + name + "."
		}
		if location := fixScalar(owner, "in"); location != "" {
			return name + " " + location + " parameter."
		}
		return name + "."
	}
	if len(parts) == 2 && (parts[0] == "operations" || parts[0] == "channels") {
		return parts[1] + " " + strings.TrimSuffix(parts[0], "s") + "."
	}
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return strings.ToUpper(field[:1]) + field[1:] + " for " + strings.Join(parts, " / ") + "."
	}
	return "API " + field + "."
}

// Locate the changed range without a line slice for every line in a large spec.
// Rendering is capped independently on each side and omitted data is explicit.
func fixDraftExcerpts(before, after []byte, ownerLine int) (string, string) {
	next := func(data []byte, offset int) int {
		if n := bytes.IndexByte(data[offset:], '\n'); n >= 0 {
			return offset + n + 1
		}
		return len(data)
	}
	previous := func(data []byte, end int) int {
		if end > 0 && data[end-1] == '\n' {
			end--
		}
		return bytes.LastIndexByte(data[:end], '\n') + 1
	}
	first, line, ownerStart := 0, 1, -1
	for first < len(before) && first < len(after) {
		if line == ownerLine {
			ownerStart = first
		}
		a, b := next(before, first), next(after, first)
		if !bytes.Equal(before[first:a], after[first:b]) {
			break
		}
		first = a
		line++
	}
	endA, endB := len(before), len(after)
	for endA > first && endB > first {
		a, b := previous(before, endA), previous(after, endB)
		if a < first || b < first || !bytes.Equal(before[a:endA], after[b:endB]) {
			break
		}
		endA, endB = a, b
	}
	start := first
	for n := 0; n < 3 && start > 0; n++ {
		start = previous(before, start)
	}
	for n := 0; n < 3 && endA < len(before); n++ {
		endA = next(before, endA)
	}
	for n := 0; n < 3 && endB < len(after); n++ {
		endB = next(after, endB)
	}

	// Include a short owner's key and body so an appended description has a
	// visible subject. Large owners keep the nearby change and separate pointer.
	fits := func(data []byte) bool {
		lines := bytes.Count(data, []byte{'\n'})
		if len(data) > 0 && data[len(data)-1] != '\n' {
			lines++
		}
		return lines <= 40 && len(data) <= 32768
	}
	if ownerStart >= 0 && ownerStart < start && fits(before[ownerStart:endA]) && fits(after[ownerStart:endB]) {
		start = ownerStart
	}
	render := func(data []byte) string {
		var out strings.Builder
		for offset, line := 0, 0; offset < len(data); line++ {
			if line == 40 || out.Len() >= 32768 {
				out.WriteString("… more lines omitted\n")
				break
			}
			end := next(data, offset)
			part := bytes.TrimSuffix(data[offset:end], []byte{'\n'})
			part = bytes.TrimSuffix(part, []byte{'\r'})
			truncated := len(part) > 4096
			if truncated {
				limit := 4096
				for limit > 0 && !utf8.RuneStart(part[limit]) {
					limit--
				}
				part = part[:limit]
			}
			out.WriteString(strings.Map(func(r rune) rune {
				if (r < 32 && r != '\t' && r != '\n') || r == 127 {
					return '�'
				}
				return r
			}, string(part)))
			if truncated {
				out.WriteString("… line truncated")
			}
			out.WriteByte('\n')
			offset = end
		}
		return strings.TrimRight(out.String(), "\n")
	}
	return render(before[start:endA]), render(after[start:endB])
}

func suggestedOperationID(pointer string) string {
	parts := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	if len(parts) != 3 || parts[0] != "paths" {
		return ""
	}
	path := strings.ReplaceAll(strings.ReplaceAll(parts[1], "~1", "/"), "~0", "~")
	words := strings.FieldsFunc(parts[2]+" "+path, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	if len(words) == 0 {
		return ""
	}
	id := strings.ToLower(words[0])
	for _, word := range words[1:] {
		chars := []rune(word)
		chars[0] = unicode.ToUpper(chars[0])
		id += string(chars)
	}
	return id
}

// reviewSourceNodes exposes only sources already loaded and frozen by discovery.
func (r *FixReview) reviewSourceNodes() map[string]*yaml.Node {
	if r.baseline != nil && r.baseline.RuleSetExecution.Arazzo != nil {
		return r.baseline.RuleSetExecution.Arazzo.Sources
	}
	return nil
}
