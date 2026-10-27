package openapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/require"
)

func TestComponentDescriptionPreservesComponentKeyNodes(t *testing.T) {
	components := []struct{ name, body string }{
		{"schemas", "type: string"},
		{"parameters", "name: q\n      in: query"},
		{"requestBodies", "content: {}"},
		{"responses", "content: {}"},
		{"examples", "value: example"},
		{"headers", "schema: {type: string}"},
		{"links", "operationId: example"},
		{"securitySchemes", "type: apiKey\n      in: header\n      name: X-Test"},
	}
	for _, component := range components {
		t.Run(component.name, func(t *testing.T) {
			// OpenAPI component names are case-sensitive. A fresh linear, case-folded
			// lookup used to report the first key's location for the second component.
			spec := fmt.Sprintf("openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  %s:\n    Widget:\n      %s\n    widget:\n      %s\n", component.name, component.body, component.body)
			document, err := libopenapi.NewDocument([]byte(spec))
			require.NoError(t, err)
			defer document.Release()
			m, err := document.BuildV3Model()
			require.NoError(t, err)
			doctor := drModel.NewDrDocument(m)
			rule := buildOpenApiTestRuleAction("$", "component-description", "", nil)
			results := (ComponentDescription{}).RunRule(nil, model.RuleFunctionContext{Document: document, DrDocument: doctor, Rule: &rule})
			require.Len(t, results, 2)
			require.Equal(t, "Widget", results[0].StartNode.Value)
			require.Equal(t, "widget", results[1].StartNode.Value)
			require.NotEqual(t, results[0].StartNode.Line, results[1].StartNode.Line)
			require.Contains(t, results[0].Message, "`Widget`")
			require.Contains(t, results[1].Message, "`widget`")
			require.NotEqual(t, results[0].Path, results[1].Path)
		})
	}
}

func TestComponentDescriptionWordCountCompatibility(t *testing.T) {
	for _, tt := range []struct {
		name, description string
		min               int
		want              bool
	}{
		{"empty remains missing", "", 0, true},
		{"default minimum", "valid", 0, false},
		{"at minimum", "two words", 2, false},
		{"below minimum", "one", 2, true},
		{"repeated spaces", "two  words", 3, false},
		{"tabs are not spaces", "two\twords", 2, true},
		{"negative minimum", "valid", -1, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			spec := fmt.Sprintf("openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Example:\n      type: string\n      description: %q\n", tt.description)
			document, err := libopenapi.NewDocument([]byte(spec))
			require.NoError(t, err)
			defer document.Release()
			m, err := document.BuildV3Model()
			require.NoError(t, err)
			doctor := drModel.NewDrDocument(m)
			opts := map[string]string{"minWords": fmt.Sprint(tt.min)}
			rule := buildOpenApiTestRuleAction("$", "component-description", "", opts)
			ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), opts)
			ctx.Document = document
			ctx.DrDocument = doctor
			ctx.Rule = &rule
			results := (ComponentDescription{}).RunRule(nil, ctx)
			if tt.want {
				require.Len(t, results, 1)
			} else {
				require.Empty(t, results)
			}
		})
	}
}

func BenchmarkComponentDescriptionDescribedSchemas(b *testing.B) {
	var spec strings.Builder
	spec.WriteString("openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n")
	for i := range 1000 {
		fmt.Fprintf(&spec, "    Component%d:\n      type: string\n      description: A documented component\n", i)
	}
	document, err := libopenapi.NewDocument([]byte(spec.String()))
	require.NoError(b, err)
	defer document.Release()
	m, err := document.BuildV3Model()
	require.NoError(b, err)
	doctor := drModel.NewDrDocument(m)
	rule := buildOpenApiTestRuleAction("$", "component-description", "", nil)
	ctx := model.RuleFunctionContext{Document: document, DrDocument: doctor, Rule: &rule}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := (ComponentDescription{}).RunRule([]*yaml.Node{}, ctx)
		if len(results) != 0 {
			b.Fatal("documented components produced results")
		}
	}
}
