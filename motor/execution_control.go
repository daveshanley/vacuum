// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

const defaultMaxRuleConcurrency = 32

type executionControl struct {
	context            context.Context
	cancel             context.CancelFunc
	maxRuleConcurrency int
	ruleRunsMu         sync.Mutex
	activeRuleRuns     int
	ruleRunsSealed     bool
	ruleRunsDoneCh     chan struct{}
	autoFixGate        chan struct{}
}

func newExecutionControl(options *ExecutionOptions) (*executionControl, error) {
	if options == nil {
		options = &ExecutionOptions{}
	}
	if options.MaxRuleConcurrency < 0 {
		return nil, fmt.Errorf("max rule concurrency must be zero or greater, got %d", options.MaxRuleConcurrency)
	}

	parent := options.Context
	if parent == nil {
		parent = context.Background()
	}

	var ctx context.Context
	var cancel context.CancelFunc
	if options.RunTimeout > 0 {
		ctx, cancel = context.WithTimeout(parent, options.RunTimeout)
	} else {
		ctx = parent
		cancel = func() {}
	}

	maxConcurrency := options.MaxRuleConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = defaultMaxRuleConcurrency
	}
	control := &executionControl{
		context:            ctx,
		cancel:             cancel,
		maxRuleConcurrency: maxConcurrency,
		ruleRunsDoneCh:     make(chan struct{}),
		autoFixGate:        make(chan struct{}, 1),
	}
	control.autoFixGate <- struct{}{}
	return control, nil
}

func (c *executionControl) Context() context.Context {
	if c == nil || c.context == nil {
		return context.Background()
	}
	return c.context
}

func (c *executionControl) Done() <-chan struct{} {
	return c.Context().Done()
}

func (c *executionControl) Err() error {
	if c == nil {
		return nil
	}
	return c.Context().Err()
}

func (c *executionControl) MaxRuleConcurrency() int {
	if c == nil || c.maxRuleConcurrency <= 0 {
		return defaultMaxRuleConcurrency
	}
	return c.maxRuleConcurrency
}

func (c *executionControl) Close() {
	if c != nil && c.cancel != nil {
		c.cancel()
	}
}

func (c *executionControl) startRuleRun() {
	if c == nil {
		return
	}
	c.ruleRunsMu.Lock()
	defer c.ruleRunsMu.Unlock()
	if c.ruleRunsSealed {
		panic("motor: rule run started after execution was sealed")
	}
	c.activeRuleRuns++
}

func (c *executionControl) finishRuleRun() {
	if c == nil {
		return
	}
	c.ruleRunsMu.Lock()
	defer c.ruleRunsMu.Unlock()
	c.activeRuleRuns--
	if c.ruleRunsSealed && c.activeRuleRuns == 0 {
		close(c.ruleRunsDoneCh)
	}
}

func (c *executionControl) ruleRunsDone() <-chan struct{} {
	if c == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	c.ruleRunsMu.Lock()
	defer c.ruleRunsMu.Unlock()
	if !c.ruleRunsSealed {
		c.ruleRunsSealed = true
		if c.activeRuleRuns == 0 {
			close(c.ruleRunsDoneCh)
		}
	}
	return c.ruleRunsDoneCh
}

func appendContextError(result *RuleSetExecutionResult, contextErr error) *RuleSetExecutionResult {
	if result == nil || contextErr == nil {
		return result
	}
	result.Errors = appendContextErrorToErrors(result.Errors, contextErr)
	return result
}

func appendContextErrorToErrors(errs []error, contextErr error) []error {
	if contextErr == nil {
		return errs
	}
	for _, existing := range errs {
		if errors.Is(existing, contextErr) {
			return errs
		}
	}
	return append(errs, contextErr)
}

func executionErrorResult(
	execution *RuleSetExecution,
	control *executionControl,
	errs ...error,
) *RuleSetExecutionResult {
	result := &RuleSetExecutionResult{
		RuleSetExecution: execution,
		Errors:           errs,
	}
	return appendContextError(result, control.Err())
}

type ruleRunGuard struct {
	mu         sync.Mutex
	abandoned  bool
	sharedWork bool
}

func (g *ruleRunGuard) beginSharedWork() bool {
	if g == nil {
		return true
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.abandoned {
		return false
	}
	g.sharedWork = true
	return true
}

func (g *ruleRunGuard) abandon() (sharedWorkStarted bool) {
	if g == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.sharedWork {
		return true
	}
	g.abandoned = true
	return false
}
