// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

//go:build windows

package lsp

import (
	"errors"
	"syscall"
)

// ERROR_NO_DATA is not exported by syscall.
const windowsErrorNoData = syscall.Errno(232)

func isPlatformDisconnectError(err error) bool {
	return errors.Is(err, syscall.ERROR_BROKEN_PIPE) ||
		errors.Is(err, windowsErrorNoData)
}
