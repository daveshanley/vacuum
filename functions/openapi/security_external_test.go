// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/testify/require"
)

func TestSecurityRequirementIssue1056(t *testing.T) {
	for _, tc := range []struct {
		name, requirement, version, self string
		missing, named                   bool
	}{
		{name: "named alias", requirement: "ApiKey"},
		{name: "external URI", requirement: "./security.json#/components/securitySchemes/ApiKey"},
		{name: "dot segments", requirement: "./unused/../security.json#/components/securitySchemes/ApiKey"},
		{name: "encoded fragment", requirement: "security.json#/%63omponents/securitySchemes/%41piKey"},
		{name: "missing scheme", requirement: "security.json#/components/securitySchemes/Missing", missing: true},
		{name: "nested property", requirement: "security.json#/components/securitySchemes/ApiKey/type", missing: true},
		{name: "non-scheme pointer", requirement: "security.json#/info", missing: true},
		{name: "invalid escape", requirement: "security.json#/components/securitySchemes/Api~2Key", missing: true},
		{name: "query differs", requirement: "security.json?v=2#/components/securitySchemes/ApiKey", missing: true},
		{name: "not loaded", requirement: "unloaded.json#/components/securitySchemes/ApiKey", missing: true},
		{name: "remote not loaded", requirement: "https://example.test/security.json#/components/securitySchemes/ApiKey", missing: true},
		{name: "name precedence", requirement: "unloaded.json#/components/securitySchemes/ApiKey", named: true},
		{name: "target self", requirement: "https://example.test/security.json#/components/securitySchemes/ApiKey", self: "https://example.test/security.json"},
		{name: "target self overrides retrieval", requirement: "security.json#/components/securitySchemes/ApiKey", self: "https://example.test/security.json", missing: true},
		{name: "3.0 rejects URI", requirement: "security.json#/components/securitySchemes/ApiKey", version: "3.0.3", missing: true},
		{name: "3.1 rejects URI", requirement: "security.json#/components/securitySchemes/ApiKey", version: "3.1.0", missing: true},
	} {
		for _, rootSecurity := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/root=%t", tc.name, rootSecurity), func(t *testing.T) {
				dir := t.TempDir()
				shared := `{"openapi":"3.2.0","info":{"title":"Shared security","version":"1"},"paths":{},"components":{"securitySchemes":{"ApiKey":{"type":"apiKey","in":"header","name":"X-API-Key"}}}}`
				if tc.self != "" {
					shared = strings.Replace(shared, "{", fmt.Sprintf(`{"$self":%q,`, tc.self), 1)
				}
				require.NoError(t, os.WriteFile(filepath.Join(dir, "security.json"), []byte(shared), 0600))
				// Merely existing on disk must not admit a file to the lookup.
				require.NoError(t, os.WriteFile(filepath.Join(dir, "unloaded.json"), []byte(shared), 0600))
				version := tc.version
				if version == "" {
					version = "3.2.0"
				}
				spec := fmt.Sprintf(`openapi: %s
info: {title: External security, version: '1'}
paths:
  /items:
    get:
      security:
        - %q: []
      responses:
        '204': {description: Success}
components:
  securitySchemes:
    ApiKey:
      $ref: './security.json#/components/securitySchemes/ApiKey'
`, version, tc.requirement)
				if tc.named {
					spec += fmt.Sprintf("    %q: {type: http, scheme: bearer}\n", tc.requirement)
				}
				path := "$.paths['/items'].get.security[0]"
				if rootSecurity {
					spec = strings.Replace(spec, fmt.Sprintf("      security:\n        - %q: []\n", tc.requirement), "", 1)
					spec += fmt.Sprintf("security:\n  - %q: []\n", tc.requirement)
					path = "$.security[0]"
				}
				cfg := datamodel.NewDocumentConfiguration()
				cfg.BasePath, cfg.SpecFilePath = dir, "api.yaml"
				cfg.AllowRemoteReferences = true
				cfg.RemoteURLHandler = func(uri string) (*http.Response, error) {
					t.Errorf("unexpected fetch: %s", uri)
					return nil, fmt.Errorf("unexpected fetch")
				}
				doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), cfg)
				require.NoError(t, err)
				built, err := doc.BuildV3Model()
				require.NoError(t, err)
				ctx := model.RuleFunctionContext{Index: built.Index, Document: doc, SpecInfo: doc.GetSpecInfo(), Rule: &model.Rule{Id: "oas3-operation-security-defined"}}
				results := (OperationSecurityDefined{}).RunRule(doc.GetSpecInfo().RootNode.Content, ctx)
				if tc.missing {
					require.Len(t, results, 1)
					require.Equal(t, tc.requirement, results[0].StartNode.Value)
					require.Equal(t, path, results[0].Path)
					require.Same(t, ctx.Rule, results[0].Rule)
					require.Positive(t, results[0].StartNode.Line)
				} else {
					require.Empty(t, results)
				}
				if tc.name == "external URI" {
					unused := (UnusedComponent{}).RunRule(doc.GetSpecInfo().RootNode.Content, ctx)
					require.Len(t, unused, 1, "direct external use does not use the local alias")
				}
			})
		}
	}
}

func TestSecurityRequirementIssue1056_ReferencedOperation(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "operations"), 0700))
	child := `openapi: 3.2.0
info: {title: Child, version: '1'}
paths:
  /items:
    get:
      security:
        - '../security.yaml#/components/securitySchemes/ApiKey': []
      responses:
        '204': {description: Success}
`
	shared := `openapi: 3.2.0
info: {title: Shared, version: '1'}
paths: {}
components:
  securitySchemes:
    ApiKey: {type: apiKey, in: header, name: X-API-Key}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "operations", "child.yaml"), []byte(child), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "security.yaml"), []byte(shared), 0600))
	root := `openapi: 3.2.0
info: {title: Root, version: '1'}
paths:
  /items:
    $ref: operations/child.yaml#/paths/~1items
components:
  securitySchemes:
    ApiKey:
      $ref: security.yaml#/components/securitySchemes/ApiKey
`
	cfg := datamodel.NewDocumentConfiguration()
	cfg.BasePath, cfg.SpecFilePath = dir, "root.yaml"
	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(root), cfg)
	require.NoError(t, err)
	built, err := doc.BuildV3Model()
	require.NoError(t, err)
	ctx := model.RuleFunctionContext{Index: built.Index, Document: doc, SpecInfo: doc.GetSpecInfo()}
	results := (OperationSecurityDefined{}).RunRule(doc.GetSpecInfo().RootNode.Content, ctx)
	require.Empty(t, results)
}
