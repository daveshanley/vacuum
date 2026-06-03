// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package model

import "testing"

func TestFormatMatches_AsyncAPI3Family(t *testing.T) {
	if !FormatMatches(AsyncAPI3, AsyncAPI30) {
		t.Fatal("asyncapi3 should match asyncapi3_0")
	}
	if !FormatMatches(AsyncAPI3, AsyncAPI31) {
		t.Fatal("asyncapi3 should match asyncapi3_1")
	}
	if FormatMatches(AsyncAPI30, AsyncAPI3) {
		t.Fatal("asyncapi3_0 should not match family-detected asyncapi3")
	}
	if FormatMatches(AsyncAPI30, AsyncAPI31) {
		t.Fatal("asyncapi3_0 should not match asyncapi3_1")
	}
}
