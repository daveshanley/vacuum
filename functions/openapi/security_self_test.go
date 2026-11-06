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

func TestSecurityRequirementIssue1053(t *testing.T) {
	for _, tc := range []struct {
		name, self, selfYAML, retrieval, requirement string
		missing                                      bool
	}{
		{name: "aliased self", selfYAML: "x-identity: &identity https://example.test/security.yaml\n$self: *identity", requirement: "./security.yaml#/components/securitySchemes/ApiKey"},
		{name: "merged self", selfYAML: "x-identity: &identity\n  $self: https://example.test/security.yaml\n<<: *identity", requirement: "./security.yaml#/components/securitySchemes/ApiKey"},
		{name: "absolute self", self: "https://example.test/security.yaml", requirement: "https://example.test/security.yaml#/components/securitySchemes/ApiKey"},
		{name: "dot segments in self", self: "https://example.test/api/../security.yaml", requirement: "https://example.test/security.yaml#/components/securitySchemes/ApiKey"},
		{name: "scheme and host case", self: "HTTPS://EXAMPLE.TEST/security.yaml", requirement: "https://example.test/security.yaml#/components/securitySchemes/ApiKey"},
		{name: "relative to self", self: "https://example.test/security.yaml", requirement: "./security.yaml#/components/securitySchemes/ApiKey"},
		{name: "dot segments", self: "https://example.test/api/security.yaml", requirement: "../api/./security.yaml#/components/securitySchemes/ApiKey"},
		{name: "fragment with self", self: "https://example.test/security.yaml", requirement: "#/components/securitySchemes/ApiKey"},
		{name: "missing target", self: "https://example.test/security.yaml", requirement: "https://example.test/security.yaml#/components/securitySchemes/Missing", missing: true},
		{name: "other document", self: "https://example.test/security.yaml", requirement: "https://example.test/other.yaml#/components/securitySchemes/ApiKey", missing: true},
		{name: "other host", self: "https://example.test/security.yaml", requirement: "https://other.test/security.yaml#/components/securitySchemes/ApiKey", missing: true},
		{name: "other query", self: "https://example.test/security.yaml?v=1", requirement: "https://example.test/security.yaml?v=2#/components/securitySchemes/ApiKey", missing: true},
		{name: "query identity", self: "https://example.test/security.yaml?v=1", requirement: "./security.yaml?v=1#/components/securitySchemes/ApiKey"},
		{name: "relative self", self: "../schemas/security.yaml", retrieval: "https://example.test/api/root.yaml", requirement: "https://example.test/schemas/security.yaml#/components/securitySchemes/ApiKey"},
		{name: "retrieval URI", retrieval: "https://example.test/security.yaml", requirement: "./security.yaml#/components/securitySchemes/ApiKey"},
		{name: "self overrides retrieval", self: "https://example.test/security.yaml", retrieval: "https://example.test/root.yaml", requirement: "./root.yaml#/components/securitySchemes/ApiKey", missing: true},
		{name: "relative file self", self: "security.yaml", retrieval: filepath.Join(t.TempDir(), "download.json"), requirement: "./security.yaml#/components/securitySchemes/ApiKey"},
		{name: "file identity", retrieval: filepath.Join(t.TempDir(), "security.yaml"), requirement: "./security.yaml#/components/securitySchemes/ApiKey"},
		{name: "invalid self", self: "https://example.test/%ZZ", requirement: "./security.yaml#/components/securitySchemes/ApiKey", missing: true},
		{name: "invalid URI", self: "https://example.test/security.yaml", requirement: "https://example.test/%ZZ#/components/securitySchemes/ApiKey", missing: true},
		{name: "invalid fragment", self: "https://example.test/security.yaml", requirement: "./security.yaml#/components/securitySchemes/ApiKey/type", missing: true},
		{name: "percent fragment once", self: "https://example.test/security.yaml", requirement: "./security.yaml#/%63omponents/securitySchemes/%41piKey"},
	} {
		for _, rootSecurity := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/root=%t", tc.name, rootSecurity), func(t *testing.T) {
				spec := fmt.Sprintf(`openapi: 3.2.0
info: {title: Security test, version: '1'}
paths:
  /items:
    get:
      security:
        - %q: []
      responses:
        '204': {description: Success}
components:
  securitySchemes:
    ApiKey: {type: apiKey, in: header, name: X-API-Key}
`, tc.requirement)
				if tc.self != "" {
					spec = "$self: " + fmt.Sprintf("%q", tc.self) + "\n" + spec
				}
				if tc.selfYAML != "" {
					spec = tc.selfYAML + "\n" + spec
				}
				if rootSecurity {
					spec = strings.Replace(spec, fmt.Sprintf("      security:\n        - %q: []\n", tc.requirement), "", 1)
					spec += fmt.Sprintf("security:\n  - %q: []\n", tc.requirement)
				}
				cfg := datamodel.NewDocumentConfiguration()
				cfg.AllowRemoteReferences = true
				cfg.RemoteURLHandler = func(uri string) (*http.Response, error) {
					t.Errorf("unexpected fetch: %s", uri)
					return nil, fmt.Errorf("unexpected fetch")
				}
				doc, err := libopenapi.NewDocumentWithConfiguration([]byte(spec), cfg)
				require.NoError(t, err)
				built, err := doc.BuildV3Model()
				require.NoError(t, err)
				built.Index.SetAbsolutePath(tc.retrieval)
				ctx := model.RuleFunctionContext{Index: built.Index, Document: doc, SpecInfo: doc.GetSpecInfo(), Rule: &model.Rule{Id: "oas3-operation-security-defined"}}
				nodes := doc.GetSpecInfo().RootNode.Content
				results := (OperationSecurityDefined{}).RunRule(nodes, ctx)
				unused := (UnusedComponent{}).RunRule(nodes, ctx)
				if tc.missing {
					require.Len(t, results, 1)
					require.Equal(t, tc.requirement, results[0].StartNode.Value)
					require.Same(t, ctx.Rule, results[0].Rule)
					require.Positive(t, results[0].StartNode.Line)
					require.Len(t, unused, 1)
				} else {
					require.Empty(t, results)
					require.Empty(t, unused)
				}
			})
		}
	}
}

func TestSecurityRequirementIssue1053_ReferencedSource(t *testing.T) {
	for _, tc := range []struct{ name, requirement string }{
		{"relative", "./child.yaml#/components/securitySchemes/ApiKey"},
		{"absolute", "https://example.test/child.yaml#/components/securitySchemes/ApiKey"},
	} {
		requirement := tc.requirement
		for _, childDefinesScheme := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/childScheme=%t", tc.name, childDefinesScheme), func(t *testing.T) {
				dir := t.TempDir()
				const root = `openapi: 3.2.0
info: {title: Root, version: '1'}
paths:
  /items:
    $ref: child.yaml#/paths/~1items
components:
  securitySchemes:
    ApiKey: {type: apiKey, in: header, name: X-Root-Key}
`
				child := fmt.Sprintf(`openapi: 3.2.0
$self: https://example.test/child.yaml
info: {title: Child, version: '1'}
paths:
  /items:
    get:
      security:
        - %q: []
      responses:
        '204': {description: Success}
`, requirement)
				if childDefinesScheme {
					child += "components:\n  securitySchemes:\n    ApiKey: {type: apiKey, in: header, name: X-Child-Key}\n"
				}
				require.NoError(t, os.WriteFile(filepath.Join(dir, "child.yaml"), []byte(child), 0600))
				cfg := datamodel.NewDocumentConfiguration()
				cfg.BasePath, cfg.SpecFilePath = dir, "root.yaml"
				doc, err := libopenapi.NewDocumentWithConfiguration([]byte(root), cfg)
				require.NoError(t, err)
				built, err := doc.BuildV3Model()
				require.NoError(t, err)
				ctx := model.RuleFunctionContext{Index: built.Index, Document: doc, SpecInfo: doc.GetSpecInfo()}
				results := (OperationSecurityDefined{}).RunRule(doc.GetSpecInfo().RootNode.Content, ctx)
				if childDefinesScheme {
					require.Empty(t, results)
				} else {
					require.Len(t, results, 1)
					require.Equal(t, requirement, results[0].StartNode.Value)
				}
				unused := (UnusedComponent{}).RunRule(doc.GetSpecInfo().RootNode.Content, ctx)
				require.Len(t, unused, 1, "a URI to the child must not mark the root scheme as used")
			})
		}
	}
}
