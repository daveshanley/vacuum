// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

const languageServerHelperEnvironment = "VACUUM_LANGUAGE_SERVER_HELPER"

func TestLanguageServerCommandSubprocessContract(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=TestLanguageServerCommandHelperProcess")
	command.Env = append(os.Environ(), languageServerHelperEnvironment+"=1")
	stdin, err := command.StdinPipe()
	require.NoError(t, err)
	stdout, err := command.StdoutPipe()
	require.NoError(t, err)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	require.NoError(t, command.Start())

	writer := &processFrameWriter{writer: stdin}
	reader := newProcessFrameReader(stdout)
	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"processId": nil, "rootUri": nil,
			"capabilities": map[string]any{
				"workspace": map[string]any{
					"configuration": true,
					"didChangeConfiguration": map[string]any{
						"dynamicRegistration": true,
					},
				},
			},
			"workspaceFolders": []any{},
		},
	}))
	response := reader.read(t)
	assert.Equal(t, float64(1), response["id"])
	result, ok := response["result"].(map[string]any)
	require.True(t, ok)
	serverInfo, ok := result["serverInfo"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "vacuum", serverInfo["name"])

	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{},
	}))
	registration := reader.read(t)
	assert.Equal(t, "client/registerCapability", registration["method"])
	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0", "id": registration["id"], "result": nil,
	}))

	uri := "file:///tmp/vacuum-process-contract.yaml"
	document := "openapi: 3.1.0\ninfo:\n  title: process\n  version: 1.0.0\npaths: {}\n"
	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": uri, "languageId": "yaml", "version": 1, "text": document,
			},
		},
	}))
	configuration := reader.read(t)
	assert.Equal(t, "workspace/configuration", configuration["method"])
	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0", "id": configuration["id"], "result": []any{nil},
	}))
	diagnostics := reader.read(t)
	assert.Equal(t, "textDocument/publishDiagnostics", diagnostics["method"])
	assert.Equal(t, uri, processMessageParams(t, diagnostics)["uri"])

	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didChange",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri, "version": 2},
			"contentChanges": []any{
				map[string]any{
					"range": map[string]any{
						"start": map[string]any{"line": 2, "character": 9},
						"end":   map[string]any{"line": 2, "character": 16},
					},
					"text": "changed",
				},
			},
		},
	}))
	diagnostics = reader.read(t)
	assert.Equal(t, "textDocument/publishDiagnostics", diagnostics["method"])

	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/codeAction",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"range": map[string]any{
				"start": map[string]any{"line": 0, "character": 0},
				"end":   map[string]any{"line": 0, "character": 1},
			},
			"context": map[string]any{
				"diagnostics": []any{
					map[string]any{
						"range": map[string]any{
							"start": map[string]any{"line": 0, "character": 0},
							"end":   map[string]any{"line": 0, "character": 1},
						},
						"message":         "test",
						"codeDescription": map[string]any{"href": "https://quobix.com/vacuum/rules/test"},
					},
				},
			},
		},
	}))
	codeActions := reader.read(t)
	actions, ok := codeActions["result"].([]any)
	require.True(t, ok)
	require.Len(t, actions, 1)

	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "workspace/executeCommand",
		"params":  map[string]any{"command": "vacuum.noop"},
	}))
	execute := reader.read(t)
	assert.Contains(t, execute, "result")
	assert.Nil(t, execute["result"])

	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0", "id": 5, "method": "vacuum/unknown", "params": map[string]any{},
	}))
	unknown := reader.read(t)
	assert.Equal(t, float64(-32601), processResponseError(t, unknown)["code"])

	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0", "id": 6, "method": "textDocument/completion", "params": []any{},
	}))
	invalidParams := reader.read(t)
	invalidError := processResponseError(t, invalidParams)
	assert.Equal(t, float64(-32602), invalidError["code"])
	assert.Contains(t, invalidError["message"], "json: cannot unmarshal array")

	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didClose",
		"params":  map[string]any{"textDocument": map[string]any{"uri": uri}},
	}))
	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0",
		"id":      7,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	}))
	_ = reader.read(t)

	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0", "id": 8, "method": "shutdown", "params": nil,
	}))
	shutdown := reader.read(t)
	assert.Contains(t, shutdown, "result")
	assert.Nil(t, shutdown["result"])
	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0", "method": "exit", "params": nil,
	}))
	require.NoError(t, stdin.Close())

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- command.Wait()
	}()
	select {
	case waitErr := <-waitCh:
		require.NoError(t, waitErr)
	case <-time.After(20 * time.Second):
		_ = command.Process.Kill()
		t.Fatal("language-server subprocess did not exit")
	}
	assert.Empty(t, stderr.String())
}

func TestLanguageServerCommandExitWithoutShutdownIsNonZero(t *testing.T) {
	command := exec.Command(os.Args[0], "-test.run=TestLanguageServerCommandHelperProcess")
	command.Env = append(os.Environ(), languageServerHelperEnvironment+"=1")
	stdin, err := command.StdinPipe()
	require.NoError(t, err)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	require.NoError(t, command.Start())

	writer := &processFrameWriter{writer: stdin}
	require.NoError(t, writer.write(map[string]any{
		"jsonrpc": "2.0", "method": "exit", "params": nil,
	}))
	require.NoError(t, stdin.Close())
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- command.Wait()
	}()
	select {
	case err = <-waitCh:
	case <-time.After(20 * time.Second):
		_ = command.Process.Kill()
		t.Fatal("language-server subprocess did not exit")
	}
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLanguageServerCommandHelperProcess(t *testing.T) {
	if os.Getenv(languageServerHelperEnvironment) != "1" {
		return
	}
	os.Args = []string{os.Args[0], "language-server", "--no-update-check"}
	Execute("v-test", "", "")
	os.Exit(0)
}

type processFrameWriter struct {
	writer io.Writer
}

func (w *processFrameWriter) write(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(payload), payload)
	_, err = io.WriteString(w.writer, frame)
	return err
}

type processFrameReader struct {
	reader *bufio.Reader
}

func newProcessFrameReader(reader io.Reader) *processFrameReader {
	return &processFrameReader{reader: bufio.NewReader(reader)}
}

func (r *processFrameReader) read(t *testing.T) map[string]any {
	t.Helper()
	contentLength := -1
	for {
		line, err := r.reader.ReadString('\n')
		require.NoError(t, err)
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		require.True(t, ok)
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			contentLength, err = strconv.Atoi(strings.TrimSpace(value))
			require.NoError(t, err)
		}
	}
	require.True(t, contentLength >= 0)
	payload := make([]byte, contentLength)
	_, err := io.ReadFull(r.reader, payload)
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, json.Unmarshal(payload, &value))
	return value
}

func processMessageParams(t *testing.T, message map[string]any) map[string]any {
	t.Helper()
	params, ok := message["params"].(map[string]any)
	require.True(t, ok)
	return params
}

func processResponseError(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	responseErr, ok := response["error"].(map[string]any)
	require.True(t, ok)
	return responseErr
}
