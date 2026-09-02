package motor

import (
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
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

type channelBlockingRule struct {
	started chan struct{}
	unblock <-chan struct{}
}

func (c *channelBlockingRule) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "channelBlocking"}
}

func (c *channelBlockingRule) GetCategory() string {
	return model.CategoryValidation
}

func (c *channelBlockingRule) RunRule(_ []*yaml.Node, _ model.RuleFunctionContext) []model.RuleFunctionResult {
	close(c.started)
	<-c.unblock
	return nil
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

func TestRuleTimeout_ReleaseOwnedResourcesReturnsPromptlyForTimedOutRule(t *testing.T) {
	yml := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}`

	started := make(chan struct{})
	unblock := make(chan struct{})
	var unblockOnce sync.Once
	unblockRule := func() {
		unblockOnce.Do(func() { close(unblock) })
	}
	defer unblockRule()

	rule := &channelBlockingRule{started: started, unblock: unblock}
	execution := &RuleSetExecution{
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{
			"channel-blocking": {
				Id:           "channel-blocking",
				Given:        "$",
				RuleCategory: model.RuleCategories[model.CategoryValidation],
				Type:         rulesets.Validation,
				Severity:     model.SeverityError,
				Then: model.RuleAction{
					Function: "channelBlocking",
				},
			},
		}},
		Spec:    []byte(yml),
		Timeout: 20 * time.Millisecond,
		CustomFunctions: map[string]model.RuleFunction{
			"channelBlocking": rule,
		},
		SilenceLogs: true,
	}

	applyDone := make(chan *RuleSetExecutionResult, 1)
	go func() {
		applyDone <- ApplyRulesToRuleSet(execution)
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("custom rule did not start")
	}

	var results *RuleSetExecutionResult
	select {
	case results = <-applyDone:
	case <-time.After(time.Second):
		t.Fatal("ApplyRulesToRuleSet did not return promptly after the rule timeout")
	}
	assert.NotNil(t, results)

	assert.NotNil(t, execution.DrDocument)
	assert.NotNil(t, execution.IndexResolved)
	assert.NotNil(t, execution.IndexUnresolved)

	releaseReturned := make(chan struct{})
	go func() {
		results.ReleaseOwnedResources()
		close(releaseReturned)
	}()

	releaseWasPrompt := false
	select {
	case <-releaseReturned:
		releaseWasPrompt = true
	case <-time.After(time.Second):
	}
	assert.True(t, releaseWasPrompt, "ReleaseOwnedResources blocked on a timed-out rule")
	assert.Nil(t, results.RuleSetExecution)
	cleanupDone := results.releaseDone
	assert.NotNil(t, cleanupDone)
	select {
	case <-cleanupDone:
		t.Fatal("resources were released while the timed-out rule was still blocked")
	default:
	}
	assert.NotNil(t, execution.DrDocument)
	assert.NotNil(t, execution.IndexResolved)
	assert.NotNil(t, execution.IndexUnresolved)

	unblockRule()
	select {
	case <-releaseReturned:
	case <-time.After(5 * time.Second):
		t.Fatal("ReleaseOwnedResources did not return after the rule was unblocked")
	}
	select {
	case <-cleanupDone:
	case <-time.After(5 * time.Second):
		t.Fatal("deferred resource cleanup did not finish after the rule was unblocked")
	}
	assert.Nil(t, execution.DrDocument)
	assert.Nil(t, execution.IndexResolved)
	assert.Nil(t, execution.IndexUnresolved)
	assert.Nil(t, execution.CanonicalDocument)
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
