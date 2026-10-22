// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func TestReturnedDocumentConfigRemainsReusableAfterExecution(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"type":"string"}`)
	}))
	defer server.Close()

	for _, timeout := range []time.Duration{0, time.Hour} {
		for _, supplied := range []bool{false, true} {
			name := timeout.String() + "/motor-handler"
			if supplied {
				name = timeout.String() + "/caller-handler"
			}
			t.Run(name, func(t *testing.T) {
				caller, cancel := context.WithCancel(context.Background())
				defer cancel()
				execution := newExecutionOptionsTestExecution(&countingRuleFunction{})
				execution.HTTPClientConfig = utils.HTTPClientConfig{Insecure: true}
				if supplied {
					config := datamodel.NewDocumentConfiguration()
					config.RemoteURLHandler = server.Client().Get
					document, err := libopenapi.NewDocumentWithConfiguration(execution.Spec, config)
					require.NoError(t, err)
					execution.Document = document
					defer document.Release()
				}
				result := ApplyRulesToRuleSetWithOptions(execution, &ExecutionOptions{
					Context: caller, RunTimeout: timeout,
				})
				defer result.ReleaseOwnedResources()
				require.Empty(t, result.Errors)
				require.NotNil(t, result.DocumentConfig)
				require.NotNil(t, result.DocumentConfig.RemoteURLHandler)
				cancel()

				response, err := result.DocumentConfig.RemoteURLHandler(server.URL)
				require.NoError(t, err)
				defer response.Body.Close()
				body, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				require.Equal(t, `{"type":"string"}`, string(body))
			})
		}
	}
}
