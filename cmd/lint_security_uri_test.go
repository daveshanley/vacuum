// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	vacuumreport "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/testify/require"
)

func TestLintSecurityURI_Issue1044(t *testing.T) {
	const spec = `openapi: 3.2.0
info:
  title: Security URI example
  description: Minimal local security URI example.
  version: '1'
paths:
  /items:
    get:
      operationId: listItems
      security:
        - '#/components/securitySchemes/ApiKey': []
      responses:
        '204':
          description: Success
components:
  securitySchemes:
    ApiKey:
      type: apiKey
      in: header
      name: X-API-Key
`
	for _, tc := range []struct {
		name, replacement string
		invalid           bool
	}{
		{"local URI", "'#/components/securitySchemes/ApiKey': []", false},
		{"named scheme", "ApiKey: []", false},
		{"missing URI", "'#/components/securitySchemes/Missing': []", true},
		{"second missing scheme", "ApiKey: []\n          Missing: []", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := filepath.Join(t.TempDir(), "security.yaml")
			contents := strings.Replace(spec, "'#/components/securitySchemes/ApiKey': []", tc.replacement, 1)
			require.NoError(t, os.WriteFile(input, []byte(contents), 0600))
			out, _, err := exportCommand(t, "lint", input, "--format", "json", "--fail-severity", "error")
			if tc.invalid {
				var exitErr *ExitError
				require.ErrorAs(t, err, &exitErr)
				require.Equal(t, ExitCodeViolations, exitErr.Code)
			} else {
				require.NoError(t, err)
			}
			var diagnostics []vacuumreport.Diagnostic
			require.NoError(t, json.Unmarshal([]byte(out), &diagnostics))
			securityCount := 0
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == "oas3-operation-security-defined" {
					securityCount++
				}
				if !tc.invalid {
					require.NotEqual(t, "oas3-unused-component", diagnostic.Code)
				}
			}
			if tc.invalid {
				require.Equal(t, 1, securityCount)
			} else {
				require.Zero(t, securityCount)
			}
		})
	}
}
