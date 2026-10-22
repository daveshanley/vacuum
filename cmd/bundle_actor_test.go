package cmd

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/pb33f/libasyncapi"
	"github.com/pb33f/libasyncapi/bundler"
	"github.com/pb33f/testify/require"
)

func TestBundleAsyncAPIIssue1002OperationsOnlyActor(t *testing.T) {
	for _, composed := range []bool{false, true} {
		t.Run(fmt.Sprint(composed), func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "actors", "actor.yaml")
			writeTestFile(t, input, `asyncapi: 3.1.0
info: {title: Actor, version: '1'}
operations:
  receive:
    action: receive
    channel: {$ref: '../domain.yaml#/channels/events'}
    messages: [{$ref: '../domain.yaml#/channels/events/messages/created'}]
    traits: [{$ref: '../domain.yaml#/components/operationTraits/reader'}]
`)
			writeTestFile(t, filepath.Join(dir, "domain.yaml"), `asyncapi: 3.1.0
info: {title: Domain, version: '1'}
servers:
  broker: {host: broker.example.com, protocol: kafka}
channels:
  events:
    address: events
    servers: [{$ref: '#/servers/broker'}]
    messages:
      created: {$ref: '#/components/messages/created'}
  unused: {address: unused}
components:
  operationTraits:
    reader: {summary: Receive events}
  messages:
    created:
      name: Created
      payload: {$ref: './payload.yaml#/Event'}
    unused: {name: Unused}
`)
			writeTestFile(t, filepath.Join(dir, "payload.yaml"), "Event: {type: object, properties: {id: {type: string}}}")
			cmd := newBundleTestCommand()
			cmd.SetArgs([]string{input, "--stdout", fmt.Sprintf("--composed=%t", composed)})
			out, stderr, err := captureProcessStreams(t, cmd.Execute)
			require.NoError(t, err)
			require.Empty(t, stderr)
			require.Contains(t, out, "#/channels/events")
			require.Contains(t, out, "#/servers/broker")
			require.NotContains(t, out, "unused")
			require.NotContains(t, out, ".yaml")
			config := &libasyncapi.DocumentConfiguration{AllowFileReferences: false, AllowRemoteReferences: false}
			doc, err := libasyncapi.NewDocumentWithConfiguration([]byte(out), config)
			require.NoError(t, err)
			require.False(t, doc.IsPartial())
			require.Empty(t, doc.Errors())
			model := doc.Model()
			require.NotNil(t, model)
			require.Equal(t, 1, model.Channels.Len())
			require.Equal(t, 1, model.Servers.Len())
			op := model.Operations.GetOrZero("receive")
			require.NotNil(t, op)
			require.Equal(t, "receive", op.Action)
			require.Len(t, op.Messages, 1)
			require.Equal(t, "#/channels/events/messages/created", op.Messages[0].GetReference())
			// Rebundling with all external access disabled proves output portability.
			again, err := bundler.BundleBytes([]byte(out), config)
			require.NoError(t, err)
			require.NotEmpty(t, again)
		})
	}
}
