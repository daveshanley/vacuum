// Copyright 2022-2026 Dave Shanley / Quobix
// Princess Beef Heavy Industries LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	r2 "github.com/dlclark/regexp2/v2"
	"github.com/dop251/goja"
	"github.com/pb33f/jsonschema/v6"
	"github.com/pb33f/libopenapi-validator/config"
)

// oasValidationOptions are applied wherever the OpenAPI meta-schema is compiled.
var oasValidationOptions = []config.Option{config.WithRegexEngine(oasRegexEngine)}

// oasRegexTimeout bounds a single match so a pathological pattern in the
// OpenAPI meta-schema or a document cannot stall the lint run.
const oasRegexTimeout = time.Second

// oasECMARegexp adapts regexp2 to the jsonschema.Regexp interface.
type oasECMARegexp struct{ *r2.Regexp }

// MatchString reports whether the pattern matches. A match error (timeout) counts as no match.
func (r oasECMARegexp) MatchString(value string) bool {
	matched, err := r.Regexp.MatchString(value)
	return err == nil && matched
}

// oasRegexEngine compiles the patterns used while validating an OpenAPI
// document against the meta-schema.
//
// Anything Go's RE2 engine accepts keeps the fast native matcher. That covers
// every pattern baked into the OpenAPI meta-schema itself (like `^/` for path
// keys), so built-in patterns are never sent through regexp2, and Go-only
// syntax that has always been accepted, like `(?i)` or `(?P<name>...)`, keeps
// working unchanged.
//
// Only patterns RE2 rejects reach the ECMAScript path. Those are the
// document-supplied patterns that matter: JSON Schema defines `pattern` and
// `format: regex` in terms of ECMA-262, where lookaheads, lookbehinds,
// backreferences and repeat counts above 1000 are valid. They are first
// checked with a real JavaScript engine (goja), so acceptance matches
// ECMA-262 exactly - constructs JavaScript rejects, like the atomic group
// `(?>...)`, stay invalid - and then compiled with regexp2 in ECMAScript
// mode for matching.
func oasRegexEngine(pattern string) (jsonschema.Regexp, error) {
	if native, err := regexp.Compile(pattern); err == nil {
		return native, nil
	}
	if err := checkECMAScriptPattern(pattern); err != nil {
		return nil, err
	}
	rx, err := r2.Compile(pattern, r2.ECMAScript)
	if err != nil {
		return nil, err
	}
	rx.MatchTimeout = oasRegexTimeout
	return oasECMARegexp{rx}, nil
}

// checkECMAScriptPattern reports an error unless pattern is a valid ECMA-262
// regular expression, using goja (an ECMAScript engine) as the reference
// implementation instead of relying on what a regex library happens to allow.
func checkECMAScriptPattern(pattern string) error {
	quoted, err := json.Marshal(pattern)
	if err != nil {
		return err
	}
	vm := goja.New()
	if _, err := vm.RunString("new RegExp(" + string(quoted) + ")"); err != nil {
		return fmt.Errorf("invalid ECMAScript pattern: %w", err)
	}
	return nil
}
