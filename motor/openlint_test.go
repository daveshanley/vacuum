// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/go-yaml"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestOpenLintUnresolvedFindingKeepsAuthoredReferencePath(t *testing.T) {
	raw, err := rulesets.CreateRuleSetFromData([]byte(`rules:
  description:
    resolved: false
    given: $..description
    then: {function: length, functionOptions: {max: 1}}
`), model.OpenLintV1Draft)
	require.NoError(t, err)
	rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
	result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
		RuleSet: rs, Spec: []byte(`components:
  schemas:
    Response:
      properties:
        accounts:
          items:
            $ref: '#/components/schemas/Identity'
    Identity:
      allOf:
        - $ref: '#/components/schemas/Base'
        - properties:
            owners:
              description: long
    Base: {type: object}
`), SilenceLogs: true,
	}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.Equal(t, "$.components.schemas['Identity'].allOf[1].properties['owners'].description", result.Results[0].Path)
	for _, path := range result.Results[0].Paths {
		require.Equal(t, "$.components.schemas['Identity'].allOf[1].properties['owners'].description", path)
	}
}

func TestOpenLintParserDiagnosticPaths(t *testing.T) {
	info, _, valid := parseOpenLintDocument([]byte("items:\n  - 'a.b':\n      name: first\n      name: second\n      1: value\n"))
	require.True(t, valid)
	rs := &rulesets.RuleSet{}
	results := openLintParserDiagnostics(info.RootNode, rs)
	require.Len(t, results, 2)
	require.Equal(t, "$.items[0]['a.b'].name", results[0].Path)
	require.Equal(t, "$.items[0]['a.b']['1']", results[1].Path)
	require.Equal(t, 4, results[0].StartNode.Line)
	require.Equal(t, 5, results[1].StartNode.Line)
}

func BenchmarkOpenLintParserDiagnostics(b *testing.B) {
	var source strings.Builder
	for i := 0; i < 2048; i++ {
		fmt.Fprintf(&source, "property%d: {type: string, description: text}\n", i)
	}
	info, _, valid := parseOpenLintDocument([]byte(source.String()))
	if !valid {
		b.Fatal("invalid benchmark document")
	}
	rs := &rulesets.RuleSet{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if results := openLintParserDiagnostics(info.RootNode, rs); len(results) != 0 {
			b.Fatal("unexpected parser diagnostic")
		}
	}
}

func TestOpenLintDocumentExecution(t *testing.T) {
	cases := []struct {
		name, document, rules string
		codes, paths          []string
	}{
		{"empty", "", `rules: {r: {given: '$', then: {function: truthy}}}`, []string{"r"}, []string{"$"}},
		{"array", "[]", `rules: {r: {given: '$', then: {function: falsy}}}`, []string{"r"}, []string{"$"}},
		{"keys", "paths: {'/good': {}, '/Bad': {}}", `rules: {r: {given: '$.paths', then: {field: '@key', function: pattern, functionOptions: {match: '^/[a-z]+$'}}}}`, []string{"r"}, []string{"$.paths['/Bad']"}},
		{"missing", "info: {}", `rules: {r: {given: '$.info', then: {field: description, function: defined}}}`, []string{"r"}, []string{"$.info"}},
		{"missing-wildcard", "info: {}", `rules: {r: {given: '$.info', then: {field: '$.missing[*]', function: defined}}}`, []string{"r"}, []string{"$.info"}},
		{"missing-bracket", "info: {}", `rules: {r: {given: '$.info', then: {field: '$["missing"]', function: defined}}}`, []string{"r"}, []string{"$.info"}},
		{"missing-index", "info: {values: []}", `rules: {r: {given: '$.info', then: {field: '$.values[1]', function: defined}}}`, []string{"r"}, []string{"$.info"}},
		{"ref-key", "x: {$ref: missing}", `rules: {r: {given: '$.x', resolved: false, then: {field: '$ref', function: undefined}}}`, []string{"invalid-ref", "r"}, nil},
		{"multiple-documents", "{}\n---\n{}", `rules: {r: {given: '$', then: {function: falsy}}}`, []string{"parser", "r"}, nil},
		{"duplicate", "x: 1\nx: 2", `rules: {}`, []string{"parser"}, nil},
		{"duplicate-off", "x: 1\nx: 2", "parserOptions: {duplicateKeys: off}\nrules: {}", nil, nil},
		{"nonstring-key", "1: x", `rules: {}`, []string{"parser"}, nil},
		{"reference-cycle", "defs: {A: {$ref: '#/defs/B'}, B: {$ref: '#/defs/A'}}", `rules: {}`, nil, nil},
		{"scalar", "42", `rules: {r: {given: '$', then: {function: falsy}}}`, []string{"r"}, []string{"$"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := rulesets.CreateRuleSetFromData([]byte(tc.rules), model.OpenLintV1Draft)
			require.NoError(t, err)
			rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
			require.NoError(t, rs.LoadError())
			result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte(tc.document), SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft, RunTimeout: 5 * time.Second})
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			codes := []string{}
			paths := []string{}
			for _, r := range result.Results {
				codes = append(codes, r.RuleId)
				paths = append(paths, r.Path)
			}
			require.ElementsMatch(t, tc.codes, codes)
			if tc.paths != nil {
				require.Equal(t, tc.paths, paths)
			}
		})
	}
}

func TestOpenLintUTF16Range(t *testing.T) {
	raw, err := rulesets.CreateRuleSetFromData([]byte(`rules: {b: {given: '$.b', then: {function: truthy}}}`), model.OpenLintV1Draft)
	require.NoError(t, err)
	rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
	result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte(`{"a":"😀", "b":""}`), SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft, Context: context.Background()})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	report := model.NewRuleResultSet(result.Results).GenerateSpectralReport("test.json")
	require.Equal(t, 15, report[0].Range.Start.Char)
	require.Equal(t, 17, report[0].Range.End.Char)
	encoded, err := json.Marshal(report)
	require.NoError(t, err)
	require.True(t, json.Valid(encoded))
}

func TestOpenLintOverrideParserAndFormats(t *testing.T) {
	for _, tc := range []struct {
		name, document, extra string
		codes                 []string
	}{
		{"duplicate-off", "x: 1\nx: 2", "rules: {}\n    parserOptions: {duplicateKeys: off}", nil},
		{"duplicate-warn", "x: 1\nx: 2", "rules: {}\n    parserOptions: {duplicateKeys: warn}", []string{"parser"}},
		{"override-format", "openapi: 3.1.0", "formats: [oas3]\n    rules: {r: {given: '$', then: {function: falsy}}}", []string{"r"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := rulesets.CreateRuleSetFromData([]byte("rules: {}\noverrides:\n  - files: ['**/api.yaml']\n    "+tc.extra+"\n"), model.OpenLintV1Draft)
			require.NoError(t, err)
			rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
			result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte(tc.document), SpecFileName: "/tmp/api.yaml", SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			var codes []string
			for _, r := range result.Results {
				codes = append(codes, r.RuleId)
				if r.RuleId == "parser" {
					require.Equal(t, "warn", r.Rule.Severity)
				}
			}
			require.ElementsMatch(t, tc.codes, codes)
		})
	}
}

func TestOpenLintContainerRanges(t *testing.T) {
	for _, tc := range []struct {
		source                    string
		line, start, endLine, end int
	}{
		{"x: {}", 0, 3, 0, 5},
		{"x: []", 0, 3, 0, 5},
		{`{"x":{"a":[1,2]}}`, 0, 5, 0, 16},
		{"x:\n  a: one\n  b: two\ny: end", 1, 2, 2, 8},
		{`{"x":["😀", "a\\\"b"]}`, 0, 5, 0, 21},
	} {
		t.Run(tc.source, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.source), &root))
			node := root.Content[0].Content[1]
			result := model.RuleFunctionResult{StartNode: node}
			index := newOpenLintRangeIndex([]byte(tc.source), &root)
			index.setResultRange(&result)
			require.Equal(t, reports.RangeItem{Line: tc.line, Char: tc.start}, result.Range.Start)
			require.Equal(t, reports.RangeItem{Line: tc.endLine, Char: tc.end}, result.Range.End)
		})
	}
}

func BenchmarkOpenLintResultRanges(b *testing.B) {
	source := []byte(`{"items":[` + strings.Repeat("1,", 31999) + `1]}`)
	var root yaml.Node
	if err := yaml.Unmarshal(source, &root); err != nil {
		b.Fatal(err)
	}
	nodes := root.Content[0].Content[1].Content
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		index := newOpenLintRangeIndex(source, &root)
		for _, node := range nodes {
			result := model.RuleFunctionResult{StartNode: node}
			index.setResultRange(&result)
		}
	}
}

func TestOpenLintSchemaFormatRequiresExactHost(t *testing.T) {
	for _, host := range []string{"json-schema.org", "json-schemaXorg"} {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte("$schema: https://"+host+"/draft-07/schema#"), &root))
		matches := detectOpenLintFormats(&root, []string{"jsonSchemaDraft7"})
		require.Equal(t, host == "json-schema.org", matches["jsonSchemaDraft7"])
	}
}
