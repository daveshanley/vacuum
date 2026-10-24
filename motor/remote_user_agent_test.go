// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/require"
)

func TestRemoteReferencesSendVacuumUserAgent(t *testing.T) {
	for _, useTLS := range []bool{false, true} {
		for _, format := range []string{"OpenAPI", "AsyncAPI"} {
			t.Run(fmt.Sprintf("%s/TLS=%t", format, useTLS), func(t *testing.T) {
				var accepted, rejected atomic.Int32
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.UserAgent() != "vacuum-linter/1.0" {
						rejected.Add(1)
						http.Error(w, "User-Agent required", http.StatusForbidden)
						return
					}
					accepted.Add(1)
					w.Header().Set("Content-Type", "application/yaml")
					_, _ = io.WriteString(w, "type: string\n")
				})
				var server *httptest.Server
				if useTLS {
					server = httptest.NewTLSServer(handler)
				} else {
					server = httptest.NewServer(handler)
				}
				defer server.Close()
				spec := fmt.Sprintf("openapi: 3.0.3\ninfo: {title: Test, version: '1.0'}\npaths: {}\ncomponents:\n  schemas:\n    Value:\n      $ref: '%s/schema.yaml'\n", server.URL)
				if format == "AsyncAPI" {
					spec = fmt.Sprintf("asyncapi: 3.0.0\ninfo: {title: Test, version: '1.0'}\nchannels: {}\ncomponents:\n  schemas:\n    Value:\n      $ref: '%s/schema.yaml'\n", server.URL)
				}
				result := ApplyRulesToRuleSet(&RuleSetExecution{
					Spec: []byte(spec), AllowLookup: true, SilenceLogs: true,
					RuleSet:          &rulesets.RuleSet{Rules: map[string]*model.Rule{}},
					HTTPClientConfig: utils.HTTPClientConfig{Insecure: useTLS},
				})
				defer result.ReleaseOwnedResources()
				require.Empty(t, result.Errors)
				require.Zero(t, rejected.Load())
				require.Positive(t, accepted.Load())
				require.NotNil(t, result.Index)
				require.Empty(t, result.Index.GetRolodex().GetCaughtErrors())
				require.NotEmpty(t, result.Index.GetRolodex().GetAllMappedReferences())
			})
		}
	}
}
