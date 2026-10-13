// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libasyncapi"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
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

func TestAsyncAPIValidFixtureHasNoRecommendedRuleViolations(t *testing.T) {
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: rulesets.BuildDefaultRuleSets().GenerateAsyncAPIRecommendedRuleSet(),
		Spec:    []byte(validAsyncAPI31Fixture),
	})

	require.Empty(t, result.Errors)
	assert.Empty(t, result.Results)
}

func TestAsyncAPIChannelMessagesAndInlineSecuritySchemesAreValid(t *testing.T) {
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: asyncAPIReferenceRuleSet(),
		Spec: []byte(`asyncapi: 3.0.0
info:
  title: Streetlights
  version: 1.0.0
servers:
  scram-connections:
    host: test.mykafkacluster.org:18092
    protocol: kafka-secure
    security:
      - $ref: '#/components/securitySchemes/saslScram'
      - type: scramSha256
        description: Provide your username and password for SASL/SCRAM authentication
channels:
  lightingMeasured:
    address: lighting/measured
    messages:
      lightMeasured:
        $ref: '#/components/messages/lightMeasured'
  lightTurnOn:
    address: lighting/on
    messages:
      turnOn:
        $ref: '#/components/messages/lightMeasured'
operations:
  receiveLightMeasurement:
    action: receive
    channel:
      $ref: '#/channels/lightingMeasured'
    security:
      - type: scramSha256
        description: Provide your username and password for SASL/SCRAM authentication
    messages:
      - $ref: '#/channels/lightingMeasured/messages/lightMeasured'
    reply:
      channel:
        $ref: '#/channels/lightTurnOn'
      messages:
        - $ref: '#/channels/lightTurnOn/messages/turnOn'
components:
  messages:
    lightMeasured:
      payload:
        type: object
  securitySchemes:
    saslScram:
      type: scramSha256
      description: Provide your username and password for SASL/SCRAM authentication
`),
	})

	require.Empty(t, result.Errors)
	assert.Empty(t, result.Results)
}

func TestAsyncAPIMessagesOutsideChannelAndInvalidSecurityAreReported(t *testing.T) {
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: asyncAPIReferenceRuleSet(),
		Spec: []byte(`asyncapi: 3.0.0
info:
  title: Streetlights
  version: 1.0.0
servers:
  scram-connections:
    host: test.mykafkacluster.org:18092
    protocol: kafka-secure
    security:
      - saslScram: []
channels:
  lightingMeasured:
    address: lighting/measured
    messages:
      lightMeasured:
        $ref: '#/components/messages/lightMeasured'
  lightTurnOn:
    address: lighting/on
    messages:
      turnOn:
        $ref: '#/components/messages/lightMeasured'
operations:
  receiveLightMeasurement:
    action: receive
    channel:
      $ref: '#/channels/lightingMeasured'
    security:
      - description: A scheme without a type.
    messages:
      - $ref: '#/components/messages/lightMeasured'
    reply:
      channel:
        $ref: '#/channels/lightTurnOn'
      messages:
        - $ref: '#/channels/lightingMeasured/messages/lightMeasured'
components:
  messages:
    lightMeasured:
      payload:
        type: object
  securitySchemes:
    saslScram:
      type: scramSha256
`),
	})

	require.Empty(t, result.Errors)
	messages := make(map[string]string, len(result.Results))
	for _, finding := range result.Results {
		messages[finding.RuleId] = finding.Message
	}
	assert.Equal(t, map[string]string{
		rulesets.AsyncAPIServerSecurity:     "Security scheme must define a supported string `type`.",
		rulesets.AsyncAPI3OperationSecurity: "Security scheme must define a supported string `type`.",
		rulesets.AsyncAPIOperationMessages:  "Operation message `#/components/messages/lightMeasured` must reference a message of channel `#/channels/lightingMeasured`.",
		rulesets.AsyncAPIOperationReply:     "Reply message `#/channels/lightingMeasured/messages/lightMeasured` must reference a message of channel `#/channels/lightTurnOn`.",
	}, messages)
	assert.Len(t, result.Results, 4)
}

func asyncAPIReferenceRuleSet() *rulesets.RuleSet {
	allRules := rulesets.GetAllAsyncAPIRules()
	return &rulesets.RuleSet{Rules: map[string]*model.Rule{
		rulesets.AsyncAPI3DocumentResolved:  allRules[rulesets.AsyncAPI3DocumentResolved],
		rulesets.AsyncAPIServerSecurity:     allRules[rulesets.AsyncAPIServerSecurity],
		rulesets.AsyncAPI3OperationSecurity: allRules[rulesets.AsyncAPI3OperationSecurity],
		rulesets.AsyncAPIOperationChannel:   allRules[rulesets.AsyncAPIOperationChannel],
		rulesets.AsyncAPIOperationMessages:  allRules[rulesets.AsyncAPIOperationMessages],
		rulesets.AsyncAPIOperationReply:     allRules[rulesets.AsyncAPIOperationReply],
	}}
}

func TestAsyncAPISecurityOwnersAndExternalLocations(t *testing.T) {
	for _, source := range []string{
		"operations: {send: {action: send, channel: {$ref: '#/channels/c'}, traits: [{security: [{type: bogus}]}]}}",
		"operations: {send: {$ref: 'remote.yaml#/operations/send'}}",
		"servers: {production: {$ref: 'remote.yaml#/servers/production'}}",
		"components: {securitySchemes: {unused: {type: bogus}}}",
	} {
		base := t.TempDir()
		result := ApplyRulesToRuleSet(&RuleSetExecution{
			RuleSet: asyncAPIReferenceRuleSet(),
			Spec:    []byte("asyncapi: 3.1.0\nchannels: {c: {}}\n" + source),
			Base:    base, SpecFileName: filepath.Join(base, "entry.yaml"), AllowLookup: true,
			RolodexFS: fstest.MapFS{"remote.yaml": &fstest.MapFile{Data: []byte("channels: {c: {}}\noperations:\n  send:\n    action: send\n    channel: {$ref: '#/channels/c'}\n    security: [{type: bogus}]\nservers:\n  production: {host: localhost, protocol: kafka, security: [{type: bogus}]}\n")}},
		})
		require.Empty(t, result.Errors)
		require.Len(t, result.Results, 1, source)
		assert.Contains(t, result.Results[0].Message, "supported string `type`")
		assert.NotEqual(t, "$", result.Results[0].Path)
		assert.Positive(t, result.Results[0].StartNode.Line)
	}
}

func TestAsyncAPISelfQualifiedReferences(t *testing.T) {
	for _, channel := range []string{"#/channels/c", "entry.yaml#/channels/c"} {
		base := t.TempDir()
		spec := "asyncapi: 3.1.0\nchannels: {c: {messages: {m: {name: event}}}}\noperations: {send: {action: send, channel: {$ref: '" + channel + "'}, messages: [{$ref: './entry.yaml#/channels/c/messages/m'}]}}\n"
		result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: asyncAPIReferenceRuleSet(), Spec: []byte(spec), Base: base, SpecFileName: filepath.Join(base, "entry.yaml"), AllowLookup: true, RolodexFS: fstest.MapFS{"entry.yaml": &fstest.MapFile{Data: []byte(spec)}}})
		require.Empty(t, result.Errors)
		assert.Empty(t, result.Results)
	}
}

func TestAsyncAPIExternalOperationErrorPathsAndIgnores(t *testing.T) {
	for _, tc := range []struct{ name, operation, ruleID, path string }{
		{"missing channel", "action: send", rulesets.AsyncAPIOperationChannel, "$.operations.send"},
		{"invalid reply", "action: send, channel: {$ref: '#/channels/c'}, reply: []", rulesets.AsyncAPIOperationReply, "$.operations.send.reply"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			spec := []byte("asyncapi: 3.1.0\nchannels: {c: {}}\noperations: {send: {$ref: 'remote.yaml#/operations/send'}}\n")
			outcome := ApplyRulesToRuleSet(&RuleSetExecution{
				RuleSet: asyncAPIReferenceRuleSet(), Spec: spec,
				Base: base, SpecFileName: filepath.Join(base, "entry.yaml"), AllowLookup: true,
				RolodexFS: fstest.MapFS{"remote.yaml": &fstest.MapFile{Data: []byte("channels: {c: {}}\noperations:\n  send: {" + tc.operation + "}\n")}},
			})
			require.Empty(t, outcome.Errors)
			require.Len(t, outcome.Results, 1)
			finding := &outcome.Results[0]
			assert.Equal(t, tc.ruleID, finding.RuleId)
			assert.Equal(t, tc.path, finding.Path)
			require.NotNil(t, finding.Origin)
			assert.Equal(t, filepath.Join(base, "remote.yaml"), finding.Origin.AbsoluteLocation)
			assert.Equal(t, 3, finding.StartNode.Line)
			matcher := utils.NewIgnoreMatcher(model.IgnoredItems{tc.ruleID: {tc.path}}, utils.IgnoreMatcherOptions{SpecBytes: spec})
			assert.True(t, matcher.Matches(finding))
		})
	}
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
  tags:
    - name: events
      description: Event APIs.
defaultContentType: application/json
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
    messages:
      UserSignedUp:
        $ref: '#/components/messages/UserSignedUp'
operations:
  receiveUserSignedUp:
    action: receive
    description: Receive user signup events.
    channel:
      $ref: '#/channels/userSignedUp'
    messages:
      - $ref: '#/channels/userSignedUp/messages/UserSignedUp'
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

func TestAsyncAPISchemaRulesExecuteWithoutOptionErrors(t *testing.T) {
	for _, valid := range []bool{true, false} {
		headerType := "object"
		servers := "{production: {host: api.example.com, protocol: mqtt}}"
		if !valid {
			headerType, servers = "string", "{}"
		}
		spec := "asyncapi: 3.1.0\ninfo: {title: Events, version: '1'}\nservers: " + servers + "\nchannels: {}\noperations: {}\ncomponents:\n  messages:\n    event:\n      headers: {type: " + headerType + "}\n"
		all := rulesets.GetAllAsyncAPIRules()
		rs := &rulesets.RuleSet{Rules: map[string]*model.Rule{rulesets.AsyncAPI3HeadersSchemaTypeObject: all[rulesets.AsyncAPI3HeadersSchemaTypeObject], rulesets.AsyncAPIServers: all[rulesets.AsyncAPIServers]}}
		result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(spec), RuleSet: rs})
		require.Empty(t, result.Errors)
		if valid {
			require.Empty(t, result.Results)
		} else {
			require.Len(t, result.Results, 2)
			for _, finding := range result.Results {
				require.NotContains(t, finding.Message, "not a valid property")
			}
		}
	}
}

func TestAsyncAPIVersionAndMissingServersAreChecked(t *testing.T) {
	all := rulesets.GetAllAsyncAPIRules()
	rs := &rulesets.RuleSet{Rules: map[string]*model.Rule{
		rulesets.AsyncAPILatestVersion: all[rulesets.AsyncAPILatestVersion],
		rulesets.AsyncAPIServers:       all[rulesets.AsyncAPIServers],
	}}
	result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte("asyncapi: 3.0.0\ninfo: {title: Events, version: '1'}\nchannels: {}\noperations: {}\n"), RuleSet: rs})
	require.Empty(t, result.Errors)
	ids := make(map[string]bool)
	for _, finding := range result.Results {
		ids[finding.RuleId] = true
	}
	require.True(t, ids[rulesets.AsyncAPILatestVersion])
	require.True(t, ids[rulesets.AsyncAPIServers])
}

func TestAsyncAPITagsUseInfoLocation(t *testing.T) {
	all := rulesets.GetAllAsyncAPIRules()
	selected := &rulesets.RuleSet{Rules: map[string]*model.Rule{}}
	for _, id := range []string{rulesets.AsyncAPI3Tags, rulesets.AsyncAPI3TagsUniqueness, rulesets.AsyncAPI3TagsAlphabetical, rulesets.AsyncAPI3TagDescription} {
		selected.Rules[id] = all[id]
	}
	for _, tc := range []struct{ name, tags, ruleID string }{
		{"valid", "[{name: events, description: Events}]", ""},
		{"duplicate", "[{name: events, description: Events}, {name: events, description: Events}]", rulesets.AsyncAPI3TagsUniqueness},
		{"missing description", "[{name: events}]", rulesets.AsyncAPI3TagDescription},
		{"unordered", "[{name: zebra, description: Z}, {name: alpha, description: A}]", rulesets.AsyncAPI3TagsAlphabetical},
		{"empty", "[]", rulesets.AsyncAPI3Tags},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outcome := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: selected, Spec: []byte("asyncapi: 3.1.0\ninfo: {title: Test, version: '1', tags: " + tc.tags + "}\n")})
			require.Empty(t, outcome.Errors)
			if tc.ruleID == "" {
				assert.Empty(t, outcome.Results)
			} else {
				require.Len(t, outcome.Results, 1)
				assert.Equal(t, tc.ruleID, outcome.Results[0].RuleId)
				assert.Contains(t, outcome.Results[0].Path, "$.info")
			}
		})
	}
}
