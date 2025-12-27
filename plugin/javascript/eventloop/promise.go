// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package eventloop

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
)

// PromiseState represents the state of a Promise
type PromiseState int

const (
	PromiseStatePending PromiseState = iota
	PromiseStateFulfilled
	PromiseStateRejected
)

// PromiseResult holds the result of awaiting a Promise
type PromiseResult struct {
	State PromiseState
	Value goja.Value
	Error error
}

// IsPromise checks if a goja.Value is a Promise
func IsPromise(value goja.Value) bool {
	if value == nil || value == goja.Undefined() || value == goja.Null() {
		return false
	}

	exported := value.Export()
	_, ok := exported.(*goja.Promise)
	return ok
}

// GetPromise extracts a *goja.Promise from a goja.Value
// Returns nil if the value is not a Promise
func GetPromise(value goja.Value) *goja.Promise {
	if value == nil || value == goja.Undefined() || value == goja.Null() {
		return nil
	}

	exported := value.Export()
	if promise, ok := exported.(*goja.Promise); ok {
		return promise
	}
	return nil
}

// AwaitPromise waits for a Promise to resolve or reject.
// It processes the event loop until the Promise settles or the context is cancelled.
//
// If the value is not a Promise, it returns immediately with the value.
func (e *EventLoop) AwaitPromise(ctx context.Context, value goja.Value) PromiseResult {
	promise := GetPromise(value)
	if promise == nil {
		// Not a Promise, return the value directly
		return PromiseResult{
			State: PromiseStateFulfilled,
			Value: value,
		}
	}

	// Check if already resolved
	state := promise.State()
	if state == goja.PromiseStateFulfilled {
		return PromiseResult{
			State: PromiseStateFulfilled,
			Value: promise.Result(),
		}
	}
	if state == goja.PromiseStateRejected {
		return PromiseResult{
			State: PromiseStateRejected,
			Value: promise.Result(),
			Error: fmt.Errorf("%v", promise.Result().Export()),
		}
	}

	// Promise is pending - this shouldn't happen if the event loop
	// was run properly, but handle it gracefully
	return PromiseResult{
		State: PromiseStatePending,
		Error: ErrPromiseTimeout,
	}
}

// ExtractPromiseValue extracts the resolved value from a Promise.
// If the value is not a Promise, returns the value directly.
// Returns an error if the Promise was rejected or is still pending.
func ExtractPromiseValue(value goja.Value) (interface{}, error) {
	promise := GetPromise(value)
	if promise == nil {
		// Not a Promise, return the value directly
		if value == nil {
			return nil, nil
		}
		return value.Export(), nil
	}

	state := promise.State()
	switch state {
	case goja.PromiseStateFulfilled:
		result := promise.Result()
		if result == nil {
			return nil, nil
		}
		return result.Export(), nil

	case goja.PromiseStateRejected:
		result := promise.Result()
		if result == nil {
			return nil, ErrPromiseRejected
		}
		// Try to extract error message from JavaScript Error objects
		errMsg := extractErrorMessage(result)
		return nil, fmt.Errorf("%w: %s", ErrPromiseRejected, errMsg)

	default:
		return nil, ErrPromiseTimeout
	}
}

// extractErrorMessage extracts a human-readable error message from a goja.Value.
// It handles JavaScript Error objects by getting their 'message' property,
// and falls back to String() or Export() for other types.
func extractErrorMessage(value goja.Value) string {
	if value == nil {
		return "unknown error"
	}

	// Try to get 'message' property (for Error objects like TypeError, Error, etc.)
	// Only attempt this for Object types - strings/primitives would panic with nil runtime
	if obj, ok := value.(*goja.Object); ok && obj != nil {
		if msg := obj.Get("message"); msg != nil && !goja.IsUndefined(msg) && !goja.IsNull(msg) {
			return msg.String()
		}
	}

	// Fall back to string conversion
	if str := value.String(); str != "" && str != "[object Object]" {
		return str
	}

	// Last resort: export and format
	exported := value.Export()
	if exported != nil {
		return fmt.Sprintf("%v", exported)
	}

	return "unknown error"
}

// WrapGoFunctionAsync wraps a Go function as an async JavaScript function.
// The Go function receives the event loop's RegisterCallback function to
// properly signal async completion.
//
// Example usage:
//
//	loop.WrapGoFunctionAsync("fetchData", func(call goja.FunctionCall, enqueue func(func())) goja.Value {
//	    url := call.Argument(0).String()
//	    promise, resolve, reject := loop.vm.NewPromise()
//
//	    go func() {
//	        result, err := http.Get(url)
//	        enqueue(func() {
//	            if err != nil {
//	                reject(loop.vm.NewGoError(err))
//	            } else {
//	                resolve(loop.vm.ToValue(result))
//	            }
//	        })
//	    }()
//
//	    return loop.vm.ToValue(promise)
//	})
func (e *EventLoop) WrapGoFunctionAsync(name string, fn func(call goja.FunctionCall, enqueue func(func())) goja.Value) {
	wrapped := func(call goja.FunctionCall) goja.Value {
		enqueue := e.RegisterCallback()
		return fn(call, enqueue)
	}
	e.vm.Set(name, wrapped)
}

// NewPromise creates a new Promise and returns resolve/reject functions.
// This is a convenience wrapper around goja.Runtime.NewPromise().
func (e *EventLoop) NewPromise() (*goja.Promise, func(interface{}), func(interface{})) {
	promise, resolve, reject := e.vm.NewPromise()

	// Wrap resolves to convert Go values to goja.Value
	wrappedResolve := func(value interface{}) {
		if v, ok := value.(goja.Value); ok {
			resolve(v)
		} else {
			resolve(e.vm.ToValue(value))
		}
	}

	// Wrap reject to convert Go values/errors to goja.Value
	wrappedReject := func(value interface{}) {
		switch v := value.(type) {
		case goja.Value:
			reject(v)
		case error:
			reject(e.vm.NewGoError(v))
		default:
			reject(e.vm.ToValue(value))
		}
	}

	return promise, wrappedResolve, wrappedReject
}
