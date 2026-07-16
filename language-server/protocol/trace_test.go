// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

import (
	"testing"

	"github.com/pb33f/testify/assert"
)

func TestTraceValuesAndLegacyMessagesAlias(t *testing.T) {
	t.Cleanup(func() {
		SetTraceValue(TraceValueOff)
	})

	SetTraceValue("messages")
	assert.Equal(t, TraceValueMessage, GetTraceValue())
	assert.True(t, HasTraceLevel(TraceValueMessage))
	assert.False(t, HasTraceLevel(TraceValueVerbose))

	SetTraceValue(TraceValueVerbose)
	assert.True(t, HasTraceLevel(TraceValueMessage))
	assert.True(t, HasTraceLevel(TraceValueVerbose))

	SetTraceValue("invalid")
	assert.Equal(t, TraceValueOff, GetTraceValue())
	assert.False(t, HasTraceLevel(TraceValueMessage))
}
