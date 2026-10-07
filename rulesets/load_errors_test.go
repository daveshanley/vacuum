package rulesets

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestExtendedRuleset_LoadErrors(t *testing.T) {
	for _, name := range []string{"missing", "empty", "malformed", "nested", "circular", "HTTP"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			location := filepath.Join(dir, "child.yaml")
			switch name {
			case "empty":
				require.NoError(t, os.WriteFile(location, nil, 0600))
			case "malformed":
				require.NoError(t, os.WriteFile(location, []byte("rules: ["), 0600))
			case "nested":
				require.NoError(t, os.WriteFile(location, []byte(fmt.Sprintf("extends: [%q]", filepath.Join(dir, "missing.yaml"))), 0600))
			case "circular":
				require.NoError(t, os.WriteFile(location, []byte(fmt.Sprintf("extends: [%q]", location)), 0600))
			case "HTTP":
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte("rules: {}")) // valid YAML must not hide an HTTP failure
				}))
				defer server.Close()
				location = server.URL
			}
			input, err := CreateRuleSetFromData([]byte(fmt.Sprintf("extends: [[vacuum:oas, all], %q]", location)))
			require.NoError(t, err)
			defaults := BuildDefaultRuleSets()
			generated := defaults.GenerateRuleSetFromSuppliedRuleSet(input)
			require.Error(t, generated.LoadError())
			if name == "nested" {
				location = filepath.Join(dir, "missing.yaml")
			}
			require.Contains(t, generated.LoadError().Error(), location)
			require.NoError(t, defaults.GenerateOpenAPIDefaultRuleSet().LoadError(), "failed load must not invalidate built-in defaults")
		})
	}
}
