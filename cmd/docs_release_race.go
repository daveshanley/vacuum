//go:build race

// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import "github.com/daveshanley/vacuum/motor"

func releaseDocsLintResources(execution *motor.RuleSetExecutionResult) {
	// libopenapi currently closes an index completion channel just after it
	// signals completion; releasing that index immediately trips -race even
	// though diagnostics has already copied the lint data it needs.
}
