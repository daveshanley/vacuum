package tests

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestRuleSet_OWASPOAuthFlows(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.0", "3.2.0"} {
		for _, tc := range []struct {
			flow string
			rule *model.Rule
			want int
		}{
			{"password", rulesets.GetOWASPOAuthNoPasswordRule(), 1},
			{"implicit", rulesets.GetOWASPOAuthNoImplicitRule(), 1},
			{"authorizationCode", rulesets.GetOWASPOAuthNoPasswordRule(), 0},
			{"clientCredentials", rulesets.GetOWASPOAuthNoImplicitRule(), 0},
			{"password", rulesets.GetOWASPOAuthNoImplicitRule(), 0},
			{"implicit", rulesets.GetOWASPOAuthNoPasswordRule(), 0},
		} {
			t.Run(version+"/"+tc.flow+"/"+tc.rule.Id, func(t *testing.T) {
				spec := fmt.Sprintf(`openapi: %s
info: {title: Test, version: '1'}
paths: {}
components:
  securitySchemes:
    auth:
      type: oauth2
      flows:
        %s:
          authorizationUrl: https://example.com/authorize
          tokenUrl: https://example.com/token
          scopes: {}
`, version, tc.flow)
				result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
					Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{tc.rule.Id: tc.rule}},
				})
				require.Empty(t, result.Errors)
				require.Len(t, result.Results, tc.want)
				if tc.want > 0 {
					finding := result.Results[0]
					assert.Equal(t, "$.components.securitySchemes['auth'].flows."+tc.flow, finding.Path)
					assert.Equal(t, model.SeverityError, finding.Rule.Severity)
					assert.Positive(t, finding.StartNode.Line)
					assert.Contains(t, finding.Message, "PKCE")
				}
			})
		}
	}
}

func TestRuleSet_OWASPAuthURLsHTTPS(t *testing.T) {
	for _, tc := range []struct {
		name, scheme, field, endpoint string
		want                          int
	}{
		{"authorization", "oauth2", "authorizationUrl", "http://example.com/authorize", 1},
		{"token", "oauth2", "tokenUrl", "http://example.com/token", 1},
		{"refresh", "oauth2", "refreshUrl", "http://example.com/refresh", 1},
		{"discovery", "openIdConnect", "openIdConnectUrl", "http://example.com/.well-known/openid-configuration", 1},
		{"secure", "oauth2", "tokenUrl", "https://example.com/token", 0},
		{"relative", "oauth2", "tokenUrl", "/token", 0},
		{"relative discovery", "openIdConnect", "openIdConnectUrl", "/.well-known/openid-configuration", 0},
		{"scheme prefix", "oauth2", "tokenUrl", "https-insecure://example.com/token", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  securitySchemes:\n    auth:\n      type: " + tc.scheme + "\n"
			path := "$.components.securitySchemes['auth']."
			if tc.scheme == "oauth2" {
				spec += "      flows:\n        authorizationCode:\n          scopes: {}\n          " + tc.field + ": " + tc.endpoint + "\n"
				path += "flows.authorizationCode."
			} else {
				spec += "      " + tc.field + ": " + tc.endpoint + "\n"
			}
			rule := rulesets.GetOWASPAuthURLsHTTPSRule()
			rule.Message = "use secure authentication transport"
			result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}})
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, tc.want)
			if tc.want > 0 {
				assert.Equal(t, path+tc.field, result.Results[0].Path)
				assert.Equal(t, rule.Message, result.Results[0].Message)
				assert.Positive(t, result.Results[0].StartNode.Line)
			}
		})
	}
}

func TestRuleSet_OWASPOAuthReferencedScheme(t *testing.T) {
	spec := `openapi: 3.1.0
info: {title: Test, version: '1'}
paths: {}
components:
  securitySchemes:
    alias:
      $ref: '#/components/securitySchemes/auth'
    auth:
      type: oauth2
      flows:
        password:
          tokenUrl: http://example.com/token
          scopes: {}
`
	rules := map[string]*model.Rule{
		rulesets.OwaspOAuthNoPassword: rulesets.GetOWASPOAuthNoPasswordRule(),
		rulesets.OwaspAuthURLsHTTPS:   rulesets.GetOWASPAuthURLsHTTPSRule(),
	}
	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: rules}})
	require.Empty(t, result.Errors)
	require.NotEmpty(t, result.Results)
	found := map[string]bool{}
	for _, finding := range result.Results {
		found[finding.Rule.Id] = true
		assert.Positive(t, finding.StartNode.Line)
	}
	assert.True(t, found[rulesets.OwaspOAuthNoPassword])
	assert.True(t, found[rulesets.OwaspAuthURLsHTTPS])
}

func TestRuleSet_OWASPOAuthInlineIgnore(t *testing.T) {
	spec := `openapi: 3.1.0
info: {title: Test, version: '1'}
paths: {}
components:
  securitySchemes:
    auth:
      x-lint-ignore: [owasp-oauth-no-password, owasp-auth-urls-https]
      type: oauth2
      flows:
        password:
          tokenUrl: http://example.com/token
          scopes: {}
`
	a, b := rulesets.GetOWASPOAuthNoPasswordRule(), rulesets.GetOWASPAuthURLsHTTPSRule()
	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{a.Id: a, b.Id: b}}})
	require.Empty(t, result.Errors)
	assert.Empty(t, result.Results)
	assert.Len(t, result.IgnoredResults, 2)
}

func TestRuleSet_OWASPOAuthFlowRejectsInvalidOptions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options map[string]interface{}
	}{
		{"missing", nil},
		{"unknown", map[string]interface{}{"flow": "foo"}},
		{"empty", map[string]interface{}{"flow": ""}},
		{"wrong type", map[string]interface{}{"flow": true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := &model.Rule{
				Id:       "oauth-flow",
				Given:    "$",
				Severity: model.SeverityError,
				Then: model.RuleAction{
					Function:        "owaspOAuthFlow",
					FunctionOptions: tc.options,
				},
			}
			result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
				Spec:    []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"),
				RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}},
			})
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 1)
			assert.Contains(t, result.Results[0].Message, "'flow' to be 'password' or 'implicit'")
		})
	}
}
