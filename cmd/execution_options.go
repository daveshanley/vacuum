package cmd

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
)

func newMotorExecutionOptions(resolveAllRefs, nestedRefsDocContext bool, contracts ...model.Contract) *motor.ExecutionOptions {
	contract := model.NativeContract
	if len(contracts) > 0 {
		contract = contracts[0]
	}
	if !resolveAllRefs && !nestedRefsDocContext && contract == model.NativeContract {
		return nil
	}
	return &motor.ExecutionOptions{
		Contract:             contract,
		ResolveAllRefs:       resolveAllRefs,
		NestedRefsDocContext: nestedRefsDocContext,
	}
}

func newMotorExecutionOptionsFromExecutionFlags(flags *ExecutionFlags) *motor.ExecutionOptions {
	if flags == nil {
		return nil
	}
	return newMotorExecutionOptions(flags.ResolveAllRefs, flags.NestedRefsDocContext, flags.Contract)
}

func newMotorExecutionOptionsFromLintFlags(flags *LintFlags) *motor.ExecutionOptions {
	if flags == nil {
		return nil
	}
	return newMotorExecutionOptions(flags.ResolveAllRefs, flags.NestedRefsDocContext, flags.Contract)
}
