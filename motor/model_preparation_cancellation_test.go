// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func TestCancellationDuringModelPreparationSkipsRemainingPhases(t *testing.T) {
	for _, version := range []struct {
		name, header, schemas string
	}{
		{"OpenAPI2", "swagger: '2.0'", "definitions"},
		{"OpenAPI3", "openapi: 3.0.0", "components:\n  schemas"},
	} {
		t.Run(version.name, func(t *testing.T) {
			for _, cancelOnCall := range []int32{1, 2} {
				t.Run("cancel-after-model-"+strconv.Itoa(int(cancelOnCall)), func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					config := datamodel.NewDocumentConfiguration()
					config.AllowRemoteReferences = true
					var requests atomic.Int32
					config.RemoteURLHandler = func(string) (*http.Response, error) {
						if requests.Add(1) == cancelOnCall {
							cancel()
						}
						return &http.Response{
							StatusCode: http.StatusOK,
							Header:     make(http.Header),
							Body:       io.NopCloser(strings.NewReader("type: string\n")),
						}, nil
					}
					indent := "  "
					if version.name == "OpenAPI3" {
						indent = "    "
					}
					spec := []byte(version.header + "\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}\n" +
						version.schemas + ":\n" + indent + "Remote:\n" + indent + "  $ref: https://example.test/schema.yaml\n")
					document, err := libopenapi.NewDocumentWithConfiguration(spec, config)
					require.NoError(t, err)
					defer document.Release()
					execution := &RuleSetExecution{
						Spec: spec, Document: document, RuleSet: &rulesets.RuleSet{}, SilenceLogs: true,
					}
					result := ApplyRulesToRuleSetWithOptions(execution, &ExecutionOptions{Context: ctx})
					require.Len(t, result.Errors, 1)
					require.ErrorIs(t, result.Errors[0], context.Canceled)
					require.Equal(t, cancelOnCall, requests.Load(), "cancellation must prevent the next model from loading references")
					require.Nil(t, execution.DrDocument, "cancellation must prevent Doctor construction")
					require.NotNil(t, result.Index)
					root := result.Index.GetRootNode()
					result.ReleaseOwnedResources()
					require.Same(t, root, document.GetRolodex().GetRootIndex().GetRootNode(), "release must preserve the supplied document")
				})
			}
		})
	}
}
