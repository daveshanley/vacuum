// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/libasyncapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

type captureAsyncAPIContextFunction struct {
	sawAsyncAPI *atomic.Bool
}

func (c captureAsyncAPIContextFunction) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.AsyncAPI != nil {
		c.sawAsyncAPI.Store(true)
	}
	return nil
}

func (c captureAsyncAPIContextFunction) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "captureAsyncAPIContext"}
}

func (c captureAsyncAPIContextFunction) GetCategory() string {
	return model.FunctionCategoryCore
}

func TestAsyncAPIExecutionBuildsAsyncContext(t *testing.T) {
	ruleSet := rulesets.BuildDefaultRuleSets().GenerateAsyncAPIRecommendedRuleSet()
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: ruleSet,
		Spec:    []byte(validAsyncAPI31Fixture),
	})

	require.Empty(t, result.Errors)
	require.NotNil(t, result.AsyncAPI)
	require.NotNil(t, result.Index)
	require.NotNil(t, result.SpecInfo)
	assert.Equal(t, model.AsyncAPI31, result.SpecInfo.SpecFormat)
	assert.Equal(t, "asyncapi", result.SpecInfo.SpecType)
	assert.Nil(t, result.RuleSetExecution.Document)
	assert.Nil(t, result.RuleSetExecution.DrDocument)
}

func TestOpenAPIExecutionDoesNotExposeTypedNilAsyncAPIContext(t *testing.T) {
	var sawAsyncAPI atomic.Bool
	rule := &model.Rule{
		Id:       "capture-asyncapi-context",
		Given:    "$",
		Severity: model.SeverityError,
		Then: model.RuleAction{
			Function: "captureAsyncAPIContext",
		},
	}

	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}},
		Spec: []byte(`openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`),
		CustomFunctions: map[string]model.RuleFunction{
			"captureAsyncAPIContext": captureAsyncAPIContextFunction{&sawAsyncAPI},
		},
	})

	require.Empty(t, result.Errors)
	assert.False(t, sawAsyncAPI.Load())
}

func TestAsyncAPI2ExecutionReturnsInputError(t *testing.T) {
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: rulesets.BuildDefaultRuleSets().GenerateOpenAPIRecommendedRuleSet(),
		Spec: []byte(`asyncapi: 2.6.0
info:
  title: Legacy
  version: 1.0.0
channels: {}
`),
	})

	require.NotEmpty(t, result.Errors)
	assert.True(t, errors.Is(result.Errors[0], libasyncapi.ErrAsyncAPI2NotSupported))
}

func TestMalformedAsyncAPIDoesNotEnterOpenAPIPath(t *testing.T) {
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: rulesets.BuildDefaultRuleSets().GenerateOpenAPIRecommendedRuleSet(),
		Spec: []byte(`asyncapi: 3.1.0
info:
  title: [
`),
	})

	require.NotEmpty(t, result.Errors)
	assert.Nil(t, result.RuleSetExecution.Document)
	assert.Nil(t, result.RuleSetExecution.DrDocument)
}

func TestAsyncAPIDocumentErrorsAreEmittedWithoutDocumentRule(t *testing.T) {
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{}},
		Spec: []byte(`asyncapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
channels:
  testChannel:
    address: test/channel
    messages:
      - $ref: '#/components/messages/NonExistent'
components:
  messages: {}
`),
	})

	require.Empty(t, result.Errors)
	require.NotEmpty(t, result.Results)
	assert.Equal(t, rulesets.AsyncAPI3DocumentResolved, result.Results[0].RuleId)
}

func TestAsyncAPIPatternRulesFire(t *testing.T) {
	allRules := rulesets.GetAllAsyncAPIRules()
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{
		rulesets.AsyncAPI3ChannelNoEmptyParameter:   allRules[rulesets.AsyncAPI3ChannelNoEmptyParameter],
		rulesets.AsyncAPI3ChannelNoQueryNorFragment: allRules[rulesets.AsyncAPI3ChannelNoQueryNorFragment],
		rulesets.AsyncAPI3ChannelNoTrailingSlash:    allRules[rulesets.AsyncAPI3ChannelNoTrailingSlash],
		rulesets.AsyncAPI3ServerNoEmptyVariable:     allRules[rulesets.AsyncAPI3ServerNoEmptyVariable],
		rulesets.AsyncAPI3ServerNoTrailingSlash:     allRules[rulesets.AsyncAPI3ServerNoTrailingSlash],
		rulesets.AsyncAPI3ServerNotExampleCom:       allRules[rulesets.AsyncAPI3ServerNotExampleCom],
	}}

	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: ruleSet,
		Spec: []byte(`asyncapi: 3.1.0
info:
  title: Events
  version: 1.0.0
servers:
  production:
    host: example.com
    pathname: /events/{}/
    protocol: mqtt
channels:
  invalid:
    address: events/{}?debug=true/
`),
	})

	require.Empty(t, result.Errors)
	ruleIDs := asyncAPIRuleIDs(result.Results)
	assert.Contains(t, ruleIDs, rulesets.AsyncAPI3ChannelNoEmptyParameter)
	assert.Contains(t, ruleIDs, rulesets.AsyncAPI3ChannelNoQueryNorFragment)
	assert.Contains(t, ruleIDs, rulesets.AsyncAPI3ChannelNoTrailingSlash)
	assert.Contains(t, ruleIDs, rulesets.AsyncAPI3ServerNoEmptyVariable)
	assert.Contains(t, ruleIDs, rulesets.AsyncAPI3ServerNoTrailingSlash)
	assert.Contains(t, ruleIDs, rulesets.AsyncAPI3ServerNotExampleCom)
}

func asyncAPIRuleIDs(results []model.RuleFunctionResult) map[string]bool {
	ruleIDs := make(map[string]bool, len(results))
	for _, result := range results {
		ruleIDs[result.RuleId] = true
	}
	return ruleIDs
}

const validAsyncAPI31Fixture = `asyncapi: 3.1.0
info:
  title: Events
  version: 1.0.0
  description: Event contract.
  contact:
    name: API Team
    url: https://example.com
    email: api@example.com
  license:
    name: MIT
defaultContentType: application/json
tags:
  - name: events
    description: Event APIs.
servers:
  production:
    host: api.example.com
    protocol: mqtt
channels:
  userSignedUp:
    address: user/signedup/{userId}
    parameters:
      userId:
        description: User identifier.
operations:
  receiveUserSignedUp:
    action: receive
    description: Receive user signup events.
    channel:
      $ref: '#/channels/userSignedUp'
    messages:
      - $ref: '#/components/messages/UserSignedUp'
components:
  messages:
    UserSignedUp:
      payload:
        type: object
        properties:
          id:
            type: string
      examples:
        - payload:
            id: abc
`
