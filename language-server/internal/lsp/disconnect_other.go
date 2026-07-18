// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

//go:build !windows

package lsp

func isPlatformDisconnectError(_ error) bool {
	return false
}
