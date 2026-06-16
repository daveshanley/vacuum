// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
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
	return model.RuleFunctionContext{
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
security:
  - unknown: []
  - $ref: '#/components/securitySchemes/missing'
  - $ref: '#/servers/production'
`)
	_, security := mappingValue(root, "security")

	results := Security{}.RunRule(security.Content, context)

	require.Len(t, results, 3)
	assert.Contains(t, resultMessages(results), "Security scheme `unknown` is not defined.")
	assert.Contains(t, resultMessages(results), "Security scheme `missing` is not defined.")
	assert.Contains(t, resultMessages(results), "Security scheme references must target `#/components/securitySchemes`.")
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
