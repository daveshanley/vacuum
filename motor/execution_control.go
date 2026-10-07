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

// Keep rule execution independent from GOMAXPROCS without flooding shared
// spec/index lookup state with hundreds of concurrent rules.
const defaultMaxRuleConcurrency = 32

// executionControl owns cancellation, worker limits, and auto-fix serialization
// for one invocation. Close ends its context without cancelling the caller.
type executionControl struct {
	context            context.Context
	cancel             context.CancelFunc
	maxRuleConcurrency int

	autoFixGate chan struct{}
}

// newExecutionControl validates options and derives an invocation-owned context.
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
		ctx, cancel = context.WithCancel(parent)
	}

	maxConcurrency := options.MaxRuleConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = defaultMaxRuleConcurrency
	}
	control := &executionControl{
		context:            ctx,
		cancel:             cancel,
		maxRuleConcurrency: maxConcurrency,

		autoFixGate: make(chan struct{}, 1),
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

func (c *executionControl) AutoFixGate() chan struct{} {
	if c == nil {
		return nil
	}
	return c.autoFixGate
}

func (c *executionControl) Close() {
	if c != nil && c.cancel != nil {
		c.cancel()
	}
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
	result.Errors = appendContextErrorToErrors(result.Errors, control.Err())
	return result
}

// ruleRunGuard makes auto-fix publication and scheduler detachment mutually exclusive.
// The commit calls beginSharedWork before touching caller-owned nodes. On timeout
// or cancellation, abandon returns true if publication has started; the scheduler
// must then wait for doneChan instead of returning while those nodes are changing.
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
