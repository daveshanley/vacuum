//go:build js && wasm

// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import "fmt"

// GenerateChangeReportWithTree is unavailable in browser builds because the
// change renderer depends on native terminal and database packages.
func GenerateChangeReportWithTree(originalSpecPath string, newSpecBytes []byte, newSpecFilePath string) (*ChangeResult, error) {
	return nil, fmt.Errorf("API change generation is unavailable in browser builds")
}
