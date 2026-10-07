package motor

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestComponentNameFindingsReachDoctor(t *testing.T) {
	for _, version := range []string{"3.0.4", "3.1.1"} {
		t.Run(version, func(t *testing.T) {
			rule := &model.Rule{Id: "oas3-schema", Given: "$", Severity: "error", Resolved: true, Then: model.RuleAction{Function: "oasDocumentSchema"}}
			spec := fmt.Sprintf("openapi: %s\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Bad Name:\n      type: string\n", version)
			result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}})
			defer result.Release()
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 1)
			stored := result.RuleSetExecution.DrDocument.V3Document.GetRuleFunctionResults()
			require.Len(t, stored, 1)
			require.Equal(t, result.Results[0].Message, stored[0].Message)
			require.Equal(t, result.Results[0].Path, stored[0].Path)
			require.Equal(t, "Bad Name", stored[0].StartNode.Value)
		})
	}
}
