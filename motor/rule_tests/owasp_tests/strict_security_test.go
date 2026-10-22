package tests

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestRuleSet_OWASPStrictWriteSecurity(t *testing.T) {
	for _, tc := range []struct {
		name, global, operation string
		want                    int
	}{
		{"missing", "", "", 1},
		{"inherited", "security: [{auth: []}]", "", 0},
		{"override", "security: [{}]", "security: [{auth: []}]", 0},
		{"explicit empty", "security: [{auth: []}]", "security: []", 1},
		{"global empty", "security: []", "", 1},
		{"anonymous override", "security: [{auth: []}]", "security: [{}]", 1},
		{"anonymous alternative", "", "security: [{auth: []}, {}]", 1},
		{"anonymous inherited alternative", "security: [{auth: []}, {}]", "", 1},
		{"authenticated alternatives", "", "security: [{auth: []}, {other: []}]", 0},
	} {
		for _, method := range []string{"post", "put", "patch", "delete"} {
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				spec := fmt.Sprintf(`openapi: 3.1.0
info: {title: Test, version: '1'}
%s
paths:
  /items:
    %s:
      %s
      responses: {'200': {description: OK}}
components:
  securitySchemes:
    auth: {type: http, scheme: bearer}
    other: {type: apiKey, in: header, name: X-Key}
`, tc.global, method, tc.operation)
				rule := rulesets.GetOWASPProtectionGlobalUnsafeStrictRule()
				result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}})
				require.Empty(t, result.Errors)
				require.Len(t, result.Results, tc.want)
				for _, finding := range result.Results {
					assert.Equal(t, model.SeverityError, finding.Rule.Severity)
					assert.Positive(t, finding.StartNode.Line)
					assert.NotEmpty(t, finding.Path)
				}
			})
		}
	}
}
