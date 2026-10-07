package rulesets

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestIssue941UnsupportedRulesetReferences(t *testing.T) {
	for _, location := range []string{
		"spectral:arazzo", "Specs.Ruleset/spectral:arazzo", `Specs.Ruleset\spectral:arazzo`,
		"rules.mjs", "rules.js", "rules.cjs", "rules.MJS",
		`C:\repo\rules.mjs`, `C:\repo\rules.js`, `C:\repo\rules.cjs`, `\\server\share\rules.mjs`,
		"https://unpkg.com/@stoplight/spectral-owasp-ruleset/dist/ruleset.mjs?version=1",
	} {
		t.Run(location, func(t *testing.T) {
			data := []byte(fmt.Sprintf("extends: [%q]\n", location))
			_, err := CreateRuleSetFromData(data)
			require.ErrorContains(t, err, "not supported")
			rs := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(&RuleSet{Extends: location})
			require.ErrorContains(t, rs.LoadError(), "not supported")
		})
	}
	for _, location := range []string{"spectral:oas", "vacuum:owasp", "./child.yaml", "https://example.com/spectral:arazzo"} {
		require.NoError(t, unsupportedRulesetReference(location), location)
	}
}

func TestIssue941ScriptLoadersReportMigration(t *testing.T) {
	_, err := LoadLocalRuleSet(context.Background(), filepath.Join(t.TempDir(), "rules.mjs"))
	require.ErrorContains(t, err, "#spectral-migration")
	_, err = DownloadRemoteRuleSet(context.Background(), "https://example.invalid/rules.mjs", nil)
	require.ErrorContains(t, err, "JavaScript ruleset")
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/rules" {
			http.Redirect(w, r, "/rules.mjs", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("export default {};"))
	}))
	defer server.Close()
	_, err = DownloadRemoteRuleSet(context.Background(), server.URL+"/rules", server.Client())
	require.ErrorContains(t, err, "JavaScript ruleset")
	require.Equal(t, 2, calls)
}

func TestIssue941NestedUnsupportedRulesetAndNativeMigration(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "main.yaml")
	child := filepath.Join(dir, "child.yaml")
	require.NoError(t, os.WriteFile(parent, []byte("extends: [./child.yaml, [vacuum:owasp, all]]\n"), 0600))
	require.NoError(t, os.WriteFile(child, []byte("extends: [spectral:arazzo]\n"), 0600))
	loaded, err := LoadLocalRuleSet(context.Background(), parent)
	require.NoError(t, err)
	rs := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(loaded)
	require.ErrorContains(t, rs.LoadError(), "Arazzo")
	require.NoError(t, os.WriteFile(child, []byte("extends: [spectral:oas]\nrules:\n  local-title:\n    given: $.info.title\n    then: {function: truthy}\n"), 0600))
	loaded, err = LoadLocalRuleSet(context.Background(), parent)
	require.NoError(t, err)
	rs = BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(loaded)
	require.NoError(t, rs.LoadError())
	require.Contains(t, rs.Rules, "local-title")
	require.Contains(t, rs.Rules, OwaspArrayLimit)
	require.Contains(t, rs.Rules, OwaspStringLimit)
}
