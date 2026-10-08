package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestRulesetMigrationErrorsAreVisible(t *testing.T) {
	for _, tc := range []struct{ name, filename, contents, message string }{
		{"local module", "rules.mjs", "export default {};", "JavaScript ruleset"},
		{"remote module", "rules.yaml", "extends: [https://example.invalid/rules.mjs]\n", "#spectral-migration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			spec := filepath.Join(dir, "api.yaml")
			rules := filepath.Join(dir, tc.filename)
			require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"), 0600))
			require.NoError(t, os.WriteFile(rules, []byte(tc.contents), 0600))
			output, code := runVacuum(t, "lint", spec, "--ruleset", rules, "--no-update-check", "--no-banner", "--no-style")
			require.Equal(t, ExitCodeInputError, code)
			require.Contains(t, string(output), tc.message)
			require.NotContains(t, string(output), "100/100")
			require.NotContains(t, string(output), "go-yaml load error")
		})
	}
}
