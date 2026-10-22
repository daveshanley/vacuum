// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func arazzoExecution(t testing.TB, fixture string) *RuleSetExecution {
	t.Helper()
	filename := filepath.Join("..", "model", "test_files", "arazzo", fixture)
	spec, err := os.ReadFile(filename)
	require.NoError(t, err)
	return &RuleSetExecution{Spec: spec, SpecFileName: filename, AllowLookup: true,
		RuleSet: rulesets.BuildDefaultRuleSets().GenerateArazzoRecommendedRuleSet(), SilenceLogs: true}
}

func TestArazzoLintVersionsAndSources(t *testing.T) {
	for _, fixture := range []string{"workflow.yaml", "mixed.yaml"} {
		for _, jsonInput := range []bool{false, true} {
			t.Run(fixture+map[bool]string{true: "/json", false: "/yaml"}[jsonInput], func(t *testing.T) {
				execution := arazzoExecution(t, fixture)
				if jsonInput {
					var document any
					require.NoError(t, yaml.Unmarshal(execution.Spec, &document))
					var err error
					execution.Spec, err = json.Marshal(document)
					require.NoError(t, err)
				}
				result := ApplyRulesToRuleSet(execution)
				require.Empty(t, result.Errors)
				require.Empty(t, result.Results)
				require.NotNil(t, result.Arazzo)
				require.True(t, result.Arazzo.Validation.Complete)
				require.Equal(t, "arazzo", result.SpecInfo.SpecType)
				require.Nil(t, result.RuleSetExecution.Document)
				require.Nil(t, result.RuleSetExecution.DrDocument)
				if fixture == "mixed.yaml" {
					require.Equal(t, 3, result.FilesProcessed)
				} else {
					require.Equal(t, 1, result.FilesProcessed)
				}
			})
		}
	}
}

func TestArazzoFindingsHaveExactPathsAndConfigurableSeverity(t *testing.T) {
	execution := arazzoExecution(t, "workflow.yaml")
	execution.Spec = []byte(strings.ReplaceAll(string(execution.Spec), "$sourceDescriptions.api.read", "$sourceDescriptions.api.missing"))
	execution.RuleSet.Rules["arazzo-reference"].Severity = model.SeverityWarn
	result := ApplyRulesToRuleSet(execution)
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	finding := result.Results[0]
	require.Equal(t, "arazzo-reference", finding.RuleId)
	require.Equal(t, model.SeverityWarn, finding.Rule.Severity)
	require.Equal(t, "$.workflows[0].steps[0].operationId", finding.Path)
	require.Equal(t, []string{finding.Path}, finding.Paths)
	require.Equal(t, 18, finding.StartNode.Line)
	execution = arazzoExecution(t, "workflow.yaml")
	execution.Spec = []byte(strings.ReplaceAll(string(execution.Spec), "$sourceDescriptions.api.read", "$sourceDescriptions.api.missing"))
	delete(execution.RuleSet.Rules, "arazzo-reference")
	result = ApplyRulesToRuleSet(execution)
	require.Empty(t, result.Errors)
	require.Empty(t, result.Results)
}

func TestArazzoIncompleteAndInputErrors(t *testing.T) {
	execution := arazzoExecution(t, "workflow.yaml")
	execution.AllowLookup = false
	result := ApplyRulesToRuleSet(execution)
	require.Empty(t, result.Errors)
	require.NotEmpty(t, result.Results)
	for _, finding := range result.Results {
		require.Equal(t, rulesets.ArazzoValidationIncomplete, finding.RuleId)
	}
	require.False(t, result.Arazzo.Validation.Complete)
	for _, spec := range []string{"arazzo: 2.0.0\n", "arazzo: [\n", "arazzo: 1.0.1\n---\narazzo: 1.0.1\n"} {
		execution = arazzoExecution(t, "workflow.yaml")
		execution.Spec = []byte(spec)
		require.NotEmpty(t, ApplyRulesToRuleSet(execution).Errors)
	}
	execution = arazzoExecution(t, "workflow.yaml")
	execution.Base = t.TempDir()
	require.NotEmpty(t, ApplyRulesToRuleSet(execution).Errors, "an enabled lookup failure is a tool error")
}

func TestArazzoCustomRulesAndCancellation(t *testing.T) {
	execution := arazzoExecution(t, "workflow.yaml")
	execution.RuleSet = &rulesets.RuleSet{Rules: map[string]*model.Rule{
		"custom":       {Id: "custom", Severity: model.SeverityError, Formats: []string{model.Arazzo}, Given: "$.info", Then: model.RuleAction{Field: "title", Function: "falsy"}},
		"openapi-only": {Id: "openapi-only", Severity: model.SeverityError, Formats: model.OAS3Format, Given: "$", Then: model.RuleAction{Function: "falsy"}},
	}}
	result := ApplyRulesToRuleSet(execution)
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.Equal(t, "custom", result.Results[0].RuleId)
	require.Nil(t, result.Arazzo.Validation, "custom rules do not trigger source validation")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result = ApplyRulesToRuleSetWithOptions(arazzoExecution(t, "workflow.yaml"), &ExecutionOptions{Context: ctx})
	require.ErrorIs(t, result.Errors[0], context.Canceled)
}

func TestArazzoInputPropertyPathsRemainAuthored(t *testing.T) {
	for _, test := range []struct {
		action model.RuleAction
		path   string
	}{
		{model.RuleAction{Field: "type", Function: "falsy"}, "$.workflows[0].inputs.properties.name.type"},
		{model.RuleAction{Field: "description", Function: "truthy"}, "$.workflows[0].inputs.properties.name.description"},
		{model.RuleAction{Field: "type", Function: "pattern", FunctionOptions: map[string]interface{}{"notMatch": "string"}}, "$.workflows[0].inputs.properties.name.type"},
	} {
		t.Run(test.action.Function, func(t *testing.T) {
			execution := arazzoExecution(t, "workflow.yaml")
			execution.Spec = []byte(strings.Replace(string(execution.Spec), "    steps:", "    inputs:\n      type: object\n      properties:\n        name:\n          type: string\n    steps:", 1))
			execution.RuleSet = &rulesets.RuleSet{Rules: map[string]*model.Rule{
				"custom": {Id: "custom", Severity: model.SeverityError, Formats: []string{model.Arazzo}, Given: "$.workflows[*].inputs.properties.*", Then: test.action},
			}}
			result := ApplyRulesToRuleSet(execution)
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 1)
			require.Equal(t, test.path, result.Results[0].Path)
		})
	}
}

func BenchmarkArazzoLint(b *testing.B) {
	execution := arazzoExecution(b, "workflow.yaml")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		result := ApplyRulesToRuleSet(execution)
		if len(result.Errors) != 0 || len(result.Results) != 0 {
			b.Fatalf("incomplete lint: %v %v", result.Errors, result.Results)
		}
	}
}

func TestArazzoAuthoringPathsKeepIgnoresWithinOneWorkflow(t *testing.T) {
	execution := arazzoExecution(t, "workflow.yaml")
	spec := strings.Replace(string(execution.Spec), "    description: Retrieve a pet.\n", "", 1)
	start := strings.Index(spec, "  - workflowId:")
	workflow := spec[start:]
	ignored := strings.Replace(workflow, "    summary:", "    x-lint-ignore: [arazzo-workflow-description]\n    summary:", 1)
	second := strings.Replace(workflow, "workflowId: readPet", "workflowId: anotherPet", 1)
	execution.Spec = []byte(spec[:start] + ignored + second)
	result := ApplyRulesToRuleSet(execution)
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.Equal(t, "arazzo-workflow-description", result.Results[0].RuleId)
	require.Equal(t, "$.workflows[1].description", result.Results[0].Path)
	require.Len(t, result.IgnoredResults, 1)
	require.Equal(t, "arazzo-workflow-description", result.IgnoredResults[0].RuleId)
}

func TestArazzoValidatorFindingsHonorInlineIgnore(t *testing.T) {
	execution := arazzoExecution(t, "workflow.yaml")
	spec := strings.ReplaceAll(string(execution.Spec), "$sourceDescriptions.api.read", "$sourceDescriptions.api.missing")
	spec = strings.Replace(spec, "        operationId:", "        x-lint-ignore: [arazzo-reference]\n        operationId:", 1)
	execution.Spec = []byte(spec)
	result := ApplyRulesToRuleSet(execution)
	require.Empty(t, result.Errors)
	require.Empty(t, result.Results)
	require.Len(t, result.IgnoredResults, 1)
	require.Equal(t, "$.workflows[0].steps[0].operationId", result.IgnoredResults[0].Path)
}

func TestArazzoMarkdownPathsKeepIgnoresWithinTheirObject(t *testing.T) {
	execution := arazzoExecution(t, "workflow.yaml")
	spec := strings.Replace(string(execution.Spec), "description: Retrieve a pet.", "description: '<script>alert(1)</script>'", 1)
	spec = strings.Replace(spec, "info:\n", "info:\n  x-lint-ignore: [arazzo-no-script-tags-in-markdown]\n", 1)
	execution.Spec = []byte(spec)
	result := ApplyRulesToRuleSet(execution)
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.Equal(t, "arazzo-no-script-tags-in-markdown", result.Results[0].RuleId)
	require.Equal(t, "$.workflows[0].description", result.Results[0].Path)
}

func TestArazzoSourceIgnoresUseOwningDocument(t *testing.T) {
	root := `arazzo: 1.1.0
info: {title: Root, version: '1', description: Root workflow., summary: Root.}
sourceDescriptions: [{name: other, url: other.yaml, type: arazzo}]
workflows:
  - workflowId: run
    description: Run the source.
    summary: Run.
    steps: [{stepId: call, description: Call., workflowId: $sourceDescriptions.other.remote, successCriteria: [{condition: "true", x-lint-ignore: [arazzo-reference]}]}]
`
	child := `arazzo: 1.1.0
info: {title: Other, version: '1'}
sourceDescriptions: []
workflows:
  - workflowId: remote
    steps: [{stepId: call, workflowId: absent, successCriteria: [{condition: "$steps.missing.outputs.value == 2"}]}]
`
	for _, sourceIgnore := range []bool{false, true} {
		t.Run(map[bool]string{false: "root ignore cannot hide source", true: "source ignore retains origin"}[sourceIgnore], func(t *testing.T) {
			dir := t.TempDir()
			sourcePath := filepath.Join(dir, "other.yaml")
			source := child
			if sourceIgnore {
				source = strings.Replace(source, `condition: "$steps.missing.outputs.value == 2"`, `condition: "$steps.missing.outputs.value == 2", x-lint-ignore: [arazzo-reference]`, 1)
			}
			require.NoError(t, os.WriteFile(sourcePath, []byte(source), 0600))
			execution := &RuleSetExecution{Spec: []byte(root), SpecFileName: filepath.Join(dir, "main.yaml"), AllowLookup: true,
				RuleSet: rulesets.BuildDefaultRuleSets().GenerateArazzoRecommendedRuleSet(), SilenceLogs: true}
			result := ApplyRulesToRuleSet(execution)
			require.Empty(t, result.Errors)
			findings := result.Results
			if sourceIgnore {
				require.Empty(t, findings)
				findings = result.IgnoredResults
			} else {
				require.Empty(t, result.IgnoredResults)
			}
			require.Len(t, findings, 1)
			require.Equal(t, "$.workflows[0].steps[0].successCriteria[0].condition", findings[0].Path)
			require.Equal(t, "arazzo-reference", findings[0].RuleId)
			require.NotNil(t, findings[0].Origin)
			require.Equal(t, sourcePath, findings[0].Origin.AbsoluteLocation)
		})
	}
}

func TestArazzoOperationPathHintKeepsExactPathAndIgnoreScope(t *testing.T) {
	execution := arazzoExecution(t, "workflow.yaml")
	spec := strings.Replace(string(execution.Spec), "operationId: $sourceDescriptions.api.read", "operationPath: '{$sourceDescriptions.api.url}#/paths/~1pets/get'", 1)
	// Put the ignore on the first step, then use the same operation on another.
	start := strings.Index(spec, "      - stepId:")
	step := spec[start:]
	first := strings.Replace(step, "        description:", "        x-lint-ignore: [arazzo-step-operationPath]\n        description:", 1)
	second := strings.Replace(step, "stepId: read", "stepId: readAgain", 1)
	execution.Spec = []byte(spec[:start] + first + second)
	all := rulesets.BuildDefaultRuleSets().GenerateArazzoDefaultRuleSet()
	execution.RuleSet.Rules["arazzo-step-operationPath"] = all.Rules["arazzo-step-operationPath"]
	result := ApplyRulesToRuleSet(execution)
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.Equal(t, "arazzo-step-operationPath", result.Results[0].RuleId)
	require.Equal(t, "$.workflows[0].steps[1].operationPath", result.Results[0].Path)
	require.Len(t, result.IgnoredResults, 1)
}

func TestArazzoRedirectedSourcesUseFinalRetrievalBase(t *testing.T) {
	var leafRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alias/flows.yaml":
			http.Redirect(w, r, "/v1/flows.yaml", http.StatusFound)
		case "/v1/flows.yaml":
			fmt.Fprint(w, "arazzo: 1.1.0\ninfo: {title: Other, version: '1'}\nsourceDescriptions: [{name: leaf, url: leaf.yaml, type: arazzo}]\nworkflows: [{workflowId: run, dependsOn: [$sourceDescriptions.leaf.run], steps: [{stepId: read, workflowId: $sourceDescriptions.leaf.run}]}]\n")
		case "/v1/leaf.yaml":
			leafRequests.Add(1)
			fmt.Fprint(w, "arazzo: 1.1.0\ninfo: {title: Leaf, version: '1'}\nsourceDescriptions: []\nworkflows: [{workflowId: run, steps: [{stepId: read, operationId: read}]}]\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	execution := arazzoExecution(t, "workflow.yaml")
	execution.SpecFileName = server.URL + "/main.yaml"
	execution.Spec = []byte("arazzo: 1.1.0\ninfo: {title: Root, version: '1', description: Root., summary: Root.}\nsourceDescriptions: [{name: other, url: alias/flows.yaml, type: arazzo}]\nworkflows: [{workflowId: root, description: Run., summary: Run., steps: [{stepId: read, description: Read., workflowId: $sourceDescriptions.other.run}]}]\n")
	result := ApplyRulesToRuleSet(execution)
	require.Empty(t, result.Errors)
	require.Equal(t, int32(1), leafRequests.Load())
	require.Contains(t, result.Arazzo.Sources, server.URL+"/v1/flows.yaml")
	require.Contains(t, result.Arazzo.Sources, server.URL+"/v1/leaf.yaml")
}
