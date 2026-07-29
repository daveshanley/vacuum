// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

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

// NormalizeTraceValue validates a trace value and accepts the legacy
// "messages" alias used by older Vacuum clients.
func NormalizeTraceValue(value TraceValue) TraceValue {
	if value == "messages" {
		value = TraceValueMessage
	}
	switch value {
	case TraceValueOff, TraceValueMessage, TraceValueVerbose:
		return value
	default:
		return TraceValueOff
	}
}
