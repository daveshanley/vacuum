// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestIssue879AliasedResultPathsAreCompleteAndStable(t *testing.T) {
	dir, specPath, specBytes := writeIssue879AliasedResponseFixture(t)

	rule := &model.Rule{
		Id:          "check-string-attribute-minlength",
		Description: "check string attribute minLength",
		Message:     "string minLength must be at least 1",
		Given:       "$.paths[*][*].responses['400'].content['*/*'].schema.properties.error",
		Resolved:    true,
		Severity:    model.SeverityError,
		Then: &model.RuleAction{
			Field:    "minLength",
			Function: "length",
			FunctionOptions: map[string]interface{}{
				"min": 1,
			},
		},
	}
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}

	expectedPaths := []string{
		"$.paths['/v1/bar'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/bar'].post.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/baz'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/baz'].post.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/foo'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/foo'].post.responses['400'].content['*/*'].schema.properties['error']",
	}

	for i := 0; i < 30; i++ {
		results := ApplyRulesToRuleSet(&RuleSetExecution{
			RuleSet:           ruleSet,
			Spec:              specBytes,
			SpecFileName:      specPath,
			Base:              dir,
			AllowLookup:       true,
			NodeLookupTimeout: 5 * time.Second,
			SilenceLogs:       true,
		})

		require.Empty(t, results.Errors, "iteration %d", i)
		if assert.Len(t, results.Results, 1, "iteration %d", i) {
			assert.Equal(t, expectedPaths[0], results.Results[0].Path, "iteration %d", i)
			assert.Equal(t, expectedPaths, results.Results[0].Paths, "iteration %d", i)
		}
	}
}

func TestIssue879MissingExampleSharedResponsePathsAreCompleteAndStable(t *testing.T) {
	dir, specPath, specBytes := writeIssue879MissingExampleResponseFixture(t)

	rule := rulesets.GetOAS3ExamplesMissingRule()
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}

	expectedPaths := []string{
		"$.paths['/v1/resource'].get.responses['400'].content['*/*'].schema.properties['error-code']",
		"$.paths['/v1/resource'].get.responses['404'].content['*/*'].schema.properties['error-code']",
		"$.paths['/v1/resource'].get.responses['500'].content['*/*'].schema.properties['error-code']",
	}

	for i := 0; i < 100; i++ {
		results := ApplyRulesToRuleSet(&RuleSetExecution{
			RuleSet:           ruleSet,
			Spec:              specBytes,
			SpecFileName:      specPath,
			Base:              dir,
			AllowLookup:       true,
			NodeLookupTimeout: 5 * time.Second,
			SilenceLogs:       true,
		})

		require.Empty(t, results.Errors, "iteration %d", i)

		var exampleResults []model.RuleFunctionResult
		for _, result := range results.Results {
			if result.RuleId == rulesets.Oas3ExampleMissingCheck &&
				strings.Contains(result.Message, "`error-code`") {
				exampleResults = append(exampleResults, result)
			}
		}

		if assert.Len(t, exampleResults, 1, "iteration %d", i) {
			assert.Equal(t, expectedPaths[0], exampleResults[0].Path, "iteration %d", i)
			assert.Equal(t, expectedPaths, exampleResults[0].Paths, "iteration %d", i)
		}
	}
}

func TestIssue879AliasedResultPathsSupportUnquotedKeyUnion(t *testing.T) {
	dir, specPath, specBytes := writeIssue879AliasedResponseFixture(t)

	rule := &model.Rule{
		Id:          "check-string-attribute-minlength",
		Description: "check string attribute minLength",
		Message:     "string minLength must be at least 1",
		Given:       "$.paths[*][get,post].responses['400'].content['*/*'].schema.properties.error",
		Resolved:    true,
		Severity:    model.SeverityError,
		Then: &model.RuleAction{
			Field:    "minLength",
			Function: "length",
			FunctionOptions: map[string]interface{}{
				"min": 1,
			},
		},
	}
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}

	expectedPaths := []string{
		"$.paths['/v1/bar'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/bar'].post.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/baz'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/baz'].post.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/foo'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/foo'].post.responses['400'].content['*/*'].schema.properties['error']",
	}

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:           ruleSet,
		Spec:              specBytes,
		SpecFileName:      specPath,
		Base:              dir,
		AllowLookup:       true,
		NodeLookupTimeout: 5 * time.Second,
		SilenceLogs:       true,
	})

	require.Empty(t, results.Errors)
	if assert.Len(t, results.Results, 1) {
		assert.Equal(t, expectedPaths[0], results.Results[0].Path)
		assert.Equal(t, expectedPaths, results.Results[0].Paths)
	}
}

func TestIssue879AliasedResultPathsSupportQuotedKeyUnion(t *testing.T) {
	dir, specPath, specBytes := writeIssue879AliasedResponseFixture(t)

	rule := &model.Rule{
		Id:          "check-string-attribute-minlength",
		Description: "check string attribute minLength",
		Message:     "string minLength must be at least 1",
		Given:       "$.paths[*]['get','post'].responses['400'].content['*/*'].schema.properties.error",
		Resolved:    true,
		Severity:    model.SeverityError,
		Then: &model.RuleAction{
			Field:    "minLength",
			Function: "length",
			FunctionOptions: map[string]interface{}{
				"min": 1,
			},
		},
	}
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}

	expectedPaths := []string{
		"$.paths['/v1/bar'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/bar'].post.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/baz'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/baz'].post.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/foo'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/foo'].post.responses['400'].content['*/*'].schema.properties['error']",
	}

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:           ruleSet,
		Spec:              specBytes,
		SpecFileName:      specPath,
		Base:              dir,
		AllowLookup:       true,
		NodeLookupTimeout: 5 * time.Second,
		SilenceLogs:       true,
	})

	require.Empty(t, results.Errors)
	if assert.Len(t, results.Results, 1) {
		assert.Equal(t, expectedPaths[0], results.Results[0].Path)
		assert.Equal(t, expectedPaths, results.Results[0].Paths)
	}
}

func TestCollectResultPathCandidatesSupportsQuotedKeyUnion(t *testing.T) {
	root := testResultPathDocumentNode(testResultPathMappingNode(
		"paths", testResultPathMappingNode(
			"/v1/foo", testResultPathMappingNode(
				"get", &yaml.Node{Kind: yaml.MappingNode, Line: 10, Column: 3},
				"post", &yaml.Node{Kind: yaml.MappingNode, Line: 20, Column: 3},
				"put", &yaml.Node{Kind: yaml.MappingNode, Line: 30, Column: 3},
			),
		),
	))

	candidates, truncated := collectResultPathCandidates(root, `$.paths[*]["get", "post"]`)

	assert.False(t, truncated)
	assert.Equal(t, []string{
		"$.paths['/v1/foo'].get",
		"$.paths['/v1/foo'].post",
	}, resultPathCandidatePaths(candidates))
}

func TestParseResultPathStepsRejectsMalformedQuotedKeyUnion(t *testing.T) {
	_, ok := parseResultPathSteps("$.paths[*]['get',].responses")
	assert.False(t, ok)
}

func TestNeedsAliasedResultPathCompletion(t *testing.T) {
	rule := &model.Rule{Id: "shared-schema", Given: "$.paths[*][*].schema"}
	clean := []model.RuleFunctionResult{
		{
			Rule:      rule,
			RuleId:    rule.Id,
			Path:      "$.paths['/v1/foo'].get.schema",
			StartNode: &yaml.Node{Kind: yaml.MappingNode, Line: 10, Column: 3},
		},
	}
	needsCompletion := []model.RuleFunctionResult{
		{
			Rule:      rule,
			RuleId:    rule.Id,
			Path:      "unknown",
			StartNode: &yaml.Node{Kind: yaml.MappingNode, Line: 10, Column: 3},
		},
	}

	assert.False(t, needsAliasedResultPathCompletion(clean))
	assert.True(t, needsAliasedResultPathCompletion(needsCompletion))
}

func TestWalkResultPathCandidatesStopsAtLimit(t *testing.T) {
	candidates := make([]resultPathCandidate, maxResultPathCandidates)
	root := &yaml.Node{Kind: yaml.MappingNode, Line: 10, Column: 3}

	ok := walkResultPathCandidates(root, "$", nil, &candidates)

	assert.False(t, ok)
	assert.Len(t, candidates, maxResultPathCandidates)
}

func TestCompleteAliasedResultPathsMergesUnknownPaths(t *testing.T) {
	sharedSchema := &yaml.Node{Kind: yaml.MappingNode, Line: 42, Column: 7}
	root := testResultPathDocumentNode(testResultPathMappingNode(
		"paths", testResultPathMappingNode(
			"/v1/foo", testResultPathMappingNode(
				"get", testResultPathMappingNode(
					"schema", sharedSchema,
				),
			),
			"/v1/bar", testResultPathMappingNode(
				"post", testResultPathMappingNode(
					"schema", sharedSchema,
				),
			),
		),
	))
	rule := &model.Rule{
		Id:    "shared-schema",
		Given: "$.paths[*][*].schema",
	}
	results := []model.RuleFunctionResult{
		{
			Rule:      rule,
			RuleId:    rule.Id,
			Message:   "schema issue",
			Path:      "unknown",
			StartNode: sharedSchema,
		},
	}

	completeAliasedResultPathsFromGiven(results, root, nil, nil)

	expectedPaths := []string{
		"$.paths['/v1/bar'].post.schema",
		"$.paths['/v1/foo'].get.schema",
	}
	assert.Equal(t, expectedPaths[0], results[0].Path)
	assert.Equal(t, expectedPaths, results[0].Paths)
}

func TestCompleteAliasedResultPathsExpandsGivenAliases(t *testing.T) {
	sharedOperation := &yaml.Node{Kind: yaml.MappingNode, Line: 42, Column: 7}
	root := testResultPathDocumentNode(testResultPathMappingNode(
		"paths", testResultPathMappingNode(
			"/v1/foo", testResultPathMappingNode(
				"get", sharedOperation,
			),
			"/v1/bar", testResultPathMappingNode(
				"post", sharedOperation,
			),
		),
	))
	rule := &model.Rule{
		Id:    "shared-operation",
		Given: "#Operations",
	}
	results := []model.RuleFunctionResult{
		{
			Rule:      rule,
			RuleId:    rule.Id,
			Message:   "operation issue",
			Path:      "unknown",
			StartNode: sharedOperation,
		},
	}

	completeAliasedResultPathsFromGiven(results, root, nil, map[string][]string{
		"Operations": {"$.paths[*][get,post]"},
	})

	expectedPaths := []string{
		"$.paths['/v1/bar'].post",
		"$.paths['/v1/foo'].get",
	}
	assert.Equal(t, expectedPaths[0], results[0].Path)
	assert.Equal(t, expectedPaths, results[0].Paths)
}

func TestResultPathCandidateIndexMatchesByNodeAndPosition(t *testing.T) {
	nodeMatch := &yaml.Node{Kind: yaml.MappingNode, Line: 10, Column: 2}
	positionMatch := &yaml.Node{Kind: yaml.MappingNode, Line: 20, Column: 4}
	other := &yaml.Node{Kind: yaml.MappingNode, Line: 30, Column: 6}
	candidateIndex := newResultPathCandidateIndex([]resultPathCandidate{
		{path: "$.paths['/v1/foo'].get", node: nodeMatch},
		{path: "$.paths['/v1/bar'].post", node: positionMatch},
		{path: "$.paths['/v1/baz'].put", node: other},
	})

	nodePaths := candidateIndex.matchingPaths(&model.RuleFunctionResult{
		StartNode: nodeMatch,
	}, nil, nil)
	assert.Equal(t, []string{"$.paths['/v1/foo'].get"}, nodePaths)

	positionPaths := candidateIndex.matchingPaths(&model.RuleFunctionResult{
		StartNode: &yaml.Node{Kind: yaml.MappingNode, Line: 20, Column: 4},
	}, nil, nil)
	assert.Equal(t, []string{"$.paths['/v1/bar'].post"}, positionPaths)
}

func resultPathCandidatePaths(candidates []resultPathCandidate) []string {
	paths := make([]string, len(candidates))
	for i := range candidates {
		paths[i] = candidates[i].path
	}
	return paths
}

func writeIssue879AliasedResponseFixture(t *testing.T) (string, string, []byte) {
	t.Helper()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi-test.yaml")
	commonPath := filepath.Join(dir, "common-responses.yaml")

	require.NoError(t, os.WriteFile(commonPath, []byte(`BadRequest:
  description: bad request
  content:
    '*/*':
      schema:
        type: object
        properties:
          error:
            type: string
            minLength: 0
`), 0644))

	specBytes := []byte(`openapi: 3.0.3
info:
  title: Vacuum issue 879 repro
  version: 1.0.0
paths:
  /v1/foo:
    get:
      responses:
        '400':
          $ref: './common-responses.yaml#/BadRequest'
    post:
      responses:
        '400':
          $ref: './common-responses.yaml#/BadRequest'
  /v1/bar:
    get:
      responses:
        '400':
          $ref: './common-responses.yaml#/BadRequest'
    post:
      responses:
        '400':
          $ref: './common-responses.yaml#/BadRequest'
  /v1/baz:
    get:
      responses:
        '400':
          $ref: './common-responses.yaml#/BadRequest'
    post:
      responses:
        '400':
          $ref: './common-responses.yaml#/BadRequest'
components:
  schemas: {}
`)
	require.NoError(t, os.WriteFile(specPath, specBytes, 0644))
	return dir, specPath, specBytes
}

func writeIssue879MissingExampleResponseFixture(t *testing.T) (string, string, []byte) {
	t.Helper()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi-test.yaml")
	commonPath := filepath.Join(dir, "common-responses.yaml")

	require.NoError(t, os.WriteFile(commonPath, []byte(`ErrorResponse:
  description: error response
  content:
    '*/*':
      schema:
        type: object
        properties:
          error-code:
            type: string
`), 0644))

	specBytes := []byte(`openapi: 3.0.3
info:
  title: Vacuum issue 879 missing example repro
  version: 1.0.0
paths:
  /v1/resource:
    get:
      responses:
        '400':
          $ref: './common-responses.yaml#/ErrorResponse'
        '404':
          $ref: './common-responses.yaml#/ErrorResponse'
        '500':
          $ref: './common-responses.yaml#/ErrorResponse'
components:
  schemas: {}
`)
	require.NoError(t, os.WriteFile(specPath, specBytes, 0644))
	return dir, specPath, specBytes
}

func testResultPathDocumentNode(child *yaml.Node) *yaml.Node {
	return &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{child},
	}
}

func testResultPathMappingNode(items ...interface{}) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for i := 0; i+1 < len(items); i += 2 {
		key, _ := items[i].(string)
		value, _ := items[i+1].(*yaml.Node)
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			value,
		)
	}
	return node
}
