package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	htmlreport "github.com/daveshanley/vacuum/html-report"
	"github.com/daveshanley/vacuum/rulesets"
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
	for _, failure := range []struct{ name, rules, message string }{
		{"missing", "extends: ['./missing.yaml']\n", "missing.yaml"},
		{"unknown-matching", "rules:\n  typo:\n    given: '$'\n    severity: info\n    then: {function: patternd}\n", "patternd"},
		{"unknown-unmatched", "rules:\n  typo:\n    given: '$.missing'\n    severity: info\n    then: [{function: truthy}, {function: patternd}]\n", "patternd"},
	} {
		for _, command := range []string{"lint", "report", "spectral-report", "html-report", "schema"} {
			t.Run(failure.name+"/"+command, func(t *testing.T) {
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
				require.NoError(t, os.WriteFile(rules, []byte(failure.rules), 0600))
				output := filepath.Join(dir, "report-output")
				args := []string{"-test.run=^TestRulesetConfigurationErrors$", "--", command, spec}
				if strings.Contains(command, "report") {
					args = append(args, output)
				}
				args = append(args, "--ruleset", rules, "--no-update-check")
				if command == "lint" || command == "schema" {
					args = append(args, "--fail-severity", "none")
				}
				child := exec.Command(os.Args[0], args...)
				child.Dir = dir
				child.Env = append(os.Environ(), "VACUUM_TEST_RULESET_EXIT=1", "XDG_CONFIG_HOME="+dir)
				out, err := child.CombinedOutput()
				var exit *exec.ExitError
				require.ErrorAs(t, err, &exit, "%s", out)
				require.Equal(t, ExitCodeInputError, exit.ExitCode(), "%s", out)
				require.Contains(t, string(out), failure.message)
				require.NotContains(t, string(out), "100/100")
				matches, err := filepath.Glob(output + "*")
				require.NoError(t, err)
				require.Empty(t, matches, "invalid ruleset must not write a successful report")
			})
		}
	}
}

func TestRulesetRelativeExtensions(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.yaml"), []byte("extends: [./child.yaml]\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child.yaml"), []byte("rules:\n  child-rule:\n    given: $.info.title\n    then: {function: falsy}\n"), 0600))
	rs, err := BuildRuleSetFromUserSuppliedLocation(filepath.Join(dir, "main.yaml"), rulesets.BuildDefaultRuleSets(), false, nil)
	require.NoError(t, err)
	require.Contains(t, rs.Rules, "child-rule")
}
