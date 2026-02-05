package motor

import (
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

type slowRule struct {
	sleep time.Duration
}

func (s *slowRule) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "slow",
	}
}

func (s *slowRule) GetCategory() string {
	return model.CategoryValidation
}

func (s *slowRule) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	time.Sleep(s.sleep)
	return []model.RuleFunctionResult{
		{
			Message: "slow rule finished",
		},
	}
}

func TestRuleTimeout_DropsResults(t *testing.T) {
	yml := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}`

	rules := map[string]*model.Rule{
		"slow": {
			Id:           "slow",
			Given:        "$",
			RuleCategory: model.RuleCategories[model.CategoryValidation],
			Type:         rulesets.Validation,
			Severity:     model.SeverityError,
			Then: model.RuleAction{
				Function: "slow",
			},
		},
	}

	ex := &RuleSetExecution{
		RuleSet: &rulesets.RuleSet{
			Rules: rules,
		},
		Spec:    []byte(yml),
		Timeout: 20 * time.Millisecond,
		CustomFunctions: map[string]model.RuleFunction{
			"slow": &slowRule{sleep: 100 * time.Millisecond},
		},
	}

	results := ApplyRulesToRuleSet(ex)
	assert.Len(t, results.Results, 0)

	time.Sleep(150 * time.Millisecond)
	assert.Len(t, results.Results, 0)
}
