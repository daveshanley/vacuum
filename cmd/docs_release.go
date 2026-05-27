//go:build !race

// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import "github.com/daveshanley/vacuum/motor"

func releaseDocsLintResources(execution *motor.RuleSetExecutionResult) {
	if execution != nil {
		execution.ReleaseOwnedResources()
	}
}
