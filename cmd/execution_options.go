package cmd

import "github.com/daveshanley/vacuum/motor"

func newMotorExecutionOptions(resolveAllRefs, nestedRefsDocContext bool) *motor.ExecutionOptions {
	if !resolveAllRefs && !nestedRefsDocContext {
		return nil
	}
	return &motor.ExecutionOptions{
		ResolveAllRefs:       resolveAllRefs,
		NestedRefsDocContext: nestedRefsDocContext,
	}
}

func newMotorExecutionOptionsFromExecutionFlags(flags *ExecutionFlags) *motor.ExecutionOptions {
	if flags == nil {
		return nil
	}
	return newMotorExecutionOptions(flags.ResolveAllRefs, flags.NestedRefsDocContext)
}

func newMotorExecutionOptionsFromLintFlags(flags *LintFlags) *motor.ExecutionOptions {
	if flags == nil {
		return nil
	}
	return newMotorExecutionOptions(flags.ResolveAllRefs, flags.NestedRefsDocContext)
}
