// Copyright 2022-2026 Dave Shanley / Quobix
// Princess Beef Heavy Industries LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"regexp"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/pb33f/jsonschema/v6"
	"github.com/pb33f/libopenapi-validator/config"
)

// oasValidationOptions are applied wherever the OpenAPI meta-schema is compiled.
var oasValidationOptions = []config.Option{config.WithRegexEngine(oasRegexEngine)}

// oasRegexTimeout bounds a single match so a pathological pattern in the
// OpenAPI meta-schema or a document cannot stall the lint run.
const oasRegexTimeout = time.Second

// oasECMARegexp adapts regexp2 to the jsonschema.Regexp interface.
type oasECMARegexp struct{ *regexp2.Regexp }

// MatchString reports whether the pattern matches. A match error (timeout) counts as no match.
func (r oasECMARegexp) MatchString(value string) bool {
	matched, err := r.Regexp.MatchString(value)
	return err == nil && matched
}

// oasRegexEngine compiles patterns for the OpenAPI meta-schema. JSON Schema
// defines `pattern` and `format: regex` in terms of ECMA-262, so patterns are
// compiled in ECMAScript mode first (lookaheads, lookbehinds, backreferences
// and repeat counts above 1000 are valid there but not in Go's RE2). A pattern
// that only the Go engine accepts, such as `(?i)` or `(?P<name>...)`, was
// accepted before this engine existed, so it is still accepted.
func oasRegexEngine(pattern string) (jsonschema.Regexp, error) {
	rx, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err == nil {
		rx.MatchTimeout = oasRegexTimeout
		return oasECMARegexp{rx}, nil
	}
	if native, nativeErr := regexp.Compile(pattern); nativeErr == nil {
		return native, nil
	}
	return nil, err
}
