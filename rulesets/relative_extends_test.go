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

const relativeChildRule = "rules:\n  child-rule:\n    given: $.info.title\n    severity: error\n    then: {function: falsy}\n"

func TestExtendedRuleset_RelativeLocalPaths(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "nested"), 0700))
	for name, data := range map[string]string{
		"main.yaml":          "extends: [./nested/parent.yaml]\n",
		"nested/parent.yaml": "extends: [../child.yaml]\n",
		"child.yaml":         relativeChildRule,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(data), 0600))
	}
	input, err := LoadLocalRuleSet(context.Background(), filepath.Join(dir, "main.yaml"))
	require.NoError(t, err)
	generated := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(input)
	require.NoError(t, generated.LoadError())
	require.Contains(t, generated.Rules, "child-rule")
	// A relative return to the containing ruleset is still a cycle.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child.yaml"), []byte("extends: [./nested/../main.yaml]\n"), 0600))
	generated = BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(input)
	require.ErrorContains(t, generated.LoadError(), "circular ruleset extension")
}

func TestExtendedRuleset_RelativeRemotePaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start.yaml":
			http.Redirect(w, r, "/rules/main.yaml", http.StatusFound)
		case "/rules/main.yaml":
			fmt.Fprint(w, "extends: [./nested/parent.yaml]\n")
		case "/rules/nested/parent.yaml":
			fmt.Fprint(w, "extends: [../child.yaml?rev=1]\n")
		case "/rules/child.yaml":
			if r.URL.Query().Get("rev") != "1" {
				http.Error(w, "missing revision", 400)
				return
			}
			fmt.Fprint(w, relativeChildRule)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	input, err := DownloadRemoteRuleSet(context.Background(), server.URL+"/start.yaml", server.Client())
	require.NoError(t, err)
	generated := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(input, server.Client())
	require.NoError(t, generated.LoadError())
	require.Contains(t, generated.Rules, "child-rule")
}

func TestExtendedRuleset_RemoteReferencesNeverLoadLocalFiles(t *testing.T) {
	for _, child := range []string{"./invalid%name.yaml", "file:///tmp/private-rules.yaml"} {
		t.Run(child, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "extends: [%q]\n", child) }))
			defer server.Close()
			input, err := DownloadRemoteRuleSet(context.Background(), server.URL+"/rules.yaml", server.Client())
			require.NoError(t, err)
			generated := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(input, server.Client())
			require.Error(t, generated.LoadError())
			require.Contains(t, generated.LoadError().Error(), "remote ruleset")
			require.NotContains(t, generated.LoadError().Error(), "no such file")
		})
	}
}
