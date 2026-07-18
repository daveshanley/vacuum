// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

import (
	"testing"

	"github.com/pb33f/testify/assert"
)

func TestNormalizeTraceValue(t *testing.T) {
	assert.Equal(t, TraceValueMessage, NormalizeTraceValue("messages"))
	assert.Equal(t, TraceValueMessage, NormalizeTraceValue(TraceValueMessage))
	assert.Equal(t, TraceValueVerbose, NormalizeTraceValue(TraceValueVerbose))
	assert.Equal(t, TraceValueOff, NormalizeTraceValue(TraceValueOff))
	assert.Equal(t, TraceValueOff, NormalizeTraceValue("invalid"))
}
