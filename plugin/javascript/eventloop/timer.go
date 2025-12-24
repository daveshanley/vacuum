// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package eventloop

import (
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
)

// Timer represents a scheduled timeout or interval
type Timer struct {
	id        int64
	callback  goja.Callable
	duration  time.Duration
	interval  bool
	cancelled int32 // atomic
	timer     *time.Timer
	loop      *EventLoop
}

// cancel marks the timer as cancelled and stops the underlying timer
func (t *Timer) cancel() {
	if atomic.CompareAndSwapInt32(&t.cancelled, 0, 1) {
		if t.timer != nil {
			t.timer.Stop()
		}
	}
}

// isCancelled returns true if the timer has been canceled
func (t *Timer) isCancelled() bool {
	return atomic.LoadInt32(&t.cancelled) == 1
}

// registerTimerFunctions registers setTimeout and clearTimeout in the runtime
func (e *EventLoop) registerTimerFunctions() {
	e.vm.Set("setTimeout", e.setTimeout)
	e.vm.Set("clearTimeout", e.cancelTimer)
	e.vm.Set("setInterval", e.setInterval)
	e.vm.Set("clearInterval", e.cancelTimer)
}

// setTimeout implements the JavaScript setTimeout function
func (e *EventLoop) setTimeout(call goja.FunctionCall) goja.Value {
	return e.scheduleTimer(call, false)
}

// setInterval implements the JavaScript setInterval function
func (e *EventLoop) setInterval(call goja.FunctionCall) goja.Value {
	return e.scheduleTimer(call, true)
}

// scheduleTimer creates and schedules a timer (used by both setTimeout and setInterval)
func (e *EventLoop) scheduleTimer(call goja.FunctionCall, interval bool) goja.Value {
	if len(call.Arguments) < 1 {
		panic(e.vm.NewTypeError("setTimeout requires at least 1 argument"))
	}

	callback, ok := goja.AssertFunction(call.Argument(0))
	if !ok {
		panic(e.vm.NewTypeError("first argument must be a function"))
	}

	var delay int64 = 0
	if len(call.Arguments) > 1 {
		delay = call.Argument(1).ToInteger()
	}
	if delay < 0 {
		delay = 0
	}

	// Create and register timer in single lock acquisition
	e.timersLock.Lock()
	e.nextTimerID++
	timerID := e.nextTimerID
	timer := &Timer{
		id:       timerID,
		callback: callback,
		duration: time.Duration(delay) * time.Millisecond,
		interval: interval,
		loop:     e,
	}
	e.timers[timer] = struct{}{}
	e.timersLock.Unlock()

	timer.timer = time.AfterFunc(timer.duration, func() {
		e.timerFired(timer)
	})

	return e.vm.ToValue(timerID)
}

// timerFired is called when a timer fires
func (e *EventLoop) timerFired(timer *Timer) {
	if timer.isCancelled() {
		return
	}

	// Queue the callback to run on the event loop
	e.EnqueueJob(func() {
		if timer.isCancelled() {
			return
		}

		// Call the callback
		_, _ = timer.callback(goja.Undefined())

		if timer.interval && !timer.isCancelled() {
			// Reschedule for an interval
			timer.timer = time.AfterFunc(timer.duration, func() {
				e.timerFired(timer)
			})
		} else {
			// Remove from active timers
			e.timersLock.Lock()
			delete(e.timers, timer)
			e.timersLock.Unlock()
		}
	})
}

// cancelTimer cancels a timer by ID (used for both clearTimeout and clearInterval)
func (e *EventLoop) cancelTimer(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return goja.Undefined()
	}

	timerID := call.Argument(0).ToInteger()

	e.timersLock.Lock()
	defer e.timersLock.Unlock()

	for timer := range e.timers {
		if timer.id == timerID {
			timer.cancel()
			delete(e.timers, timer)
			break
		}
	}

	return goja.Undefined()
}

// SetTimeout schedules a Go function to be called after the specified delay.
// Returns a Timer that can be used with ClearTimeout.
func (e *EventLoop) SetTimeout(fn func(), delay time.Duration) *Timer {
	// Create and register timer in single lock acquisition
	e.timersLock.Lock()
	e.nextTimerID++
	timerID := e.nextTimerID
	timer := &Timer{
		id:       timerID,
		duration: delay,
		interval: false,
		loop:     e,
	}
	e.timers[timer] = struct{}{}
	e.timersLock.Unlock()

	timer.timer = time.AfterFunc(delay, func() {
		if timer.isCancelled() {
			return
		}

		e.EnqueueJob(func() {
			if !timer.isCancelled() {
				fn()
			}
			e.timersLock.Lock()
			delete(e.timers, timer)
			e.timersLock.Unlock()
		})
	})

	return timer
}

// ClearTimeout cancels a pending timeout created with SetTimeout
func (e *EventLoop) ClearTimeout(t *Timer) {
	if t == nil {
		return
	}

	t.cancel()

	e.timersLock.Lock()
	delete(e.timers, t)
	e.timersLock.Unlock()
}
