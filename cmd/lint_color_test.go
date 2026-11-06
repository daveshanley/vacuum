package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestLintColorEnvironment(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "openapi.yaml")
	rules := filepath.Join(dir, "ruleset.yaml")
	require.NoError(t, os.WriteFile(spec, []byte(`openapi: 3.0.3
info: {title: Repro, version: 1.0.0}
paths:
  /things:
    get:
      operationId: Get things
      responses:
        '200': {description: OK}
`), 0600))
	require.NoError(t, os.WriteFile(rules, []byte(`rules:
  valid-operation-id:
    given: $.paths.*.*.operationId
    severity: error
    message: Operation ID contains a space
    then:
      function: pattern
      functionOptions: {notMatch: ' '}
`), 0600))

	for _, tc := range []struct {
		name     string
		force    string
		forceSet bool
		noColor  string
		args     []string
		colored  bool
	}{
		{name: "pipe defaults to plain"},
		{name: "force colors in pipe", force: "1", forceSet: true, colored: true},
		{name: "empty force enables colors", forceSet: true, colored: true},
		{name: "force 256 colors", force: "2", forceSet: true, colored: true},
		{name: "force true colors", force: "3", forceSet: true, colored: true},
		{name: "zero disables colors", force: "0", forceSet: true},
		{name: "no color overrides force", force: "1", forceSet: true, noColor: "1"},
		{name: "zero no color is nonempty", force: "1", forceSet: true, noColor: "0"},
		{name: "no style overrides force", force: "1", forceSet: true, args: []string{"--no-style"}},
		{name: "pipeline keeps markdown output", args: []string{"--pipeline-output"}},
		{name: "pipeline respects no color", noColor: "1", args: []string{"--pipeline-output"}},
		{name: "pipeline respects force zero", force: "0", forceSet: true, args: []string{"--pipeline-output"}},
		{name: "pipeline respects no style", force: "1", forceSet: true, args: []string{"--pipeline-output", "--no-style"}},
		{name: "json stays plain", force: "1", forceSet: true, args: []string{"--format", "json", "--fail-severity", "none"}},
		{name: "annotations stay plain", force: "1", forceSet: true, args: []string{"--github-annotations"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FORCE_COLOR", tc.force)
			if !tc.forceSet {
				require.NoError(t, os.Unsetenv("FORCE_COLOR"))
			}
			t.Setenv("NO_COLOR", tc.noColor)
			args := []string{"lint", spec, "--ruleset", rules, "--no-banner", "--no-update-check"}
			wantCode := 0
			if !slices.Contains(tc.args, "--format") {
				args = append(args, "--details", "--no-clip")
				wantCode = 1
			}
			output, code := runVacuum(t, append(args, tc.args...)...)
			require.Equal(t, wantCode, code, "%s", output)
			require.Equal(t, tc.colored, strings.Contains(string(output), "\x1b["), "%s", output)
			switch {
			case slices.Contains(tc.args, "--format"):
				require.True(t, json.Valid(output), "%s", output)
			case slices.Contains(tc.args, "--github-annotations"):
				require.Contains(t, string(output), "::error ")
			default:
				require.Contains(t, strings.ToLower(string(output)), "failed with")
				require.Contains(t, string(output), "valid-operation-id")
				if !slices.Contains(tc.args, "--pipeline-output") {
					require.Contains(t, string(output), "Operation ID contains a space")
					for line := range strings.SplitSeq(string(output), "\n") {
						if strings.Contains(line, "Operation ID contains a space") {
							require.Equal(t, tc.colored, strings.Contains(line, "\x1b["), "%s", line)
						}
					}
				}
			}
		})
	}
}
