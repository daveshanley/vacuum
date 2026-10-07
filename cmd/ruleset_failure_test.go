package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	htmlreport "github.com/daveshanley/vacuum/html-report"
	"github.com/pb33f/testify/require"
)

// Exercise the real process exit contract, including commands that return a
// plain error instead of an ExitError.
func TestRulesetConfigurationErrors(t *testing.T) {
	if os.Getenv("VACUUM_TEST_RULESET_EXIT") == "1" {
		for i, arg := range os.Args {
			if arg == "--" {
				os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
				break
			}
		}
		Execute("test", "test", "test")
		os.Exit(0)
	}
	for _, command := range []string{"lint", "report", "spectral-report", "html-report", "schema"} {
		t.Run(command, func(t *testing.T) {
			if command == "html-report" && !htmlreport.AssetsAvailable() {
				t.Skip("requires html_report_ui build")
			}
			dir := t.TempDir()
			spec := filepath.Join(dir, "spec.yaml")
			contents := "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"
			if command == "schema" {
				contents = "type: object\n"
			}
			require.NoError(t, os.WriteFile(spec, []byte(contents), 0600))
			rules := filepath.Join(dir, "rules.yaml")
			require.NoError(t, os.WriteFile(rules, []byte("extends: ['./missing.yaml']\n"), 0600))
			output := filepath.Join(dir, "report-output")
			args := []string{"-test.run=^TestRulesetConfigurationErrors$", "--", command, spec}
			if strings.Contains(command, "report") {
				args = append(args, output)
			}
			args = append(args, "--ruleset", rules, "--no-update-check")
			child := exec.Command(os.Args[0], args...)
			child.Dir = dir
			child.Env = append(os.Environ(), "VACUUM_TEST_RULESET_EXIT=1", "XDG_CONFIG_HOME="+dir)
			out, err := child.CombinedOutput()
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit, "%s", out)
			require.Equal(t, ExitCodeInputError, exit.ExitCode(), "%s", out)
			require.Contains(t, string(out), "missing.yaml")
			require.NotContains(t, string(out), "100/100")
			matches, err := filepath.Glob(output + "*")
			require.NoError(t, err)
			require.Empty(t, matches, "invalid ruleset must not write a successful report")
		})
	}
}
