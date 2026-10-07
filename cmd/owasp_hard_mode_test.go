package cmd

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"path/filepath"
	"testing"
)

func TestOWASPHardModeSelection(t *testing.T) {
	ids := []string{rulesets.OwaspOAuthNoPassword, rulesets.OwaspOAuthNoImplicit, rulesets.OwaspAuthURLsHTTPS}
	for _, tc := range []struct {
		name, spec string
		hard, want bool
	}{
		{"regular OpenAPI", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}", false, false},
		{"hard OpenAPI", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}", true, true},
		{"hard AsyncAPI", "asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {}", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rs, _, err := selectRuleSetForBuildResults(true, tc.hard, "", []byte(tc.spec), false, utils.HTTPClientConfig{}, nil)
			require.NoError(t, err)
			for _, id := range ids {
				_, exists := rs.Rules[id]
				assert.Equal(t, tc.want, exists, id)
			}
		})
	}
	schema := rulesets.BuildDefaultRuleSets().GenerateJSONSchemaDefaultRuleSet()
	for _, id := range ids {
		assert.NotContains(t, schema.Rules, id)
	}
}

func TestOWASPHardModePreservesSeverityOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ruleset.yaml")
	writeTestFile(t, path, "extends: [[vacuum:owasp, all]]\nrules:\n  owasp-oauth-no-password: warn\n")
	rs, _, err := selectRuleSetForBuildResults(true, true, path, []byte("openapi: 3.1.0\npaths: {}"), false, utils.HTTPClientConfig{}, nil)
	require.NoError(t, err)
	require.NotNil(t, rs.Rules[rulesets.OwaspOAuthNoPassword])
	assert.Equal(t, model.SeverityWarn, rs.Rules[rulesets.OwaspOAuthNoPassword].Severity)
}
