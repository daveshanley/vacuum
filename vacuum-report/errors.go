// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package vacuum_report

const ReportErrorTypeRuleLookup = "rule_lookup"

// ReportErrors contains non-fatal execution errors captured while building a
// machine-readable report. When present, report consumers should treat results
// as potentially incomplete.
type ReportErrors struct {
	Items []ReportError `json:"items" yaml:"items"`
}

// ReportError is a stable, serializable representation of an internal error.
type ReportError struct {
	Message string `json:"message" yaml:"message"`
	Type    string `json:"type,omitempty" yaml:"type,omitempty"`
	RuleId  string `json:"ruleId,omitempty" yaml:"ruleId,omitempty"`
	Given   string `json:"given,omitempty" yaml:"given,omitempty"`
}
