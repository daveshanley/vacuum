// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestBundleAsyncAPIModesAndStreams(t *testing.T) {
	for _, composed := range []bool{false, true} {
		for _, stdin := range []bool{false, true} {
			for _, stdout := range []bool{false, true} {
				for _, format := range []string{"yaml", "json"} {
					t.Run(fmt.Sprintf("composed=%t/stdin=%t/stdout=%t/%s", composed, stdin, stdout, format), func(t *testing.T) {
						dir := t.TempDir()
						input := filepath.Join(dir, "asyncapi.yaml")
						output := filepath.Join(dir, "bundled."+format)
						spec := "asyncapi: 3.1.0\ninfo: {title: Test, version: '1'}\nchannels:\n  events:\n    messages:\n      event: {$ref: message.yaml}\noperations:\n  send:\n    action: send\n    channel: {$ref: '#/channels/events'}\n    messages: [{$ref: '#/channels/events/messages/event'}]\n"
						writeTestFile(t, input, spec)
						writeTestFile(t, filepath.Join(dir, "message.yaml"), "name: Event\npayload: {type: string}")
						cmd := newBundleTestCommand()
						args := []string{}
						if stdin {
							args = append(args, "--stdin", "--base", dir)
							cmd.SetIn(strings.NewReader(spec))
						} else {
							args = append(args, input)
						}
						if stdout {
							args = append(args, "--stdout")
						} else {
							args = append(args, output)
						}
						if composed {
							args = append(args, "--composed", "--delimiter", "--")
						}
						args = append(args, "--format", format)
						cmd.SetArgs(args)
						var err error
						out, stderr := captureOSStreams(t, func() { err = cmd.Execute() })
						require.NoError(t, err)
						assert.Empty(t, stderr)
						if !stdout {
							out = readOutputFile(t, output)
						}
						var document map[string]any
						require.NoError(t, yaml.Unmarshal([]byte(out), &document))
						assert.Equal(t, "3.1.0", document["asyncapi"])
						assert.NotContains(t, out, "message.yaml")
						assert.Contains(t, out, "#/channels/events/messages/event")
						if format == "json" {
							assert.True(t, strings.HasPrefix(strings.TrimSpace(out), "{"))
						}
						if composed {
							assert.Contains(t, out, "#/components/messages/message")
						}
					})
				}
			}
		}
	}
}

func TestBundleAsyncAPIFailurePreservesOutput(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "asyncapi.yaml"), filepath.Join(dir, "output.yaml")
	writeTestFile(t, input, "asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {events: {$ref: missing.yaml}}")
	writeTestFile(t, output, "keep me")
	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{input, output})
	_, stderr := captureOSStreams(t, func() {
		err := cmd.Execute()
		require.Error(t, err)
		var exit *ExitError
		require.True(t, errors.As(err, &exit))
		assert.Equal(t, 2, exit.Code)
	})
	assert.Contains(t, stderr, "missing.yaml")
	b, err := os.ReadFile(output)
	require.NoError(t, err)
	assert.Equal(t, "keep me\n", string(b))
}
