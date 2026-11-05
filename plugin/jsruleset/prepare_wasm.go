//go:build js && wasm

// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package jsruleset

import (
	"context"
	"fmt"
)

// PrepareWithReader rejects local JavaScript module compilation in browser builds.
// Browser builds use the built-in rules and have no local module filesystem.
func PrepareWithReader(ctx context.Context, entry string, readFile func(string) ([]byte, error)) (*Prepared, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("JavaScript ruleset module compilation is unavailable in browser builds: %s", entry)
}
