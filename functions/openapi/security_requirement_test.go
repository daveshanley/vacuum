package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/require"
)

func TestSecurityRequirementIssue1044(t *testing.T) {
	for _, tc := range []struct {
		name, version, requirement, scheme string
		missing                            []string
		unused                             bool
	}{
		{name: "local URI", requirement: "'#/components/securitySchemes/ApiKey': []"},
		{name: "named scheme", requirement: "ApiKey: []"},
		{name: "missing name", requirement: "Missing: []", missing: []string{"Missing"}, unused: true},
		{name: "missing URI", requirement: "'#/components/securitySchemes/Missing': []", missing: []string{"#/components/securitySchemes/Missing"}, unused: true},
		{name: "non-security target", requirement: "'#/info': []", missing: []string{"#/info"}, unused: true},
		{name: "nested target", requirement: "'#/components/securitySchemes/ApiKey/type': []", missing: []string{"#/components/securitySchemes/ApiKey/type"}, unused: true},
		{name: "encoded URI", requirement: "'#/%63omponents/securitySchemes/%41piKey': []"},
		{name: "encoded slash", requirement: "'#/components/securitySchemes/Api%2FKey': []", scheme: "Api/Key", missing: []string{"#/components/securitySchemes/Api%2FKey"}, unused: true},
		{name: "escaped slash", requirement: "'#/components/securitySchemes/Api~1Key': []", scheme: "Api/Key"},
		{name: "escaped tilde", requirement: "'#/components/securitySchemes/Api~0Key': []", scheme: "Api~Key"},
		{name: "decode pointer once", requirement: "'#/components/securitySchemes/Api~01Key': []", scheme: "Api~1Key"},
		{name: "decode percent once", requirement: "'#/components/securitySchemes/Api%2541Key': []", scheme: "Api%41Key"},
		{name: "literal plus", requirement: "'#/components/securitySchemes/Api+Key': []", scheme: "Api+Key"},
		{name: "bad percent", requirement: "'#/components/securitySchemes/Api%ZZKey': []", scheme: "Api%ZZKey", missing: []string{"#/components/securitySchemes/Api%ZZKey"}, unused: true},
		{name: "trailing tilde", requirement: "'#/components/securitySchemes/Api~': []", scheme: "Api~", missing: []string{"#/components/securitySchemes/Api~"}, unused: true},
		{name: "bad tilde", requirement: "'#/components/securitySchemes/Api~2Key': []", scheme: "Api~2Key", missing: []string{"#/components/securitySchemes/Api~2Key"}, unused: true},
		{name: "name precedence", requirement: "'#/components/securitySchemes/ApiKey': []", scheme: "#/components/securitySchemes/ApiKey"},
		{name: "external URI stays external", requirement: "'other.yaml#/components/securitySchemes/ApiKey': []", missing: []string{"other.yaml#/components/securitySchemes/ApiKey"}, unused: true},
		{name: "absolute URI stays external", requirement: "'https://example.com/api.yaml#/components/securitySchemes/ApiKey': []", missing: []string{"https://example.com/api.yaml#/components/securitySchemes/ApiKey"}, unused: true},
		{name: "3.0 rejects URI", version: "3.0.3", requirement: "'#/components/securitySchemes/ApiKey': []", missing: []string{"#/components/securitySchemes/ApiKey"}, unused: true},
		{name: "3.1 rejects URI", version: "3.1.0", requirement: "'#/components/securitySchemes/ApiKey': []", missing: []string{"#/components/securitySchemes/ApiKey"}, unused: true},
		{name: "3.0 accepts name", version: "3.0.3", requirement: "ApiKey: []"},
		{name: "3.1 accepts name", version: "3.1.0", requirement: "ApiKey: []"},
		{name: "second missing", requirement: "ApiKey: []\n          Missing: []", missing: []string{"Missing"}},
		{name: "first missing", requirement: "Missing: []\n          ApiKey: []", missing: []string{"Missing"}},
		{name: "both missing", requirement: "Missing: []\n          Other: []", missing: []string{"Missing", "Other"}, unused: true},
		{name: "two valid keys", requirement: "ApiKey: []\n          '#/components/securitySchemes/ApiKey': []"},
		{name: "anonymous", requirement: "{}", unused: true},
	} {
		for _, location := range []string{"operation", "root"} {
			t.Run(tc.name+"/"+location, func(t *testing.T) {
				version, scheme := tc.version, tc.scheme
				if version == "" {
					version = "3.2.0"
				}
				if scheme == "" {
					scheme = "ApiKey"
				}
				spec := fmt.Sprintf(`openapi: %s
info: {title: Security test, version: '1'}
paths:
  /items:
    get:
      security:
        - %s
      responses:
        '204': {description: Success}
components:
  securitySchemes:
    %q:
      type: apiKey
      in: header
      name: X-API-Key
`, version, tc.requirement, scheme)
				path := "$.paths['/items'].get.security[0]"
				if location == "root" {
					block := "      security:\n        - " + tc.requirement + "\n"
					spec = strings.Replace(spec, block, "", 1)
					rootSecurity := "security:\n  - " + strings.ReplaceAll(tc.requirement, "\n          ", "\n    ") + "\n"
					spec = strings.Replace(spec, "paths:\n", rootSecurity+"paths:\n", 1)
					path = "$.security[0]"
				}
				var root yaml.Node
				require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
				info, err := datamodel.ExtractSpecInfo([]byte(spec))
				require.NoError(t, err)
				config := index.CreateClosedAPIIndexConfig()
				config.SpecInfo = info
				idx := index.NewSpecIndexWithConfig(&root, config)
				ctx := model.RuleFunctionContext{Index: idx, SpecInfo: info, Rule: &model.Rule{Id: "oas3-operation-security-defined"}}
				results := (OperationSecurityDefined{}).RunRule(root.Content, ctx)
				require.Len(t, results, len(tc.missing))
				for i, result := range results {
					require.Equal(t, tc.missing[i], result.StartNode.Value)
					require.Contains(t, result.Message, "`"+tc.missing[i]+"`")
					require.Equal(t, path, result.Path)
					require.Same(t, ctx.Rule, result.Rule)
					require.Positive(t, result.StartNode.Line)
					require.NotNil(t, result.EndNode)
				}
				unusedResults := setupUnusedComponentsTestContext(t, spec)
				if tc.unused {
					require.Len(t, unusedResults, 1)
					require.Contains(t, unusedResults[0].Message, "potentially unused")
				} else {
					require.Empty(t, unusedResults)
				}
			})
		}
	}
}

func TestSecurityRequirementIssue1044_ReferencedOperation(t *testing.T) {
	for _, childDefinesScheme := range []bool{false, true} {
		for _, sameCoordinates := range []bool{false, true} {
			t.Run(fmt.Sprintf("child scheme %t/same coordinates %t", childDefinesScheme, sameCoordinates), func(t *testing.T) {
				dir := t.TempDir()
				root := `openapi: 3.2.0
info: {title: Child test, version: '1'}
paths:
  /items:
    $ref: 'child.yaml#/paths/~1items'
components:
  securitySchemes:
    ApiKey:
      type: apiKey
      in: header
      name: X-Root-Key
`
				child := `openapi: 3.2.0
info: {title: Child test, version: '1'}
paths:
  /items:
    get:
      security:
        - '#/components/securitySchemes/ApiKey': []
      responses:
        '204': {description: Success}
`
				if sameCoordinates {
					root = strings.Replace(root, "paths:\n", `paths:
  /root:
    get:
      security:
        - '#/components/securitySchemes/ApiKey': []
      responses:
        '204': {description: Success}
`, 1)
				}
				if childDefinesScheme {
					child += `components:
  securitySchemes:
    ApiKey:
      type: apiKey
      in: header
      name: X-Child-Key
`
				}
				require.NoError(t, os.WriteFile(filepath.Join(dir, "child.yaml"), []byte(child), 0600))
				cfg := datamodel.NewDocumentConfiguration()
				cfg.BasePath = dir
				cfg.SpecFilePath = "root.yaml"
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
					require.Equal(t, "#/components/securitySchemes/ApiKey", results[0].StartNode.Value)
				}
				unused := (UnusedComponent{}).RunRule(doc.GetSpecInfo().RootNode.Content, ctx)
				if sameCoordinates {
					require.Empty(t, unused)
				} else {
					require.Len(t, unused, 1)
					require.Contains(t, unused[0].Message, "#/components/securitySchemes/ApiKey")
				}
			})
		}
	}
}

func TestSecurityRequirementIssue1044_UnownedNode(t *testing.T) {
	// Content equality cannot authorize a fragment lookup when its document is unknown.
	spec := []byte(`openapi: 3.2.0
security:
  - '#/components/securitySchemes/ApiKey': []
components:
  securitySchemes:
    ApiKey: {type: http, scheme: bearer}
`)
	var root, copied yaml.Node
	require.NoError(t, yaml.Unmarshal(spec, &root))
	require.NoError(t, yaml.Unmarshal(spec, &copied))
	cfg := index.CreateClosedAPIIndexConfig()
	cfg.Rolodex = index.NewRolodex(index.CreateClosedAPIIndexConfig())
	child := index.NewSpecIndexWithConfig(&copied, index.CreateClosedAPIIndexConfig())
	cfg.Rolodex.AddIndex(child)
	rootIndex := index.NewSpecIndexWithConfig(&root, cfg)
	info, err := datamodel.ExtractSpecInfo(spec)
	require.NoError(t, err)
	// A third copy belongs to neither index, despite identical content and coordinates.
	var unowned yaml.Node
	require.NoError(t, yaml.Unmarshal(spec, &unowned))
	ctx := model.RuleFunctionContext{Index: rootIndex, SpecInfo: info}
	definitions := map[*index.SpecIndex]map[string]*index.Reference{rootIndex: rootIndex.GetAllSecuritySchemes()}
	results := (OperationSecurityDefined{}).checkSecurityNode(unowned.Content[0].Content[3], definitions, nil, "$", ctx)
	require.Len(t, results, 1)
	require.Equal(t, "#/components/securitySchemes/ApiKey", results[0].StartNode.Value)
}
