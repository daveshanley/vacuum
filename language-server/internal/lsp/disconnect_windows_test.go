// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

//go:build windows

package lsp

import (
	"fmt"
	"syscall"
	"testing"

	"github.com/pb33f/testify/assert"
)

func TestWindowsPipeErrorsAreDisconnects(t *testing.T) {
	assert.True(t, isDisconnectError(fmt.Errorf("write header: %w", syscall.ERROR_BROKEN_PIPE)))
	assert.True(t, isDisconnectError(fmt.Errorf("write payload: %w", windowsErrorNoData)))
}
