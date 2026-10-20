package motor

import "fmt"

// RuleErrorKind identifies the operation that failed during strict execution.
type RuleErrorKind string

const (
	RuleErrorConfiguration RuleErrorKind = "rule_configuration"
	RuleErrorFunction      RuleErrorKind = "function_error"
	RuleErrorAutoFix       RuleErrorKind = "autofix_error"
	RuleErrorPanic         RuleErrorKind = "rule_panic"
)

// RuleExecutionError retains rule and function identity without parsing messages.
// Err remains available through errors.Is and errors.As.
type RuleExecutionError struct {
	Kind     RuleErrorKind
	RuleID   string
	Function string
	Err      error
}

func (e *RuleExecutionError) Error() string {
	if e.Function != "" {
		return fmt.Sprintf("rule %q function %q: %v", e.RuleID, e.Function, e.Err)
	}
	return fmt.Sprintf("rule %q: %v", e.RuleID, e.Err)
}
func (e *RuleExecutionError) Unwrap() error { return e.Err }
