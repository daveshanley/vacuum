package rulesets

import (
	"context"
	"github.com/pb33f/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestHTTPPrefixedLocalRulesetName(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "parent.yaml")
	require.NoError(t, os.WriteFile(parent, []byte("extends: [httpclient-rules.yaml]"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "httpclient-rules.yaml"), []byte("rules:\n  title:\n    given: $.info.title\n    then: {function: truthy}\n"), 0600))
	supplied, err := LoadLocalRuleSet(context.Background(), parent)
	require.NoError(t, err)
	result := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(supplied)
	require.NoError(t, result.LoadError())
	require.Contains(t, result.Rules, "title")
	require.False(t, CheckForRemoteExtends(map[string]string{"httpclient-rules.yaml": "all"}))
	for _, location := range []string{"http://example.com/rules", "https://example.com/rules"} {
		require.True(t, CheckForRemoteExtends(map[string]string{location: "all"}))
		target, err := resolveRulesetLocation(rulesetLocation{}, location)
		require.NoError(t, err)
		require.NotNil(t, target.remoteURL)
	}
}
