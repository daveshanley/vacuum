package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/testify/require"
)

func TestRulesetMigrationErrorsAreVisible(t *testing.T) {
	for _, tc := range []struct{ name, filename, contents, message string }{
		{"local module", "rules.mjs", "export default ()=>({});", "Ruleset export must be an object"},
		{"remote module", "rules.yaml", "extends: [https://example.invalid/rules.mjs]\n", "select a local JavaScript entry"},
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

func TestJavaScriptRulesetCLI(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
	}{
		{"clean", `return undefined`, 0},
		{"finding", `return [{message:'missing title'}]`, 1},
		{"failure", `throw Error('broken function')`, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			spec := filepath.Join(dir, "api.yaml")
			rules := filepath.Join(dir, "rules.mjs")
			require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}"), 0600))
			require.NoError(t, os.WriteFile(rules, []byte(`export default {rules:{check:{severity:'error',given:'$.info',then:{function:()=>{`+tc.body+`}}}}}`), 0600))
			output, code := runVacuum(t, "lint", spec, "-r", rules, "--no-update-check", "--no-banner", "--no-style", "--fail-severity", "error")
			require.Equal(t, tc.code, code, string(output))
		})
	}
}

func TestJavaScriptRulesetKeyFieldCLI(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.yaml")
	rules := filepath.Join(dir, "rules.mjs")
	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.1.0\ninfo: {title: Example, version: '1'}\npaths: {}\nx-keys: [{red: false}, [false], {}]\n"), 0600))
	require.NoError(t, os.WriteFile(rules, []byte(`export default {rules: {keys: {
      severity: 'error', given: "$['x-keys'][*]", then: {field: '@key',
        function: (key, options, ctx) => [{message: String(key) + ' at ' + ctx.path.join('/')}]
      }
    }}}`), 0600))
	output, code := runVacuum(t, "spectral-report", spec, "-r", rules, "--stdout", "--no-update-check")
	require.Zero(t, code, string(output))
	var got []reports.SpectralReport
	require.NoError(t, json.Unmarshal(output, &got))
	require.Len(t, got, 3)
	require.Equal(t, []string{"x-keys", "0", "red"}, got[0].Path)
	require.Equal(t, "red at x-keys/0/red", got[0].Message)
	require.Equal(t, []string{"x-keys", "1", "0"}, got[1].Path)
	require.Equal(t, "0 at x-keys/1/0", got[1].Message)
	require.Equal(t, []string{"x-keys", "2"}, got[2].Path)
	require.Equal(t, "undefined at x-keys/2", got[2].Message)
}
