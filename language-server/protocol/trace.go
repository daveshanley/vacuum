// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

import (
	"fmt"
	"sync"
)

const (
	// MethodSetTrace changes server trace verbosity.
	MethodSetTrace = Method("$/setTrace")
	// MethodCancelRequest cancels an active JSON-RPC request.
	MethodCancelRequest = Method("$/cancelRequest")
)

// CancelParams identifies the request to cancel.
type CancelParams struct {
	ID ID `json:"id"`
}

// TraceValue identifies LSP trace verbosity.
type TraceValue string

const (
	// TraceValueOff disables protocol tracing.
	TraceValueOff TraceValue = "off"
	// TraceValueMessage enables message-level tracing.
	TraceValueMessage TraceValue = "message"
	// TraceValueVerbose enables verbose tracing.
	TraceValueVerbose TraceValue = "verbose"
)

// SetTraceParams contains a requested trace value.
type SetTraceParams struct {
	Value TraceValue `json:"value"`
}

var traceState = struct {
	sync.RWMutex
	value TraceValue
}{value: TraceValueOff}

// GetTraceValue returns the current process-wide trace value.
func GetTraceValue() TraceValue {
	traceState.RLock()
	defer traceState.RUnlock()
	return traceState.value
}

// SetTraceValue sets the current trace value, accepting the legacy "messages" alias.
func SetTraceValue(value TraceValue) {
	if value == "messages" {
		value = TraceValueMessage
	}
	switch value {
	case TraceValueOff, TraceValueMessage, TraceValueVerbose:
	default:
		value = TraceValueOff
	}
	traceState.Lock()
	traceState.value = value
	traceState.Unlock()
}

// HasTraceLevel reports whether a trace level is currently enabled.
func HasTraceLevel(value TraceValue) bool {
	current := GetTraceValue()
	switch current {
	case TraceValueOff:
		return false
	case TraceValueMessage:
		return value == TraceValueMessage
	case TraceValueVerbose:
		return true
	default:
		panic(fmt.Sprintf("unsupported trace level: %s", current))
	}
}
