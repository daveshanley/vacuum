package core

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	openapiUtils "github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestOr_Presence(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		properties  []string
		want        int
	}{
		{"neither", "type: string", []string{"title", "description"}, 1},
		{"first", "title: name", []string{"title", "description"}, 0},
		{"second", "description: text", []string{"title", "description"}, 0},
		{"both", "title: name\ndescription: text", []string{"title", "description"}, 0},
		{"last of many", "format: date", []string{"default", "pattern", "format"}, 0},
		{"null", "title: null", []string{"title", "description"}, 0},
		{"empty yaml value", "title:", []string{"title", "description"}, 0},
		{"false", "title: false", []string{"title", "description"}, 0},
		{"zero", "title: 0", []string{"title", "description"}, 0},
		{"empty string", `title: ""`, []string{"title", "description"}, 0},
		{"empty array", "title: []", []string{"title", "description"}, 0},
		{"empty object", "title: {}", []string{"title", "description"}, 0},
		{"nested only", "nested:\n  title: name", []string{"title", "description"}, 1},
		{"value only", "other: title", []string{"title", "description"}, 1},
		{"case sensitive", "Title: name", []string{"title", "description"}, 1},
		{"literal comma", "'a,b': yes", []string{"a,b", "other"}, 0},
		{"literal whitespace", "' title ': yes", []string{" title ", "other"}, 0},
		{"no trimming array names", "title: yes", []string{" title ", "other"}, 1},
		{"empty property name", "'': yes", []string{"", "other"}, 0},
		{"duplicate properties", "title: yes", []string{"title", "title"}, 0},
		{"empty object input", "{}", []string{"title", "description"}, 1},
		{"array input", "[title, description]", []string{"title", "description"}, 0},
		{"null input", "null", []string{"title", "description"}, 0},
		{"scalar input", "title", []string{"title", "description"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var doc yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.input), &doc))
			ctx := model.RuleFunctionContext{Options: map[string]any{"properties": tc.properties}, Given: "$.schema"}
			results := (Or{}).RunRule(doc.Content, ctx)
			require.Len(t, results, tc.want)
			if tc.want != 0 {
				assert.Equal(t, "$.schema", results[0].Path)
				assert.Same(t, doc.Content[0], results[0].StartNode)
				assert.NotNil(t, results[0].EndNode)
			}
		})
	}
}

func TestOr_OptionsAndMessages(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options any
		want    string
	}{
		{"array", map[string]any{"properties": []any{"title", "description"}}, `At least one of "title" or "description" must be defined`},
		{"typed array", map[string][]string{"properties": {"title", "description"}}, `At least one of "title" or "description" must be defined`},
		{"comma string", map[string]any{"properties": "title, description"}, `At least one of "title" or "description" must be defined`},
		{"typed string map", map[string]string{"properties": " title , description "}, `At least one of "title" or "description" must be defined`},
		{"four properties", map[string]any{"properties": []string{"a", "b", "c", "d"}}, `At least one of "a" or "b" or "c" or "d" must be defined`},
		{"five properties", map[string]any{"properties": []string{"a", "b", "c", "d", "e"}}, `At least one of "a" or "b" or "c" or 2 other properties must be defined`},
		{"no options", nil, orOptionsError},
		{"wrong options type", 42, orOptionsError},
		{"no properties", map[string]any{}, orOptionsError},
		{"null properties", map[string]any{"properties": nil}, orOptionsError},
		{"empty array", map[string]any{"properties": []any{}}, orOptionsError},
		{"one name", map[string]any{"properties": []any{"title"}}, orOptionsError},
		{"one string name", map[string]any{"properties": "title"}, orOptionsError},
		{"number", map[string]any{"properties": 42}, orOptionsError},
		{"mixed array", map[string]any{"properties": []any{"title", 42}}, orOptionsError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			results := (Or{}).RunRule([]*yaml.Node{{Kind: yaml.MappingNode}}, model.RuleFunctionContext{Options: tc.options})
			require.Len(t, results, 1)
			assert.Equal(t, tc.want, results[0].Message)
			assert.Equal(t, "unknown", results[0].Path)
			assert.NotNil(t, results[0].StartNode)
			assert.NotNil(t, results[0].EndNode)
		})
	}
}

func TestOr_IndependentNodesAndCustomMessage(t *testing.T) {
	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("- title: yes\n- {}\n- description: null\n- {}"), &doc))
	rule := &model.Rule{Message: "Add a title or description"}
	ctx := model.RuleFunctionContext{Rule: rule, Options: map[string]any{"properties": []string{"title", "description"}}}
	nodes := doc.Content[0].Content
	results := (Or{}).RunRule(nodes, ctx)
	require.Len(t, results, 2)
	for i, result := range results {
		assert.Equal(t, rule.Message, result.Message)
		assert.Same(t, rule, result.Rule)
		assert.Same(t, nodes[2*i+1], result.StartNode)
	}
	assert.Empty(t, (Or{}).RunRule(nil, model.RuleFunctionContext{}))
	assert.Empty(t, (Or{}).RunRule([]*yaml.Node{nil}, ctx))
	assert.Empty(t, (Or{}).RunRule([]*yaml.Node{{Kind: yaml.AliasNode}}, ctx))
}

func TestOr_Schema(t *testing.T) {
	function := Or{}
	assert.Equal(t, "or", function.GetSchema().Name)
	assert.Equal(t, model.FunctionCategoryCore, function.GetCategory())
	for _, tc := range []struct {
		options any
		valid   bool
	}{
		{map[string]any{"properties": []any{"title", "description"}}, true},
		{map[string][]string{"properties": {"title", "description"}}, true},
		{map[string]any{"properties": []string{"title", "description"}}, true},
		{map[string]any{"properties": "title, description"}, true},
		{nil, false},
		{map[string]any{"properties": []any{}}, false},
		{map[string]any{"properties": []string{"title"}}, false},
		{map[string]any{"properties": []any{"title", 42}}, false},
		{map[string]any{"properties": 42}, false},
		{map[string]any{"properties": []any{"title", "description"}, "extra": true}, false},
	} {
		valid, errs := model.ValidateRuleFunctionContextAgainstSchema(function, model.RuleFunctionContext{Options: tc.options})
		assert.Equal(t, tc.valid, valid)
		assert.Equal(t, !tc.valid, len(errs) > 0)
	}
}

func BenchmarkOr(b *testing.B) {
	for _, size := range []int{3, 30} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			node := &yaml.Node{Kind: yaml.MappingNode}
			for i := 0; i < size; i++ {
				node.Content = append(node.Content, &yaml.Node{Value: fmt.Sprint(i)}, &yaml.Node{Value: "value"})
			}
			ctx := model.RuleFunctionContext{Options: map[string]any{"properties": []string{"title", fmt.Sprint(size - 1)}}}
			nodes := []*yaml.Node{node}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				(Or{}).RunRule(nodes, ctx)
			}
		})
	}
}

func TestOr_FieldAndDocumentNodes(t *testing.T) {
	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("info:\n  version: '1.0'"), &doc))
	ctx := model.RuleFunctionContext{
		Given:      "$",
		RuleAction: &model.RuleAction{Field: "info"},
		Options:    map[string]any{"properties": []string{"title", "description"}},
	}
	result := (Or{}).RunRule([]*yaml.Node{&doc}, ctx)
	require.Len(t, result, 1)
	assert.Equal(t, "$.info", result[0].Path)
	assert.Same(t, doc.Content[0].Content[1], result[0].StartNode)
	ctx.RuleAction.Field = "missing"
	assert.Empty(t, (Or{}).RunRule([]*yaml.Node{&doc}, ctx))
}

func TestOr_ModelPathsWithoutIndex(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: Aliases
  version: '1.0'
paths: {}
components:
  schemas:
    First: &schema {type: string}
    Second: *schema
`)
	document, err := libopenapi.NewDocument(spec)
	require.NoError(t, err)
	high, err := document.BuildV3Model()
	require.NoError(t, err)
	nodes, err := openapiUtils.FindNodes(spec, "$.components.schemas.First")
	require.NoError(t, err)
	result := (Or{}).RunRule(nodes, model.RuleFunctionContext{
		DrDocument: drModel.NewDrDocument(high),
		Options:    map[string]any{"properties": []string{"title", "description"}},
	})
	require.Len(t, result, 1)
	assert.ElementsMatch(t, []string{"$.components.schemas['First']", "$.components.schemas['Second']"}, result[0].Paths)
}
