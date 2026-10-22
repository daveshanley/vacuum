package openapi

import (
	"encoding/json"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/require"
	"strings"
	"testing"
)

func TestOASSchema_ComponentNames(t *testing.T) {
	bad := []string{"Bad Name", "Bad/Name", "Bad[Name]", "Bad=Name", "Bad,Name", "Bad`Name", "Bad<Name>", "Bad(Name)", "Bad'Name", ""}
	categories := map[string]any{
		"schemas":         map[string]any{"type": "object"},
		"responses":       map[string]any{"description": "ok"},
		"parameters":      map[string]any{"name": "id", "in": "query", "schema": map[string]any{"type": "string"}},
		"examples":        map[string]any{"value": "ok"},
		"requestBodies":   map[string]any{"content": map[string]any{}},
		"headers":         map[string]any{"schema": map[string]any{"type": "string"}},
		"securitySchemes": map[string]any{"type": "http", "scheme": "bearer"},
		"links":           map[string]any{"operationId": "test"},
		"callbacks":       map[string]any{},
	}
	for _, version := range []string{"3.0.4", "3.1.1"} {
		for category, value := range categories {
			t.Run(version+"/"+category, func(t *testing.T) {
				entries := map[string]any{"Valid_1.name-2": value}
				for _, name := range bad {
					entries[name] = value
				}
				raw, err := json.Marshal(map[string]any{"openapi": version, "info": map[string]string{"title": "Test", "version": "1"}, "paths": map[string]any{}, "components": map[string]any{category: entries}})
				require.NoError(t, err)
				document, err := libopenapi.NewDocument(raw)
				require.NoError(t, err)
				defer document.Release()
				info := document.GetSpecInfo()
				results := (OASSchema{}).RunRule([]*yaml.Node{info.RootNode}, model.RuleFunctionContext{Rule: &model.Rule{Name: "oas3-schema"}, SpecInfo: info, Document: document})
				require.Len(t, results, len(bad))
				found := map[string]bool{}
				for _, result := range results {
					name := result.StartNode.Value
					require.Contains(t, bad, name)
					require.False(t, found[name], "duplicate %q", name)
					found[name] = true
					require.Contains(t, result.Message, name)
					pointer := fmt.Sprintf("/components/%s/%s", category, strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1"))
					require.Equal(t, jsonPointerToJSONPath(pointer, info.RootNode), result.Path)
					require.Greater(t, result.StartNode.Column, 0)
				}
			})
		}
	}
}
