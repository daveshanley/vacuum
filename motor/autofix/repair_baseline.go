// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"sync"

	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/repair"
)

// A baseline stays immutable for the lifetime of its review. Cache only successful
// checks; a canceled lookup must not poison a later preview.
type repairBaselineCache struct {
	mu             sync.Mutex
	findings       []repair.Finding
	findingsReady  bool
	defects        []repairDefect
	integrityReady bool
}

func baselineRepairFindings(result *motor.RuleSetExecutionResult, cache *repairBaselineCache) ([]repair.Finding, error) {
	c := repairCache(cache)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.findingsReady {
		return c.findings, nil
	}
	findings, err := repairFindings(result.RuleSetExecution, result.Results)
	if err == nil {
		c.findings = findings
		c.findingsReady = true
	}
	return findings, err
}

func baselineRepairIntegrity(result *motor.RuleSetExecutionResult, options *motor.ExecutionOptions, cache *repairBaselineCache) ([]repairDefect, error) {
	c := repairCache(cache)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.integrityReady {
		return c.defects, nil
	}
	defects, err := repairIntegrityDefects(result, options)
	if err == nil {
		c.defects = defects
		c.integrityReady = true
	}
	return defects, err
}

// repairCache returns invocation-owned storage when no review cache is supplied.
func repairCache(cache *repairBaselineCache) *repairBaselineCache {
	if cache != nil {
		return cache
	}
	return &repairBaselineCache{}
}
