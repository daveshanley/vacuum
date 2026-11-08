// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package jsruleset

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestOpenLintFunctionPathUsesExistingAncestor(t *testing.T) {
	dir := t.TempDir()
	function := writeModule(t, dir, "check.js", `module.exports=()=>[{message:'missing root child',path:['missing','deep']},{message:'missing nested child',path:['info','absent']},{message:'existing',path:['info','title']}];`)
	prepared, err := PrepareFunctions(context.Background(), filepath.Join(dir, "rules.yaml"), map[string]string{"check": function}, os.ReadFile)
	require.NoError(t, err)
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("info: {title: ok}"), &node))
	for _, contract := range []model.Contract{model.NativeContract, model.OpenLintV1Draft} {
		scope, err := prepared.Open(context.Background(), nil)
		require.NoError(t, err)
		_, err = scope.MatchFormats(context.Background(), map[string]any{"info": map[string]any{"title": "ok"}}, "document.yaml")
		require.NoError(t, err)
		var failures []error
		rule := &model.Rule{Id: "check", Severity: model.SeverityWarn}
		results := scope.Functions()["check"].RunRule([]*yaml.Node{node.Content[0]}, model.RuleFunctionContext{Contract: contract, ExecutionRoot: &node, Rule: rule, RuleAction: &model.RuleAction{Function: "check"}, ReportError: func(err error) { failures = append(failures, err) }})
		scope.Close()
		require.Empty(t, failures)
		require.Len(t, results, 3)
		if contract == model.OpenLintV1Draft {
			require.Equal(t, "$", results[0].Path)
			require.Equal(t, "$.info", results[1].Path)
		} else {
			require.Equal(t, "$.missing.deep", results[0].Path)
			require.Equal(t, "$.info.absent", results[1].Path)
		}
		require.Equal(t, "$.info.title", results[2].Path)
	}
}

func TestOpenLintMissingFunctionFieldKeepsParentPath(t *testing.T) {
	dir := t.TempDir()
	function := writeModule(t, dir, "check.js", `module.exports=(value,options,context)=>[{message:typeof value+':'+JSON.stringify(context.path)}];`)
	prepared, err := PrepareFunctions(context.Background(), filepath.Join(dir, "rules.yaml"), map[string]string{"check": function}, os.ReadFile)
	require.NoError(t, err)
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("info: {}"), &document))
	for _, contract := range []model.Contract{model.NativeContract, model.OpenLintV1Draft} {
		scope, err := prepared.Open(context.Background(), nil)
		require.NoError(t, err)
		_, err = scope.MatchFormats(context.Background(), &document, "input.yaml")
		require.NoError(t, err)
		var failures []error
		results := scope.Functions()["check"].RunRule([]*yaml.Node{document.Content[0].Content[1]}, model.RuleFunctionContext{Contract: contract, ExecutionRoot: &document, Rule: &model.Rule{Id: "check", Severity: model.SeverityWarn}, RuleAction: &model.RuleAction{Function: "check", Field: "description"}, ReportError: func(err error) { failures = append(failures, err) }})
		scope.Close()
		require.Empty(t, failures)
		require.Len(t, results, 1)
		if contract == model.OpenLintV1Draft {
			require.Equal(t, `undefined:["info"]`, results[0].Message)
			require.Equal(t, "$.info", results[0].Path)
		} else {
			require.Equal(t, `undefined:["info","description"]`, results[0].Message)
			require.Equal(t, "$.info.description", results[0].Path)
		}
	}
}
