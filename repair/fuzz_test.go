// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"bytes"
	"encoding/json"
	"strconv"
	"testing"
)

func FuzzPointerTokenRoundTrip(f *testing.F) {
	for _, seed := range []string{"", "a/b", "~01", "pets🐈", "a~b/c"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, token string) {
		decoded, err := tokens("/" + EscapeToken(token))
		if err != nil || len(decoded) != 1 || decoded[0] != token {
			t.Fatalf("pointer token did not round trip: %q", token)
		}
	})
}

func FuzzPatchPreservesInputAndUnrelatedNumber(f *testing.F) {
	f.Add(byte(0), byte(1), "text")
	f.Add(byte(4), byte(2), "🐈")
	f.Fuzz(func(t *testing.T, op, index byte, value string) {
		if len(value) > 1024 {
			t.Skip()
		}
		source := []byte(`{"items":["a","b","c"],"untouched":9007199254740993.0}`)
		original := append([]byte(nil), source...)
		encoded, _ := json.Marshal(value)
		plan := NewPlan("s", source)
		names := []string{"add", "remove", "replace", "copy", "move", "test"}
		plan.Operations = []Operation{{Op: names[int(op)%len(names)], Path: "/items/" + strconv.Itoa(int(index)%5), From: "/items/0", Value: encoded}}
		result, err := Apply("s", source, plan)
		if !bytes.Equal(source, original) {
			t.Fatal("patch mutated input bytes")
		}
		if err != nil {
			if result != nil {
				t.Fatal("failed transaction published bytes")
			}
			return
		}
		if !json.Valid(result) || !bytes.Contains(result, []byte("9007199254740993.0")) {
			t.Fatal("patch changed unrelated exact number or produced invalid JSON")
		}
	})
}

func FuzzTypedEqualitySymmetry(f *testing.F) {
	f.Add(`1`, `1.0`)
	f.Add(`"1"`, `1`)
	f.Add(`{"a":[null,true]}`, `{"a":[null,true]}`)
	f.Fuzz(func(t *testing.T, a, b string) {
		if len(a) > 1024 || len(b) > 1024 {
			t.Skip()
		}
		left, err := valueNode(json.RawMessage(a))
		if err != nil {
			return
		}
		right, err := valueNode(json.RawMessage(b))
		if err != nil {
			return
		}
		ab, errAB := equalNodes(left, right)
		ba, errBA := equalNodes(right, left)
		if (errAB == nil) != (errBA == nil) || ab != ba {
			t.Fatal("typed equality is not symmetric")
		}
	})
}
