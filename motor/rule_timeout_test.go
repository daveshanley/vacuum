package motor

import (
	"runtime"
	"strconv"
	"sync/atomic"
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

type concurrencyTrackingRule struct {
	active    atomic.Int32
	maxActive atomic.Int32
	sleep     time.Duration
}

func (c *concurrencyTrackingRule) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "trackConcurrency",
	}
}

func (c *concurrencyTrackingRule) GetCategory() string {
	return model.CategoryValidation
}

func (c *concurrencyTrackingRule) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	active := c.active.Add(1)
	for {
		maxActive := c.maxActive.Load()
		if active <= maxActive || c.maxActive.CompareAndSwap(maxActive, active) {
			break
		}
	}
	time.Sleep(c.sleep)
	c.active.Add(-1)

	return []model.RuleFunctionResult{
		{
			Message: "tracked rule finished",
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

func TestRuleRunnerConcurrencyIgnoresGOMAXPROCSOne(t *testing.T) {
	previousProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previousProcs)

	yml := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}`

	rules := make(map[string]*model.Rule)
	for i := 0; i < 16; i++ {
		ruleID := "track-" + strconv.Itoa(i)
		rules[ruleID] = &model.Rule{
			Id:           ruleID,
			Given:        "$",
			RuleCategory: model.RuleCategories[model.CategoryValidation],
			Type:         rulesets.Validation,
			Severity:     model.SeverityError,
			Then: model.RuleAction{
				Function: "trackConcurrency",
			},
		}
	}
	tracker := &concurrencyTrackingRule{sleep: 50 * time.Millisecond}

	ex := &RuleSetExecution{
		RuleSet: &rulesets.RuleSet{
			Rules: rules,
		},
		Spec:    []byte(yml),
		Timeout: time.Second,
		CustomFunctions: map[string]model.RuleFunction{
			"trackConcurrency": tracker,
		},
		SilenceLogs: true,
	}

	start := time.Now()
	results := ApplyRulesToRuleSet(ex)
	elapsed := time.Since(start)

	assert.Len(t, results.Results, len(rules))
	assert.Greater(t, tracker.maxActive.Load(), int32(1))
	assert.Less(t, elapsed, 500*time.Millisecond)
}
