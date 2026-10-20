package motor

import "sort"

// ExecutionEvidence records rule outcomes for one invocation. Supply a fresh
// value through ExecutionOptions; read it only after execution returns.
// Timed-out workers never publish partial counters into this value.
type ExecutionEvidence struct {
	// SampleLimit bounds source samples per selector. Zero collects counters only.
	SampleLimit int            `json:"sampleLimit"`
	Rules       []RuleEvidence `json:"rules"`
}

// RuleEvidence distinguishes completed checks from excluded or incomplete work.
// Matches count selector results, not semantic coverage inside a function.
type RuleEvidence struct {
	RuleID      string             `json:"ruleId"`
	Status      string             `json:"status"`
	Reason      string             `json:"reason,omitempty"`
	Resolved    bool               `json:"resolved"`
	Selectors   []SelectorEvidence `json:"selectors"`
	Matches     int                `json:"matches"`
	Invocations int                `json:"invocations"`
	Findings    int                `json:"findings"`
	Ignored     int                `json:"ignored"`
	Fixed       int                `json:"fixed"`
}

// SelectorEvidence describes one expanded selector against the execution tree.
type SelectorEvidence struct {
	Given     string             `json:"given"`
	Matches   int                `json:"matches"`
	Samples   []EvidenceLocation `json:"samples,omitempty"`
	Truncated bool               `json:"truncated,omitempty"`
}

// EvidenceLocation is a one-based location in the selected execution tree.
// It is not a canonical external-file origin; diagnostics carry those origins.
type EvidenceLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func prepareEvidence(control *executionControl, execution *RuleSetExecution) {
	if control.evidence == nil || execution == nil || execution.RuleSet == nil {
		return
	}
	control.evidence.Rules = make([]RuleEvidence, 0, len(execution.RuleSet.Rules))
	for id, rule := range execution.RuleSet.Rules {
		if rule == nil {
			continue
		}
		if rule.Id != "" {
			id = rule.Id
		}
		control.evidence.Rules = append(control.evidence.Rules, RuleEvidence{RuleID: id, Status: "excluded", Reason: "format", Selectors: []SelectorEvidence{}})
	}
	sort.Slice(control.evidence.Rules, func(i, j int) bool { return control.evidence.Rules[i].RuleID < control.evidence.Rules[j].RuleID })
}
