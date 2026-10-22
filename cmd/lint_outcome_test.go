package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestLintOutcomeMatchesStatus(t *testing.T) {
	for _, tc := range []struct {
		name, command, severity, threshold string
		score                              int
		want                               int
	}{
		{"warning failure", "lint", "warn", "warn", 10, 1},
		{"info failure", "lint", "info", "info", 10, 1},
		{"hint failure", "lint", "hint", "hint", 10, 1},
		{"score failure", "lint", "warn", "error", 100, 1},
		{"warning allowed", "lint", "warn", "error", 10, 0},
		{"errors allowed", "lint", "error", "none", 10, 0},
		{"schema failure", "schema", "warn", "warn", 10, 1},
		{"schema allowed", "schema", "error", "none", 10, 0},
	} {
		for _, outputMode := range []string{"plain", "styled", "pipeline"} {
			t.Run(fmt.Sprintf("%s/%s", tc.name, outputMode), func(t *testing.T) {
				dir := t.TempDir()
				spec := filepath.Join(dir, "spec.yaml")
				rules := filepath.Join(dir, "rules.yaml")
				contents := "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"
				if tc.command == "schema" {
					contents = "title: Test\ntype: string\n"
				}
				require.NoError(t, os.WriteFile(spec, []byte(contents), 0600))
				require.NoError(t, os.WriteFile(rules, []byte(fmt.Sprintf("rules:\n  fail-title:\n    given: '$..title'\n    severity: %s\n    then: {function: falsy}\n", tc.severity)), 0600))
				args := []string{tc.command, spec, "--ruleset", rules, "--fail-severity", tc.threshold, "--no-update-check"}
				if tc.command == "lint" {
					args = append(args, "--min-score", fmt.Sprint(tc.score), "--no-banner")
				}
				if outputMode == "pipeline" && tc.command == "lint" {
					args = append(args, "--pipeline-output")
				}
				if outputMode != "styled" {
					args = append(args, "--no-style")
				}
				output, code := runVacuum(t, args...)
				require.Equal(t, tc.want, code, "%s", output)
				if tc.want == 0 {
					require.Contains(t, strings.ToLower(string(output)), "passed")
					require.NotContains(t, strings.ToLower(string(output)), "failed with")
				} else {
					require.Contains(t, strings.ToLower(string(output)), "failed with")
					require.False(t, strings.Contains(strings.ToLower(string(output)), "passed"), "%s", output)
				}
			})
		}
	}
}
