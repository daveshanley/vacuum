// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package eventloop provides an event loop implementation for running asynchronous
// JavaScript code with Goja. It enables async/await and Promise support in custom
// JavaScript functions by processing the microtask queue.
//
// The design is inspired by Grafana k6's event loop implementation and goja_nodejs.
package eventloop

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/daveshanley/vacuum/plugin/javascript/eventloop/fetch"
	"github.com/dop251/goja"
)

var (
	// ErrLoopAlreadyRunning is returned when Run is called on an already running loop
	ErrLoopAlreadyRunning = errors.New("event loop already running")

	// ErrLoopNotRunning is returned when trying to queue work on a stopped loop
	ErrLoopNotRunning = errors.New("event loop not running")

	// ErrPromiseRejected is returned when a Promise is rejected
	ErrPromiseRejected = errors.New("promise rejected")

	// ErrPromiseTimeout is returned when a Promise doesn't resolve within timeout
	ErrPromiseTimeout = errors.New("promise did not resolve within timeout")
)

// EventLoop manages asynchronous JavaScript execution by processing
// a job queue and handling Promise microtasks.
type EventLoop struct {
	vm *goja.Runtime

	// job queue for callbacks - using a slice protected by mutex for simplicity
	jobQueue     []func()
	jobQueueLock sync.Mutex
	jobCond      *sync.Cond

	// state tracking
	pendingOps int32 // atomic counter for pending async operations (timers, etc.)
	running    int32 // atomic flag indicating if a loop is running

	// registered timers
	timers      map[*Timer]struct{}
	timersLock  sync.Mutex
	nextTimerID int64

	// Stop signal
	stopChan chan struct{}
}

// New creates a new EventLoop wrapping the given Goja runtime.
// The runtime should not be used directly while the event loop is running.
func New(vm *goja.Runtime) *EventLoop {
	loop := &EventLoop{
		vm:       vm,
		timers:   make(map[*Timer]struct{}),
		stopChan: make(chan struct{}),
	}
	loop.jobCond = sync.NewCond(&loop.jobQueueLock)

	// register setTimeout and clearTimeout in the runtime
	loop.registerTimerFunctions()

	return loop
}

// Runtime returns the underlying Goja runtime.
// Should only be used when the loop is not running.
func (e *EventLoop) Runtime() *goja.Runtime {
	return e.vm
}

// Run executes the provided function and processes the event loop
// until all pending work completes or the context is cancelled.
//
// The function fn is called with the Goja runtime and should return
// the result of the JavaScript execution (which may be a Promise).
//
// Run blocks until:
//   - All pending jobs are processed, and no more work is queued
//   - The context is cancelled (timeout or explicit cancellation)
//   - An error occurs during execution
func (e *EventLoop) Run(ctx context.Context, fn func(*goja.Runtime) (goja.Value, error)) (goja.Value, error) {
	if !atomic.CompareAndSwapInt32(&e.running, 0, 1) {
		return nil, ErrLoopAlreadyRunning
	}
	defer func() {
		atomic.StoreInt32(&e.running, 0)
		e.cancelAllTimers()
	}()

	// reset state - clear any orphaned pendingOps from previous runs that
	// may have timed out while async operations were still in flight
	atomic.StoreInt32(&e.pendingOps, 0)
	e.stopChan = make(chan struct{})

	result, err := fn(e.vm)
	if err != nil {
		return nil, err
	}

	if loopErr := e.runLoop(ctx); loopErr != nil {
		if errors.Is(loopErr, context.DeadlineExceeded) || errors.Is(loopErr, context.Canceled) {
			return result, loopErr
		}
		return nil, loopErr
	}

	return result, nil
}

// runLoop processes jobs until there's nothing left to do or context is canceled
func (e *EventLoop) runLoop(ctx context.Context) error {
	// start a goroutine to broadcast on context cancellation
	// use a done channel to allow cleanup when runLoop exits (prevents goroutine leak)
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			e.jobCond.Broadcast()
		case <-done:
			// runLoop finished, exit goroutine
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		job := e.getNextJob()

		if job != nil {
			job()
			e.processGojaJobs()
			continue
		}

		// no jobs in queue - check if we should wait or exit
		e.jobQueueLock.Lock()
		pendingOps := atomic.LoadInt32(&e.pendingOps)

		e.timersLock.Lock()
		timerCount := len(e.timers)
		e.timersLock.Unlock()

		if pendingOps == 0 && timerCount == 0 && len(e.jobQueue) == 0 {
			e.jobQueueLock.Unlock()
			return nil
		}

		for len(e.jobQueue) == 0 && atomic.LoadInt32(&e.running) == 1 {
			select {
			case <-ctx.Done():
				e.jobQueueLock.Unlock()
				return ctx.Err()
			default:
			}
			e.jobCond.Wait()
		}
		e.jobQueueLock.Unlock()
	}
}

// getNextJob retrieves and removes the next job from the queue
func (e *EventLoop) getNextJob() func() {
	e.jobQueueLock.Lock()
	defer e.jobQueueLock.Unlock()

	if len(e.jobQueue) == 0 {
		return nil
	}

	job := e.jobQueue[0]
	e.jobQueue[0] = nil // prevent memory leak
	e.jobQueue = e.jobQueue[1:]

	// reclaim memory when queue is empty and has grown large
	if len(e.jobQueue) == 0 && cap(e.jobQueue) > 64 {
		e.jobQueue = nil
	}

	return job
}

// processGojaJobs runs a no-op script to trigger Goja to process its internal
// job queue. This is necessary for Promise reactions (resolve/reject handlers)
// to be executed after callbacks that modify the Promise state.
func (e *EventLoop) processGojaJobs() {
	// running an empty script triggers Goja to process any pending jobs
	// in its internal queue (Promise reactions, etc.)
	_, _ = e.vm.RunString("")
}

// EnqueueJob adds a job to the queue.
// Thread-safe: can be called from any goroutine.
func (e *EventLoop) EnqueueJob(fn func()) {
	if atomic.LoadInt32(&e.running) == 0 {
		return
	}

	e.jobQueueLock.Lock()
	e.jobQueue = append(e.jobQueue, fn)
	e.jobQueueLock.Unlock()
	e.jobCond.Signal()
}

// RunOnLoop queues a function to be executed on the event loop.
// This is thread-safe and can be called from any goroutine.
// Returns false if the loop is not running.
func (e *EventLoop) RunOnLoop(fn func()) bool {
	if atomic.LoadInt32(&e.running) == 0 {
		return false
	}

	e.EnqueueJob(fn)
	return true
}

// RegisterCallback returns a function that will queue work on the event loop.
// This is the k6-style pattern for async operations.
//
// When called, it increments the pending operation counter to prevent the loop
// from exiting. The returned function should be called with the callback
// to execute when the async operation completes.
func (e *EventLoop) RegisterCallback() func(func()) {
	atomic.AddInt32(&e.pendingOps, 1)

	called := int32(0) // Ensure callback is only used once

	return func(callback func()) {
		if !atomic.CompareAndSwapInt32(&called, 0, 1) {
			return
		}

		if atomic.LoadInt32(&e.running) == 0 {
			// loop has stopped - just decrement pendingOps, don't try to enqueue
			// This prevents stale pendingOps from blocking the next Run()
			atomic.AddInt32(&e.pendingOps, -1)
			return
		}

		e.EnqueueJob(func() {
			callback()
			atomic.AddInt32(&e.pendingOps, -1)
		})
	}
}

// cancelAllTimers cancels all pending timers
func (e *EventLoop) cancelAllTimers() {
	e.timersLock.Lock()
	defer e.timersLock.Unlock()

	for timer := range e.timers {
		timer.cancel()
	}
	e.timers = make(map[*Timer]struct{})
}

// RegisterFetch registers the fetch() function in the JavaScript runtime.
// If config is nil, secure defaults are used (HTTPS only, no private networks).
func (e *EventLoop) RegisterFetch(config *fetch.FetchConfig) {
	module := fetch.NewFetchModule(e, config)
	module.Register()
}
