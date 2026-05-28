// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want int
		ok   bool
	}{
		{a: "v0.27.1", b: "0.27.0", want: 1, ok: true},
		{a: "0.27.0", b: "v0.27.0", want: 0, ok: true},
		{a: "0.26.9", b: "0.27.0", want: -1, ok: true},
		{a: "0.27.0-beta.1", b: "0.26.9", ok: false},
		{a: "unknown", b: "0.26.9", ok: false},
	}

	for _, tc := range tests {
		got, ok := CompareVersions(tc.a, tc.b)
		if ok != tc.ok {
			t.Fatalf("CompareVersions(%q, %q) ok = %v, want %v", tc.a, tc.b, ok, tc.ok)
		}
		if ok && got != tc.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
