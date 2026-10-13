package asyncapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestOptionalOperationAndReplyFields(t *testing.T) {
	for _, version := range []string{"3.0.0", "3.1.0"} {
		for _, fields := range []string{"", "messages: []", "reply: {}", "reply: {messages: []}", "reply: {address: {$ref: '#/components/replyAddresses/address'}}", "reply: {$ref: '#/components/replies/reply'}"} {
			ctx, root := testRuleContext(t, fmt.Sprintf("asyncapi: %s\nchannels: {c: {messages: {m: {name: event}}}}\noperations:\n  send:\n    action: send\n    channel: {$ref: '#/channels/c'}\n    %s\ncomponents:\n  replies: {reply: {}}\n  replyAddresses: {address: {location: '$message.header#/replyTo'}}\n", version, fields))
			assert.Empty(t, OperationMessages{}.RunRule(operationNodes(root), ctx), fields)
			assert.Empty(t, OperationReply{}.RunRule(operationNodes(root), ctx), fields)
		}
	}
}

func TestOperationChannelDefinitionLocations(t *testing.T) {
	ctx, root := testRuleContext(t, `asyncapi: 3.1.0
channels: {c: {messages: {m: {name: event}}}}
operations:
  invalid: {action: send, channel: {$ref: '#/components/channels/c'}}
  valid: {$ref: '#/components/operations/send'}
components:
  channels: {c: {messages: {m: {name: event}}}}
  operations:
    send: {action: send, channel: {$ref: '#/components/channels/c'}}
`)
	results := OperationChannel{}.RunRule(operationNodes(root), ctx)
	require.Len(t, results, 1)
	assert.Equal(t, "$.operations.invalid.channel", results[0].Path)
	assert.Equal(t, 4, results[0].StartNode.Line)
}

func TestOperationReferenceChainsAndEncodedPointers(t *testing.T) {
	spec := "asyncapi: 3.1.0\nchannels: {'a/b': {$ref: '#/components/channels/c0'}}\noperations: {send: {action: send, channel: {$ref: '#%2Fchannels%2Fa~1b'}, messages: [{$ref: '#/channels/a~1b/messages/m'}]}}\ncomponents:\n  channels:\n"
	for i := 0; i < 10; i++ {
		spec += fmt.Sprintf("    c%d: {$ref: '#/components/channels/c%d'}\n", i, i+1)
	}
	spec += "    c10: {messages: {m: {name: event}}}\n"
	ctx, root := testRuleContext(t, spec)
	assert.Empty(t, OperationChannel{}.RunRule(operationNodes(root), ctx))
	assert.Empty(t, OperationMessages{}.RunRule(operationNodes(root), ctx))
}

func TestMalformedOperationMessages(t *testing.T) {
	for _, messages := range []string{"null", "{}", "true", "[{name: inline}]", "[{$ref: 42}]", "[{$ref: ''}]"} {
		ctx, root := testRuleContext(t, "asyncapi: 3.1.0\nchannels: {c: {}}\noperations: {send: {action: send, channel: {$ref: '#/channels/c'}, messages: "+messages+"}}")
		results := OperationMessages{}.RunRule(operationNodes(root), ctx)
		require.Len(t, results, 1, messages)
		assert.True(t, strings.HasPrefix(results[0].Path, "$.operations.send.messages"))
	}
}

func TestReplyAddressAndChannelConstraints(t *testing.T) {
	for _, reply := range []string{"null", "{channel: {$ref: '#/components/channels/c'}}", "{address: {}}", "{address: {location: 42}}", "{channel: {$ref: '#/channels/c'}, address: {location: '$message.header#/replyTo'}}"} {
		ctx, root := testRuleContext(t, "asyncapi: 3.1.0\nchannels: {c: {address: fixed}}\ncomponents: {channels: {c: {}}}\noperations: {send: {action: send, channel: {$ref: '#/channels/c'}, reply: "+reply+"}}")
		require.Len(t, OperationReply{}.RunRule(operationNodes(root), ctx), 1, reply)
	}
}

func TestSecuritySchemeTypesAndRequiredFields(t *testing.T) {
	valid := []string{"type: userPassword", "type: X509", "type: symmetricEncryption", "type: asymmetricEncryption", "type: plain", "type: scramSha256", "type: scramSha512", "type: gssapi", "type: apiKey, in: user", "type: httpApiKey, in: header, name: Authorization", "type: http, scheme: bearer", "type: oauth2, flows: {}", "type: openIdConnect, openIdConnectUrl: https://auth.test/discovery"}
	invalid := []string{"type: bogus", "type: 42", "type: apiKey", "type: apiKey, in: header", "type: httpApiKey, in: user, name: key", "type: httpApiKey, in: header", "type: http", "type: oauth2, flows: []", "type: openIdConnect, openIdConnectUrl: /discovery"}
	for _, good := range []bool{false, true} {
		cases := invalid
		if good {
			cases = valid
		}
		for _, fields := range cases {
			ctx, root := testRuleContext(t, "asyncapi: 3.1.0\nx-scheme: {"+fields+"}\nsecurity: [{$ref: '#/x-scheme'}]\n")
			_, security := mappingValue(root, "security")
			results := Security{}.RunRule(security.Content, ctx)
			if good {
				assert.Empty(t, results, fields)
			} else {
				assert.NotEmpty(t, results, fields)
			}
		}
	}
}

func TestReferencedReplyRetainsDefinitionScope(t *testing.T) {
	ctx, root := testRuleContext(t, `asyncapi: 3.1.0
channels: {c: {}}
operations:
  send: {action: send, channel: {$ref: '#/channels/c'}, reply: {$ref: '#/components/replies/shared'}}
components:
  channels: {c: {messages: {m: {name: event}}}}
  replies:
    shared: {channel: {$ref: '#/components/channels/c'}, messages: [{$ref: '#/components/channels/c/messages/m'}]}
`)
	assert.Empty(t, OperationReply{}.RunRule(operationNodes(root), ctx))
	_, replies := mappingValue(componentMap(root, "replies"), "shared")
	assert.Empty(t, OperationReply{}.RunRule([]*yaml.Node{replies}, ctx))
}
