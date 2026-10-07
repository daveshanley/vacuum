package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestRulesetMigrationErrorsAreVisible(t *testing.T) {
	for _, tc := range []struct{ name, filename, contents, message string }{
		{"local module", "rules.mjs", "export default {};", "JavaScript ruleset"},
		{"remote module", "rules.yaml", "extends: [https://example.invalid/rules.mjs]\n", "vacuum:owasp"},
		{"Arazzo", "rules.yaml", "extends: [spectral:arazzo]\n", "Arazzo ruleset"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			spec := filepath.Join(dir, "api.yaml")
			rules := filepath.Join(dir, tc.filename)
			require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"), 0600))
			require.NoError(t, os.WriteFile(rules, []byte(tc.contents), 0600))
			process := exec.Command(os.Args[0], "-test.run=^TestFlagErrorsAreVisible$", "--", "lint", spec, "--ruleset", rules, "--no-update-check", "--no-banner", "--no-style")
			process.Dir = dir
			process.Env = append(os.Environ(), "VACUUM_TEST_FLAG_ERROR=1", "XDG_CONFIG_HOME="+dir)
			output, err := process.CombinedOutput()
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit, "%s", output)
			require.Equal(t, ExitCodeInputError, exit.ExitCode())
			require.Contains(t, string(output), tc.message)
			require.NotContains(t, string(output), "100/100")
			require.NotContains(t, string(output), "go-yaml load error")
		})
	}
}
