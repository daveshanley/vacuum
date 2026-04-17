// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"time"

	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/utils"
)

func buildIgnoreFilterOptions(
	specBytes []byte,
	executionResult *motor.RuleSetExecutionResult,
	lookupTimeoutFlag int,
) utils.IgnoreMatcherOptions {
	options := utils.IgnoreMatcherOptions{
		SpecBytes:     specBytes,
		LookupTimeout: time.Duration(lookupTimeoutFlag) * time.Millisecond,
	}
	if executionResult != nil && executionResult.RuleSetExecution != nil {
		options.RootNode = executionResult.RuleSetExecution.CanonicalDocument
	}
	return options
}
