// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	lowasync "github.com/pb33f/libasyncapi/datamodel/low/asyncapi"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

type testAsyncAPIContext struct {
	root      *yaml.Node
	pathIndex *vacuumUtils.NodePathIndex
}

func (c *testAsyncAPIContext) Root() *yaml.Node {
	return c.root
}

func (c *testAsyncAPIContext) DocumentErrors() []error {
	return nil
}

func (c *testAsyncAPIContext) NodePath(node *yaml.Node) (string, bool) {
	if c.pathIndex == nil {
		c.pathIndex = vacuumUtils.BuildNodePathIndex(c.root)
	}
	return c.pathIndex.Lookup(node)
}

func testRuleContext(t *testing.T, spec string) (model.RuleFunctionContext, *yaml.Node) {
	t.Helper()
	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &doc))
	require.Len(t, doc.Content, 1)
	root := doc.Content[0]
	built, _ := lowasync.CreateDocument(&doc)
	return model.RuleFunctionContext{
		Index:    built.Index,
		Rule:     &model.Rule{Id: "test-asyncapi-rule"},
		AsyncAPI: &testAsyncAPIContext{root: &doc},
	}, root
}

func TestChannelParametersReportsMissingAndUnusedVariablesDeterministically(t *testing.T) {
	context, root := testRuleContext(t, `asyncapi: 3.1.0
channels:
  userChannel:
    address: users/{action}/{userId}
    parameters:
      unused: {}
      userId: {}
`)
	_, channels := mappingValue(root, "channels")
	_, channel := mappingValue(channels, "userChannel")

	first := ChannelParameters{}.RunRule([]*yaml.Node{channel}, context)
	second := ChannelParameters{}.RunRule([]*yaml.Node{channel}, context)

	require.Len(t, first, 2)
	require.Len(t, second, 2)
	assert.Equal(t, resultMessages(first), resultMessages(second))
	assert.Equal(t, "Channel address variable `action` is used but not defined.", first[0].Message)
	assert.Equal(t, "Channel address variable `unused` is defined but not used.", first[1].Message)
	assert.NotEmpty(t, first[0].Path)
	assert.NotEmpty(t, first[1].Path)
}

func TestSecurityReportsUnknownAndInvalidReferences(t *testing.T) {
	context, root := testRuleContext(t, `asyncapi: 3.1.0
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows: {}
security:
  - $ref: '#/components/securitySchemes/oauth'
  - $ref: '#/components/securitySchemes/missing'
  - $ref: '#/servers/production'
`)
	_, security := mappingValue(root, "security")

	results := Security{}.RunRule(security.Content, context)

	require.Len(t, results, 2)
	assert.Equal(t, "Security scheme `#/components/securitySchemes/missing` is not defined.", results[0].Message)
	assert.Equal(t, "$.security[1]", results[0].Path)
	assert.Equal(t, "Security scheme `#/servers/production` is not defined.", results[1].Message)
	assert.Equal(t, "$.security[2]", results[1].Path)
}

func TestSecurityAcceptsInlineSecuritySchemeObjects(t *testing.T) {
	context, root := testRuleContext(t, `asyncapi: 3.1.0
security:
  - type: scramSha256
    description: Provide your username and password for SASL/SCRAM authentication
  - type: oauth2
    description: Inline OAuth2 scheme.
    flows:
      clientCredentials:
        tokenUrl: https://auth.example.com/token
        availableScopes:
          read: Read events.
    scopes:
      - read
`)
	_, security := mappingValue(root, "security")

	assert.Empty(t, Security{}.RunRule(security.Content, context))
}

func TestSecurityReportsEntriesThatAreNotSecuritySchemes(t *testing.T) {
	context, root := testRuleContext(t, `asyncapi: 3.1.0
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows: {}
security:
  - oauth: []
  - description: A scheme without a type.
  - {}
  - oauth
  - type: userPassword
`)
	_, security := mappingValue(root, "security")

	results := Security{}.RunRule(security.Content, context)

	require.Len(t, results, 4)
	for i, finding := range results {
		assert.Equal(t, "Security scheme must define a supported string `type`.", finding.Message)
		assert.Equal(t, fmt.Sprintf("$.security[%d]", i), finding.Path)
	}
}

const operationMessagesFixture = `asyncapi: 3.0.0
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
        $ref: '#/components/messages/turnOnOff'
      turn/on~now:
        $ref: '#/components/messages/turnOnOff'
  dimLight:
    $ref: '#/components/channels/dimLight'
  external:
    $ref: './channels.yaml#/external'
  circular:
    $ref: '#/channels/circular'
components:
  channels:
    dimLight:
      address: lighting/dim
      messages:
        dim:
          $ref: '#/components/messages/turnOnOff'
  messages:
    lightMeasured:
      payload:
        type: object
    turnOnOff:
      payload:
        type: object
`

func TestOperationMessagesAcceptsMessagesOfTheOperationChannel(t *testing.T) {
	context, root := testRuleContext(t, operationMessagesFixture+`operations:
  receiveLightMeasurement:
    action: receive
    channel:
      $ref: '#/channels/lightingMeasured'
    messages:
      - $ref: '#/channels/lightingMeasured/messages/lightMeasured'
  turnOn:
    action: send
    channel:
      $ref: '#/channels/lightTurnOn'
    messages:
      - $ref: '#/channels/lightTurnOn/messages/turnOn'
      - $ref: '#/channels/lightTurnOn/messages/turn~1on~0now'
      - $ref: '#/channels/lightTurnOn/messages/turn%7E1on~0now'
  dimViaRootChannelReference:
    action: send
    channel:
      $ref: '#/channels/dimLight'
    messages:
      - $ref: '#/channels/dimLight/messages/dim'
  dimViaComponentChannel:
    action: send
    channel:
      $ref: '#/components/channels/dimLight'
    messages:
      - $ref: '#/components/channels/dimLight/messages/dim'
`)

	assert.Empty(t, OperationMessages{}.RunRule(operationNodes(root), context))
}

func TestOperationMessagesReportsMessagesOutsideTheOperationChannel(t *testing.T) {
	context, root := testRuleContext(t, operationMessagesFixture+`operations:
  receiveLightMeasurement:
    action: receive
    channel:
      $ref: '#/channels/lightingMeasured'
    messages:
      - $ref: '#/components/messages/lightMeasured'
      - $ref: '#/channels/lightTurnOn/messages/turnOn'
      - $ref: '#/channels/lightingMeasured/messages/missing'
      - $ref: '#/channels/lightingMeasured/messages/lightMeasured/payload'
      - $ref: '#/channels/lightingMeasured/messages/'
      - payload:
          type: object
  dimViaRootChannelReference:
    action: send
    channel:
      $ref: '#/channels/dimLight'
    messages:
      - $ref: '#/channels/dimLight/messages/missing'
      - $ref: '#/components/channels/dimLight/messages/dim'
`)

	results := OperationMessages{}.RunRule(operationNodes(root), context)

	assert.Equal(t, []string{
		"Operation message `#/components/messages/lightMeasured` must reference a message of channel `#/channels/lightingMeasured`.",
		"Operation message `#/channels/lightTurnOn/messages/turnOn` must reference a message of channel `#/channels/lightingMeasured`.",
		"Operation message `missing` is not defined on channel `#/channels/lightingMeasured`.",
		"Operation message `#/channels/lightingMeasured/messages/lightMeasured/payload` must reference a message of channel `#/channels/lightingMeasured`.",
		"Operation message `` is not defined on channel `#/channels/lightingMeasured`.",
		"Operation messages must be Reference Objects.",
		"Operation message `missing` is not defined on channel `#/channels/dimLight`.",
		"Operation message `#/components/channels/dimLight/messages/dim` must reference a message of channel `#/channels/dimLight`.",
	}, resultMessages(results))
	assert.Equal(t, "$.operations.receiveLightMeasurement.messages[0]", results[0].Path)
	assert.Equal(t, "$.operations.dimViaRootChannelReference.messages[1]", results[7].Path)
}

func TestOperationMessagesOnlyChecksMessageKeysOfChannelsItCanResolve(t *testing.T) {
	context, root := testRuleContext(t, operationMessagesFixture+`operations:
  undefinedChannel:
    action: send
    channel:
      $ref: '#/channels/undefined'
    messages:
      - $ref: '#/channels/undefined/messages/anything'
  externalRootChannel:
    action: send
    channel:
      $ref: '#/channels/external'
    messages:
      - $ref: '#/channels/external/messages/anything'
  circularChannel:
    action: send
    channel:
      $ref: '#/channels/circular'
    messages:
      - $ref: '#/channels/circular/messages/anything'
  missingChannel:
    action: send
    messages:
      - $ref: '#/components/messages/lightMeasured'
  missingMessages:
    action: send
    channel:
      $ref: '#/channels/lightTurnOn'
`)

	results := OperationMessages{}.RunRule(operationNodes(root), context)

	require.Len(t, results, 1)
	assert.Equal(t, "Operation messages require a channel reference.", results[0].Message)
	assert.Equal(t, "$.operations.missingChannel.messages[0]", results[0].Path)
}

func TestOperationMessagesAllowsComponentOperationsToUseChannelsAnywhere(t *testing.T) {
	context, root := testRuleContext(t, operationMessagesFixture+`  operations:
    sendViaExternalChannel:
      action: send
      channel:
        $ref: './channels.yaml#/userSignedUp'
      messages:
        - $ref: './channels.yaml#/userSignedUp/messages/userSignedUp'
        - $ref: '#/components/messages/lightMeasured'
    sendViaComponentChannel:
      action: send
      channel:
        $ref: '#/components/channels/dimLight'
      messages:
        - $ref: '#/components/channels/dimLight/messages/dim'
        - $ref: '#/channels/dimLight/messages/dim'
`)
	_, components := mappingValue(root, "components")
	_, operationMap := mappingValue(components, "operations")
	operations := mappingValues(operationMap)
	require.Len(t, operations, 2)

	results := OperationMessages{}.RunRule(operations, context)

	assert.Equal(t, []string{
		"Operation message `#/components/messages/lightMeasured` must reference a message of channel `./channels.yaml#/userSignedUp`.",
		"Operation message `#/channels/dimLight/messages/dim` must reference a message of channel `#/components/channels/dimLight`.",
	}, resultMessages(results))
}

func TestOperationReplyAcceptsMessagesOfTheReplyChannel(t *testing.T) {
	context, root := testRuleContext(t, operationMessagesFixture+`operations:
  turnOn:
    action: send
    channel:
      $ref: '#/channels/lightTurnOn'
    messages:
      - $ref: '#/channels/lightTurnOn/messages/turnOn'
    reply:
      channel:
        $ref: '#/channels/lightingMeasured'
      messages:
        - $ref: '#/channels/lightingMeasured/messages/lightMeasured'
  dim:
    action: send
    channel:
      $ref: '#/channels/lightTurnOn'
    messages:
      - $ref: '#/channels/lightTurnOn/messages/turnOn'
    reply:
      channel:
        $ref: '#/channels/dimLight'
      messages:
        - $ref: '#/channels/dimLight/messages/dim'
`)

	assert.Empty(t, OperationReply{}.RunRule(operationNodes(root), context))
}

func TestOperationReplyReportsMessagesOutsideTheReplyChannel(t *testing.T) {
	context, root := testRuleContext(t, operationMessagesFixture+`operations:
  turnOn:
    action: send
    channel:
      $ref: '#/channels/lightTurnOn'
    messages:
      - $ref: '#/channels/lightTurnOn/messages/turnOn'
    reply:
      channel:
        $ref: '#/channels/lightingMeasured'
      messages:
        - $ref: '#/components/messages/lightMeasured'
        - $ref: '#/channels/lightTurnOn/messages/turnOn'
        - $ref: '#/channels/lightingMeasured/messages/missing'
        - payload:
            type: object
  replyWithoutChannel:
    action: send
    channel:
      $ref: '#/channels/lightTurnOn'
    messages:
      - $ref: '#/channels/lightTurnOn/messages/turnOn'
    reply:
      messages:
        - $ref: '#/components/messages/lightMeasured'
`)

	results := OperationReply{}.RunRule(operationNodes(root), context)

	assert.Equal(t, []string{
		"Reply message `#/components/messages/lightMeasured` must reference a message of channel `#/channels/lightingMeasured`.",
		"Reply message `#/channels/lightTurnOn/messages/turnOn` must reference a message of channel `#/channels/lightingMeasured`.",
		"Reply message `missing` is not defined on channel `#/channels/lightingMeasured`.",
		"Reply messages must be Reference Objects.",
		"Reply messages require a channel reference.",
	}, resultMessages(results))
	assert.Equal(t, "$.operations.turnOn.reply.messages[0]", results[0].Path)
}

func operationNodes(root *yaml.Node) []*yaml.Node {
	_, operations := mappingValue(root, "operations")
	return mappingValues(operations)
}

func mappingValues(node *yaml.Node) []*yaml.Node {
	entries := mappingEntries(node)
	values := make([]*yaml.Node, 0, len(entries))
	for _, entry := range entries {
		values = append(values, entry[1])
	}
	return values
}

func TestChannelServersReportsMissingAndInvalidReferences(t *testing.T) {
	context, root := testRuleContext(t, `asyncapi: 3.1.0
servers:
  production:
    host: api.example.com
    protocol: mqtt
channels:
  userChannel:
    servers:
      - $ref: '#/servers/missing'
      - $ref: '#/components/servers/componentMissing'
      - $ref: '#/channels/wrong'
components:
  servers:
    backup:
      host: backup.example.com
      protocol: mqtt
`)
	_, channels := mappingValue(root, "channels")
	_, channel := mappingValue(channels, "userChannel")

	results := ChannelServers{}.RunRule([]*yaml.Node{channel}, context)

	require.Len(t, results, 3)
	assert.Contains(t, resultMessages(results), "Channel server `missing` is not defined.")
	assert.Contains(t, resultMessages(results), "Channel servers must reference `#/servers`.")
}

func TestChannelServersRejectsComponentServerReferences(t *testing.T) {
	context, root := testRuleContext(t, `asyncapi: 3.1.0
channels:
  userChannel:
    servers:
      - $ref: '#/components/servers/backup'
components:
  servers:
    backup:
      host: backup.example.com
      protocol: mqtt
`)
	_, channels := mappingValue(root, "channels")
	_, channel := mappingValue(channels, "userChannel")

	results := ChannelServers{}.RunRule([]*yaml.Node{channel}, context)

	require.Len(t, results, 1)
	assert.Equal(t, "Channel servers must reference `#/servers`.", results[0].Message)
}

func TestUnusedComponentsIgnoresReferencedComponents(t *testing.T) {
	context, root := testRuleContext(t, `asyncapi: 3.1.0
channels:
  userChannel:
    messages:
      - $ref: '#/components/messages/UserCreated'
components:
  messages:
    UserCreated:
      payload:
        type: object
    UserDeleted:
      payload:
        type: object
`)

	results := UnusedComponents{}.RunRule([]*yaml.Node{root}, context)

	require.Len(t, results, 1)
	assert.Equal(t, "Potentially unused AsyncAPI component `#/components/messages/UserDeleted` was detected.", results[0].Message)
}

func resultMessages(results []model.RuleFunctionResult) []string {
	messages := make([]string, 0, len(results))
	for _, result := range results {
		messages = append(messages, result.Message)
	}
	return messages
}
