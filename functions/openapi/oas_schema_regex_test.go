// Copyright 2022-2026 Dave Shanley / Quobix
// Princess Beef Heavy Industries LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func runOASSchemaForRegexTest(t *testing.T, spec string) []model.RuleFunctionResult {
	t.Helper()
	specInfo, err := datamodel.ExtractSpecInfo([]byte(spec))
	require.NoError(t, err)

	rule := buildOpenApiTestRuleAction("$", "oas_schema", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, index.CreateOpenAPIIndexConfig())
	ctx.SpecInfo = specInfo
	ctx.Document, _ = libopenapi.NewDocument([]byte(spec))

	return OASSchema{}.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)
}

func oas3SpecWithPatterns(version string, patterns ...string) string {
	var props strings.Builder
	for i, p := range patterns {
		fmt.Fprintf(&props, "                  p%d:\n                    type: string\n                    pattern: '%s'\n", i, p)
	}
	return fmt.Sprintf(`openapi: %s
info:
  title: Regex pattern repro
  version: 1.0.0
paths:
  /durations:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
%s`, version, props.String())
}

// https://github.com/daveshanley/vacuum/issues/1068
func TestOASSchema_ECMAScriptPatterns_Accepted(t *testing.T) {
	patterns := []string{
		`^P(?=\d)(\d+Y)?(\d+M)?(\d+D)?$`, // lookahead
		`^[a-z]{1,2048}$`,                // repeat count above RE2's limit of 1000
		`^(?<!x)[a-z]+$`,                 // lookbehind
		`^(a)\1$`,                        // backreference
	}
	for _, version := range []string{"3.0.3", "3.1.0", "3.2.0"} {
		t.Run(version, func(t *testing.T) {
			res := runOASSchemaForRegexTest(t, oas3SpecWithPatterns(version, patterns...))
			assert.Empty(t, res)
		})
	}
}

func TestOASSchema_ECMAScriptPatterns_EachPatternAcceptedAlone(t *testing.T) {
	for _, p := range []string{`^P(?=\d)$`, `^[a-z]{1,2048}$`} {
		t.Run(p, func(t *testing.T) {
			assert.Empty(t, runOASSchemaForRegexTest(t, oas3SpecWithPatterns("3.1.0", p)))
		})
	}
}

// The 3.1 and 3.2 meta-schemas do not assert `format: regex`, so only 3.0 reports this.
func TestOASSchema_InvalidPattern_StillReported(t *testing.T) {
	res := runOASSchemaForRegexTest(t, oas3SpecWithPatterns("3.0.3", `^(unclosed`))
	require.NotEmpty(t, res)
	assert.Contains(t, res[0].Message, "expected `regex`")
}

func TestOASSchema_Swagger2_ECMAScriptPatterns_Accepted(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Regex pattern repro
  version: 1.0.0
paths:
  /durations:
    get:
      produces: [application/json]
      parameters:
        - name: duration
          in: query
          type: string
          pattern: '^P(?=\d)(\d+Y)?$'
      responses:
        '200':
          description: OK
`
	assert.Empty(t, runOASSchemaForRegexTest(t, spec))
}

// The engine also matches the patterns baked into the OpenAPI meta-schema, so
// they must keep behaving as they did under the Go engine.
func TestOASSchema_MetaSchemaPatternsStillMatch(t *testing.T) {
	spec := oas3SpecWithPatterns("3.1.0", `^[a-z]+$`) + `
x-extension: allowed
`
	assert.Empty(t, runOASSchemaForRegexTest(t, spec))

	bad := strings.Replace(spec, "x-extension", "extension", 1)
	assert.NotEmpty(t, runOASSchemaForRegexTest(t, bad))
}

func TestOASRegexEngine_ECMAScript(t *testing.T) {
	rx, err := oasRegexEngine(`^P(?=\d)\d+$`)
	require.NoError(t, err)
	assert.True(t, rx.MatchString("P12"))
	assert.False(t, rx.MatchString("Px"))
	assert.Equal(t, `^P(?=\d)\d+$`, rx.String())
}

func TestOASRegexEngine_FallsBackToGoSyntax(t *testing.T) {
	// valid RE2 but not ECMAScript: these were accepted before the engine
	// was swapped, so they must not start failing.
	for _, p := range []string{`(?i)^abc$`, `(?P<name>a)+`, `\pL+`} {
		rx, err := oasRegexEngine(p)
		require.NoError(t, err, p)
		assert.NotNil(t, rx)
	}
}

func TestOASRegexEngine_InvalidInBothEngines(t *testing.T) {
	_, err := oasRegexEngine(`^(unclosed`)
	assert.Error(t, err)
}

func TestOASRegexEngine_MatchErrorIsNoMatch(t *testing.T) {
	// catastrophic backtracking has to time out instead of hanging the lint run
	rx, err := oasRegexEngine(`^(a+)+$`)
	require.NoError(t, err)
	assert.False(t, rx.MatchString(strings.Repeat("a", 64)+"!"))
}
