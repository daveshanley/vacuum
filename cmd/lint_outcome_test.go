package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestLintOutcomeMatchesStatus(t *testing.T) {
	if os.Getenv("VACUUM_TEST_OUTCOME") == "1" {
		for i, arg := range os.Args {
			if arg == "--" {
				os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
				break
			}
		}
		Execute("test", "test", "test")
		os.Exit(0)
	}
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
		for _, plain := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/plain=%v", tc.name, plain), func(t *testing.T) {
				dir := t.TempDir()
				spec := filepath.Join(dir, "spec.yaml")
				rules := filepath.Join(dir, "rules.yaml")
				contents := "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"
				if tc.command == "schema" {
					contents = "title: Test\ntype: string\n"
				}
				require.NoError(t, os.WriteFile(spec, []byte(contents), 0600))
				require.NoError(t, os.WriteFile(rules, []byte(fmt.Sprintf("rules:\n  fail-title:\n    given: '$..title'\n    severity: %s\n    then: {function: falsy}\n", tc.severity)), 0600))
				args := []string{"-test.run=^TestLintOutcomeMatchesStatus$", "--", tc.command, spec, "--ruleset", rules, "--fail-severity", tc.threshold, "--no-update-check"}
				if tc.command == "lint" {
					args = append(args, "--min-score", fmt.Sprint(tc.score), "--no-banner")
				}
				if plain {
					args = append(args, "--no-style")
				}
				process := exec.Command(os.Args[0], args...)
				process.Dir = dir
				process.Env = append(os.Environ(), "VACUUM_TEST_OUTCOME=1", "XDG_CONFIG_HOME="+dir)
				output, err := process.CombinedOutput()
				if tc.want == 0 {
					require.NoError(t, err, "%s", output)
					require.Contains(t, string(output), "Passed")
					require.NotContains(t, string(output), "Failed with")
				} else {
					var exit *exec.ExitError
					require.ErrorAs(t, err, &exit, "%s", output)
					require.Equal(t, tc.want, exit.ExitCode())
					require.Contains(t, string(output), "Failed with")
					require.False(t, strings.Contains(string(output), "Passed"), "%s", output)
				}
			})
		}
	}
}
