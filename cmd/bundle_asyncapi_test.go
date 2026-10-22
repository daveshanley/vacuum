// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestBundleStdinLegacyOutputArgument(t *testing.T) {
	for _, spec := range []string{
		"asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {}\n",
		"openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}\n",
	} {
		t.Run(strings.Split(spec, ":")[0], func(t *testing.T) {
			dir := t.TempDir()
			input, output := filepath.Join(dir, "input.yaml"), filepath.Join(dir, "output.json")
			writeTestFile(t, input, "preserve this file")
			cmd := newBundleTestCommand()
			cmd.SetIn(strings.NewReader(spec))
			cmd.SetArgs([]string{"--stdin", input, output})
			_, stderr := captureOSStreams(t, func() { require.NoError(t, cmd.Execute()) })
			assert.Empty(t, stderr)
			assert.Equal(t, "preserve this file\n", readOutputFile(t, input))
			assert.True(t, strings.HasPrefix(strings.TrimSpace(readOutputFile(t, output)), "{"))
		})
	}
}

func TestBundleAsyncAPIBasePreservesRootIdentity(t *testing.T) {
	for _, composed := range []bool{false, true} {
		t.Run(fmt.Sprint(composed), func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "v2", "asyncapi.yaml")
			base := filepath.Join(dir, "common")
			writeTestFile(t, input, "asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {events: {$ref: 'ch.yaml'}}\ncomponents: {messages: {m: {payload: {type: string}}}}\n")
			writeTestFile(t, filepath.Join(base, "ch.yaml"), "messages: {event: {$ref: './asyncapi.yaml#/components/messages/m'}}\n")
			writeTestFile(t, filepath.Join(base, "asyncapi.yaml"), "components: {messages: {m: {payload: {type: integer}}}}\n")
			cmd := newBundleTestCommand()
			cmd.SetArgs([]string{input, "--stdout", "--base", base, fmt.Sprintf("--composed=%t", composed)})
			out, stderr := captureOSStreams(t, func() { require.NoError(t, cmd.Execute()) })
			assert.Empty(t, stderr)
			assert.Contains(t, out, "type: integer")
		})
	}
}

func TestBundleAsyncAPIURLBase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/specs/message.yaml" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintln(w, "name: RemoteMessage\npayload: {type: integer}")
	}))
	defer server.Close()
	spec := "asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {events: {messages: {event: {$ref: message.yaml}}}}\n"
	for _, stdin := range []bool{false, true} {
		for _, remote := range []bool{false, true} {
			t.Run(fmt.Sprintf("stdin=%t/remote=%t", stdin, remote), func(t *testing.T) {
				cmd := newBundleTestCommand()
				args := []string{"--stdout", "--base", server.URL + "/specs", fmt.Sprintf("--remote=%t", remote)}
				if stdin {
					args = append(args, "--stdin")
					cmd.SetIn(strings.NewReader(spec))
				} else {
					input := filepath.Join(t.TempDir(), "asyncapi.yaml")
					writeTestFile(t, input, spec)
					args = append(args, input)
				}
				cmd.SetArgs(args)
				var err error
				out, stderr := captureOSStreams(t, func() { err = cmd.Execute() })
				if !remote {
					require.Error(t, err)
					assert.Empty(t, out)
					assert.Contains(t, stderr, "message.yaml")
					return
				}
				require.NoError(t, err)
				assert.Empty(t, stderr)
				assert.Contains(t, out, "RemoteMessage")
			})
		}
	}
}

func TestBundleAsyncAPIStdinUsesWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)
	writeTestFile(t, filepath.Join(dir, "message.yaml"), "name: LocalMessage\npayload: {type: string}")
	cmd := newBundleTestCommand()
	cmd.SetIn(strings.NewReader("asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {events: {messages: {event: {$ref: message.yaml}}}}"))
	cmd.SetArgs([]string{"--stdin", "--stdout"})
	out, stderr := captureOSStreams(t, func() { require.NoError(t, cmd.Execute()) })
	assert.Empty(t, stderr)
	assert.Contains(t, out, "LocalMessage")
}

func TestBundleErrorsPrintOnce(t *testing.T) {
	cmd := newBundleTestCommand()
	var cobraErrors bytes.Buffer
	cmd.SetErr(&cobraErrors)
	cmd.SetArgs([]string{"--stdin"})
	_, stderr := captureOSStreams(t, func() { require.Error(t, cmd.Execute()) })
	assert.Equal(t, 1, strings.Count(stderr+cobraErrors.String(), "please supply an output API document"))
}

func TestBundleAsyncAPIFailurePrintsLoaderLogs(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	cmd := newBundleTestCommand()
	cmd.SetIn(strings.NewReader("asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {events: {$ref: missing.yaml}}"))
	cmd.SetArgs([]string{"--stdin", "--stdout", "--no-style", "--base", server.URL})
	var cobraErrors bytes.Buffer
	cmd.SetErr(&cobraErrors)
	out, stderr := captureOSStreams(t, func() { require.Error(t, cmd.Execute()) })
	assert.Empty(t, out)
	assert.Empty(t, cobraErrors.String())
	assert.Contains(t, stderr, "ERR unable to fetch remote document")
	assert.Contains(t, stderr, "404")
}

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

func TestAsyncAPIBundleOutputLintsWithoutSourceFiles(t *testing.T) {
	for _, composed := range []bool{false, true} {
		for _, format := range []string{"yaml", "json"} {
			t.Run(fmt.Sprintf("composed=%t/%s", composed, format), func(t *testing.T) {
				dir := t.TempDir()
				input := filepath.Join(dir, "asyncapi.json")
				writeTestFile(t, input, `{"asyncapi":"3.1.0","info":{"title":"Events \ud83c\udf03","version":"1"},"channels":{"events":{"$ref":"#/components/channels/events"}},"operations":{"send":{"action":"send","channel":{"$ref":"#/channels/events"},"messages":[{"$ref":"#/channels/events/messages/event"}],"security":[{"type":"scramSha256"}],"reply":{"$ref":"#/components/replies/reply"}}},"components":{"channels":{"events":{"$ref":"channel.yaml"}},"replies":{"reply":{"channel":{"$ref":"#/channels/events"},"messages":[{"$ref":"#/channels/events/messages/event"}]}}}}`)
				writeTestFile(t, filepath.Join(dir, "channel.yaml"), "address: events\nmessages: {event: {$ref: message.yaml}}\n")
				writeTestFile(t, filepath.Join(dir, "message.yaml"), "name: Event\npayload: {type: string}\n")
				cmd := newBundleTestCommand()
				args := []string{input, "--stdout", "--format", format}
				if composed {
					args = append(args, "--composed")
				}
				cmd.SetArgs(args)
				var bundleErr error
				output, stderr := captureOSStreams(t, func() { bundleErr = cmd.Execute() })
				require.NoError(t, bundleErr)
				assert.Empty(t, stderr)
				require.NoError(t, os.RemoveAll(dir))
				all := rulesets.GetAllAsyncAPIRules()
				selected := &rulesets.RuleSet{Rules: make(map[string]*model.Rule)}
				for _, id := range []string{rulesets.AsyncAPI3DocumentResolved, rulesets.AsyncAPIOperationChannel, rulesets.AsyncAPIOperationMessages, rulesets.AsyncAPIOperationReply, rulesets.AsyncAPI3OperationSecurity} {
					selected.Rules[id] = all[id]
				}
				lint := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{RuleSet: selected, Spec: []byte(output)})
				require.Empty(t, lint.Errors)
				assert.Empty(t, lint.Results)
				invalid := strings.ReplaceAll(output, "scramSha256", "bogus")
				lint = motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{RuleSet: selected, Spec: []byte(invalid)})
				require.Empty(t, lint.Errors)
				require.Len(t, lint.Results, 1)
				assert.Equal(t, rulesets.AsyncAPI3OperationSecurity, lint.Results[0].RuleId)
			})
		}
	}
}
