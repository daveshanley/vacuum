// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package model

// Contract selects the rule semantics for one execution. The zero value is native vacuum.
type Contract string

const (
	// NativeContract preserves vacuum rule semantics.
	NativeContract Contract = ""
	// OpenLintV1Draft pins the OpenLint v1 draft used by this implementation.
	OpenLintV1Draft Contract = "openlint-v1@4e95c96e4a7ecb17320cd11fc2fcd2571e8d0291"
)
