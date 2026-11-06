// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	jsplugin "github.com/daveshanley/vacuum/plugin/javascript"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/require"
)

func originTestIndex(t testing.TB, spec, location string) *index.SpecIndex {
	t.Helper()
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &root))
	config := index.CreateClosedAPIIndexConfig()
	config.SpecAbsolutePath = location
	idx := index.NewSpecIndexWithConfig(&root, config)
	t.Cleanup(idx.Release)
	return idx
}

func originTestRolodex(t testing.TB, root, child *index.SpecIndex) *index.Rolodex {
	t.Helper()
	rolodex := index.NewRolodex(index.CreateClosedAPIIndexConfig())
	rolodex.SetRootIndex(root)
	rolodex.SetRootNode(root.GetRootNode())
	rolodex.AddIndex(child)
	t.Cleanup(rolodex.Release)
	return rolodex
}

func TestPopulateResultOriginsPreservesLocations(t *testing.T) {
	dir := t.TempDir()
	root := originTestIndex(t, "info:\n  title: root\n", filepath.Join(dir, "root.yaml"))
	child := originTestIndex(t, "info:\n  title: child\n", filepath.Join(dir, "child.yaml"))
	rolodex := originTestRolodex(t, root, child)
	rootTitle := root.GetRootNode().Content[0].Content[1].Content[1]
	childTitle := child.GetRootNode().Content[0].Content[1].Content[1]
	rootCopy, childCopy := *rootTitle, *childTitle
	missing := rootCopy
	missing.Value = "not in either document"
	for _, tc := range []struct {
		name     string
		node     *yaml.Node
		location string
	}{
		{"root_owner", rootTitle, "input.yaml"},
		{"child_owner", childTitle, child.GetSpecAbsolutePath()},
		{"root_fallback", &rootCopy, "input.yaml"},
		{"child_fallback", &childCopy, child.GetSpecAbsolutePath()},
		{"missing_fallback", &missing, ""},
		{"document_root", root.GetRootNode(), ""},
		{"nil_node", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			results := []model.RuleFunctionResult{{StartNode: tc.node}, {StartNode: tc.node}}
			populateResultOrigins(results, rolodex, nil, "input.yaml")
			for _, result := range results {
				if tc.location == "" {
					require.Nil(t, result.Origin)
					continue
				}
				require.NotNil(t, result.Origin)
				require.Equal(t, tc.location, result.Origin.AbsoluteLocation)
				require.Equal(t, tc.node.Line, result.Origin.Line)
				require.Equal(t, tc.node.Column, result.Origin.Column)
			}
			if tc.location != "" {
				results[0].Origin.AbsoluteLocation = "changed.yaml"
				require.Equal(t, tc.location, results[1].Origin.AbsoluteLocation)
			}
		})
	}

	existing := &index.NodeOrigin{AbsoluteLocation: "explicit.yaml", Line: 12, Column: 3}
	results := []model.RuleFunctionResult{{StartNode: &rootCopy, Origin: existing}}
	populateResultOrigins(results, rolodex, rolodex, "input.yaml")
	require.Same(t, existing, results[0].Origin)

	// A miss in one execution must not hide a later match for the same pointer.
	results = []model.RuleFunctionResult{{StartNode: &missing}}
	populateResultOrigins(results, rolodex, rolodex, "input.yaml")
	require.Nil(t, results[0].Origin)
	missing.Value = rootCopy.Value
	populateResultOrigins(results, rolodex, rolodex, "second.yaml")
	require.NotNil(t, results[0].Origin)
	require.Equal(t, "second.yaml", results[0].Origin.AbsoluteLocation)
}

func TestIssue1045RootJavaScriptFindings(t *testing.T) {
	dir := t.TempDir()
	const spec = `openapi: 3.0.3
info: {title: Test, version: '1'}
paths:
  /first:
    get:
      responses:
        '200': {$ref: './response.yaml'}
  /second:
    get:
      responses:
        '200': {$ref: './response.yaml'}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "response.yaml"), []byte("description: OK\n"), 0o600))
	for _, withPath := range []bool{false, true} {
		t.Run(fmt.Sprintf("with_path=%t", withPath), func(t *testing.T) {
			pathField := ""
			if withPath {
				pathField = `, path: "$.paths['" + p + "']"`
			}
			function := jsplugin.NewJSRuleFunction("rootFinding", `
function getSchema() { return {name: 'rootFinding'}; }
function runRule(input) {
  return Object.keys(input.paths).map(function(p) { return {message: 'finding for ' + p`+pathField+`}; });
}`)
			rule := &model.Rule{Id: "root-finding", Given: "$", Resolved: true, Severity: model.SeverityError,
				Then: model.RuleAction{Function: "rootFinding"}}
			result := ApplyRulesToRuleSet(&RuleSetExecution{
				RuleSet:         &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}},
				Spec:            []byte(spec),
				SpecFileName:    filepath.Join(dir, "openapi.yaml"),
				Base:            dir,
				AllowLookup:     true,
				SilenceLogs:     true,
				CustomFunctions: map[string]model.RuleFunction{"rootFinding": function},
			})
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 2)
			for _, finding := range result.Results {
				path := strings.TrimPrefix(finding.Message, "finding for ")
				require.Contains(t, []string{"/first", "/second"}, path)
				expected := "$"
				if withPath {
					expected = "$.paths['" + path + "']"
				}
				require.Equal(t, expected, finding.Path)
				require.Nil(t, finding.Origin)
			}
		})
	}
}

func BenchmarkPopulateResultOriginsRepeatedRoot(b *testing.B) {
	// A wide document with few root fields reproduces the uncached root hash
	// used by a JavaScript rule with `given: $`, without network dependencies.
	const pathCount = 208
	var spec strings.Builder
	spec.WriteString("openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths:\n")
	for i := 0; i < pathCount; i++ {
		fmt.Fprintf(&spec, "  /path-%d:\n    get:\n      description: %s\n      responses: {'200': {description: OK}}\n", i, strings.Repeat("x", 1024))
	}
	root := originTestIndex(b, spec.String(), filepath.Join(b.TempDir(), "root.yaml"))
	child := originTestIndex(b, "description: OK\n", filepath.Join(b.TempDir(), "child.yaml"))
	rolodex := originTestRolodex(b, root, child)
	for _, count := range []int{1, pathCount} {
		b.Run(fmt.Sprintf("findings=%d", count), func(b *testing.B) {
			results := make([]model.RuleFunctionResult, count)
			for i := range results {
				results[i].StartNode = root.GetRootNode()
			}
			b.ReportAllocs()
			for b.Loop() {
				populateResultOrigins(results, rolodex, rolodex, "input.yaml")
			}
		})
	}
}
