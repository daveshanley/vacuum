package motor

import (
	"github.com/daveshanley/vacuum/repair"
	"sync"
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

func baselineRepairFindings(result *RuleSetExecutionResult) ([]repair.Finding, error) {
	c := &result.repairBaseline
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

func baselineRepairIntegrity(result *RuleSetExecutionResult, options *ExecutionOptions) ([]repairDefect, error) {
	c := &result.repairBaseline
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
