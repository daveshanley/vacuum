package harness

import (
	"context"
	"errors"
	"os"

	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/plugin"
)

// ErrorCode identifies a failure independently of its human-readable message.
type ErrorCode string

const (
	ErrorSuite             ErrorCode = "invalid_suite"
	ErrorSelection         ErrorCode = "invalid_selection"
	ErrorInput             ErrorCode = "input_error"
	ErrorDocument          ErrorCode = "invalid_document"
	ErrorRuleset           ErrorCode = "ruleset_error"
	ErrorFunctionLoad      ErrorCode = "function_load_error"
	ErrorUnknownFunction   ErrorCode = "unknown_function"
	ErrorRuleConfiguration ErrorCode = "rule_configuration"
	ErrorFunction          ErrorCode = "function_error"
	ErrorAutoFix           ErrorCode = "autofix_error"
	ErrorRulePanic         ErrorCode = "rule_panic"
	ErrorRuleTimeout       ErrorCode = "rule_timeout"
	ErrorLookup            ErrorCode = "lookup_error"
	ErrorCancelled         ErrorCode = "cancelled"
	ErrorDeadline          ErrorCode = "deadline_exceeded"
	ErrorExecution         ErrorCode = "execution_error"
	ErrorEvidence          ErrorCode = "missing_evidence"
	ErrorPaths             ErrorCode = "incomplete_paths"
	ErrorDependency        ErrorCode = "dependency_error"
	ErrorIgnoreFile        ErrorCode = "ignore_file_error"
	ErrorBundle            ErrorCode = "bundle_error"
	ErrorArguments         ErrorCode = "invalid_arguments"
	ErrorWorker            ErrorCode = "worker_error"
	ErrorInternal          ErrorCode = "internal_error"
)

// Phase identifies the operation in which an execution error occurred.
type Phase string

const (
	PhaseLoad      Phase = "load"
	PhaseBefore    Phase = "before"
	PhaseAfter     Phase = "after"
	PhaseFixed     Phase = "fixed"
	PhaseSecondFix Phase = "secondFix"
	PhaseChange    Phase = "change"
	PhaseFixOutput Phase = "fixOutput"
	PhaseBundle    Phase = "bundle"
	PhaseCommand   Phase = "command"
)

// ExecutionError tells an agent what failed and where, without message parsing.
// Omitted identity fields were not known at the failure boundary.
type ExecutionError struct {
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Phase    Phase     `json:"phase"`
	CaseID   string    `json:"caseId,omitempty"`
	RuleID   string    `json:"ruleId,omitempty"`
	Function string    `json:"function,omitempty"`
	Source   string    `json:"source,omitempty"`
}

// NewExecutionError classifies typed causes and retains the supplied fallback
// code for errors whose producer does not expose a more specific error type.
func NewExecutionError(code ErrorCode, phase Phase, err error) ExecutionError {
	out := ExecutionError{Code: code, Phase: phase, Message: err.Error()}
	var rule *motor.RuleExecutionError
	var unknown *motor.UnknownFunctionError
	var timeout *motor.RuleTimeoutError
	var lookup *motor.RuleLookupError
	var load *plugin.FunctionLoadError
	var path *os.PathError
	if errors.As(err, &path) {
		out.Source = path.Path
	}
	switch {
	case errors.As(err, &load):
		out.Code = ErrorFunctionLoad
		out.Function = load.Function
		out.Source = load.Source
	case errors.As(err, &rule):
		switch rule.Kind {
		case motor.RuleErrorConfiguration:
			out.Code = ErrorRuleConfiguration
		case motor.RuleErrorFunction:
			out.Code = ErrorFunction
		case motor.RuleErrorAutoFix:
			out.Code = ErrorAutoFix
		case motor.RuleErrorPanic:
			out.Code = ErrorRulePanic
		}
		out.RuleID = rule.RuleID
		out.Function = rule.Function
	case errors.As(err, &unknown):
		out.Code = ErrorUnknownFunction
		out.RuleID = unknown.RuleID
		out.Function = unknown.Function
	case errors.As(err, &timeout):
		out.Code = ErrorRuleTimeout
		out.RuleID = timeout.RuleId
	case errors.As(err, &lookup):
		out.Code = ErrorLookup
		out.RuleID = lookup.RuleId
	}
	if errors.Is(err, context.Canceled) {
		out.Code = ErrorCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		out.Code = ErrorDeadline
	}
	return out
}

func (r *runner) executionErrors(err error, code ErrorCode, phase Phase, caseID, source string) []ExecutionError {
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var out []ExecutionError
		for _, cause := range joined.Unwrap() {
			out = append(out, r.executionErrors(cause, code, phase, caseID, source)...)
		}
		return out
	}
	out := NewExecutionError(code, phase, err)
	out.CaseID = caseID
	if out.Source != "" {
		out.Source = r.relative(out.Source)
	} else if source != "" {
		out.Source = r.relative(source)
	}
	return []ExecutionError{out}
}
