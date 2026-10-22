package core

import (
	"fmt"
	"sync"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi"
	openapiUtils "github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/require"
)

func TestLocateNodePathsMatchesAttachedModel(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info: {title: Locations, version: '1.0'}
paths: {}
x-settings: {version: '1.0'}
components:
  schemas:
    Missing: &missing {type: string}
    AlsoMissing: *missing
    Plain: {type: string}
`)
	for _, cached := range []bool{false, true} {
		for _, resolved := range []bool{false, true} {
			t.Run(fmt.Sprintf("cached=%t/resolved=%t", cached, resolved), func(t *testing.T) {
				document, err := libopenapi.NewDocument(spec)
				require.NoError(t, err)
				high, err := document.BuildV3Model()
				require.NoError(t, err)
				doctor := drModel.NewDrDocumentWithConfig(high, &drModel.DrConfig{UseSchemaCache: cached, DeterministicPaths: true})
				ctx := model.RuleFunctionContext{DrDocument: doctor, Document: document, Index: high.Index, Rule: &model.Rule{Resolved: resolved}, SchemaPathCache: new(sync.Map)}
				_, components := openapiUtils.FindKeyNodeTop("components", high.Index.GetRootNode().Content[0].Content)
				_, schemas := openapiUtils.FindKeyNodeTop("schemas", components.Content)
				_, extension := openapiUtils.FindKeyNodeTop("x-settings", high.Index.GetRootNode().Content[0].Content)
				for i, name := range []string{"Missing", "AlsoMissing", "Plain", "x-settings"} {
					path := "$.components.schemas['" + name + "']"
					node := extension
					if name == "x-settings" {
						path = "$['x-settings']"
					} else {
						node = schemas.Content[2*i+1]
					}
					require.NotNil(t, node)
					located, paths, objects := locateNodePaths(&ctx, node)
					require.Equal(t, path, located)
					if !resolved {
						require.Empty(t, paths)
					}
					for _, object := range objects {
						require.Equal(t, path, object.GenerateJSONPath())
					}
					if path == "$['x-settings']" {
						require.Empty(t, objects)
					}
				}
			})
		}
	}
}

func TestOr_AttachesAliasResultToMatchingModel(t *testing.T) {
	spec := []byte("openapi: 3.0.3\ninfo: {title: Aliases, version: '1.0'}\npaths: {}\ncomponents:\n  schemas:\n    Missing: &schema {type: string}\n    AlsoMissing: *schema\n")
	document, err := libopenapi.NewDocument(spec)
	require.NoError(t, err)
	high, err := document.BuildV3Model()
	require.NoError(t, err)
	doctor := drModel.NewDrDocument(high)
	nodes, err := openapiUtils.FindNodesWithoutDeserializing(high.Index.GetRootNode(), "$.components.schemas.*")
	require.NoError(t, err)
	ctx := model.RuleFunctionContext{DrDocument: doctor, Index: high.Index, Rule: &model.Rule{}, Options: map[string]any{"properties": []string{"title", "description"}}}
	results := (Or{}).RunRule(nodes, ctx)
	require.Len(t, results, 2)
	for _, schema := range doctor.Schemas {
		for _, result := range schema.GetRuleFunctionResults() {
			require.Equal(t, schema.GenerateJSONPath(), result.Path)
		}
	}
	for _, result := range results {
		objects, err := doctor.LocateModel(result.StartNode)
		require.NoError(t, err)
		attached := false
		for _, object := range objects {
			if object.GenerateJSONPath() == result.Path {
				require.Len(t, object.(v3.AcceptsRuleResults).GetRuleFunctionResults(), 1)
				attached = true
			}
		}
		require.True(t, attached, result.Path)
	}
}

func TestLocateNodePathsPreservesResolvedReferences(t *testing.T) {
	spec := []byte("openapi: 3.0.3\ninfo: {title: References, version: '1.0'}\npaths: {}\ncomponents:\n  schemas:\n    Shared: &shared {type: string}\n    Ref: *shared\n")
	document, err := libopenapi.NewDocument(spec)
	require.NoError(t, err)
	high, err := document.BuildV3Model()
	require.NoError(t, err)
	doctor := drModel.NewDrDocument(high)
	nodes, err := openapiUtils.FindNodesWithoutDeserializing(high.Index.GetRootNode(), "$.components.schemas.Shared")
	require.NoError(t, err)
	path, paths, objects := locateNodePaths(&model.RuleFunctionContext{DrDocument: doctor, Index: high.Index, Rule: &model.Rule{Resolved: true}}, nodes[0])
	require.Equal(t, "$.components.schemas['Shared']", path)
	require.ElementsMatch(t, []string{"$.components.schemas['Shared']", "$.components.schemas['Ref']"}, paths)
	require.Equal(t, path, objects[0].GenerateJSONPath())
}
