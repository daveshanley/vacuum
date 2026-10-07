package cmd

import (
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
	"os"
	"testing"
)

func TestLocalRulesetWithHTTPPrefix(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile("httpclient-rules.yaml", []byte("rules:\n  title:\n    given: $.info.title\n    then: {function: truthy}\n"), 0600))
	for _, remote := range []bool{false, true} {
		rs, err := BuildRuleSetFromUserSuppliedLocation("httpclient-rules.yaml", rulesets.BuildDefaultRuleSets(), remote, nil)
		require.NoError(t, err)
		require.Contains(t, rs.Rules, "title")
	}
}
