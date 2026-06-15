// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	jsplugin "github.com/daveshanley/vacuum/plugin/javascript"
	"github.com/daveshanley/vacuum/rulesets"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/index"
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

func TestIssue879RecursiveCustomRuleSharedResponsePathsAreCompleteAndStable(t *testing.T) {
	dir, specPath, specBytes := writeIssue879RecursiveCustomRuleFixture(t)

	rule := &model.Rule{
		Id:          "repro-property-description",
		Description: "Every property must have a description",
		Message:     "Property is missing a description",
		Given:       "$..properties.*",
		Resolved:    true,
		Severity:    model.SeverityError,
		Then: &model.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
	}
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}

	expectedPaths := []string{
		"$.paths['/v1/orders'].post.responses['400'].content['*/*'].schema.properties['error-code'].description",
		"$.paths['/v1/orders'].post.responses['500'].content['*/*'].schema.properties['error-code'].description",
		"$.paths['/v1/orders/{orderId}'].get.responses['400'].content['*/*'].schema.properties['error-code'].description",
		"$.paths['/v1/orders/{orderId}'].get.responses['404'].content['*/*'].schema.properties['error-code'].description",
		"$.paths['/v1/orders/{orderId}'].get.responses['500'].content['*/*'].schema.properties['error-code'].description",
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

		var errorCodeResults []model.RuleFunctionResult
		for _, result := range results.Results {
			if result.RuleId == rule.Id && strings.Contains(result.Path, "error-code") {
				errorCodeResults = append(errorCodeResults, result)
			}
		}

		if assert.Len(t, errorCodeResults, 1, "iteration %d", i) {
			assert.Equal(t, expectedPaths[0], errorCodeResults[0].Path, "iteration %d", i)
			assert.Equal(t, expectedPaths, errorCodeResults[0].Paths, "iteration %d", i)
		}
	}
}

func TestIssue879RecursiveFilterCustomRuleSharedResponsePathsAreCompleteAndStable(t *testing.T) {
	dir, specPath, specBytes := writeIssue879RecursiveCustomRuleFixture(t)

	rule := &model.Rule{
		Id:          "repro-string-property-description",
		Description: "Every string property must have a description",
		Message:     "String property is missing a description",
		Given:       "$..[?(@ && @.type && @.type == 'string')]",
		Resolved:    true,
		Severity:    model.SeverityError,
		Then: &model.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
	}
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}

	expectedPaths := []string{
		"$.paths['/v1/orders'].post.responses['400'].content['*/*'].schema.properties['error-code'].description",
		"$.paths['/v1/orders'].post.responses['500'].content['*/*'].schema.properties['error-code'].description",
		"$.paths['/v1/orders/{orderId}'].get.responses['400'].content['*/*'].schema.properties['error-code'].description",
		"$.paths['/v1/orders/{orderId}'].get.responses['404'].content['*/*'].schema.properties['error-code'].description",
		"$.paths['/v1/orders/{orderId}'].get.responses['500'].content['*/*'].schema.properties['error-code'].description",
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

		var errorCodeResults []model.RuleFunctionResult
		for _, result := range results.Results {
			if result.RuleId == rule.Id && strings.Contains(result.Path, "error-code") {
				errorCodeResults = append(errorCodeResults, result)
			}
		}

		if assert.Len(t, errorCodeResults, 1, "iteration %d", i) {
			assert.Equal(t, expectedPaths[0], errorCodeResults[0].Path, "iteration %d", i)
			assert.Equal(t, expectedPaths, errorCodeResults[0].Paths, "iteration %d", i)
		}
	}
}

func TestIssue879RecursiveAllOfSiblingReferencePathsAreCompleteAndStable(t *testing.T) {
	dir, specPath, specBytes := writeIssue879AllOfSiblingReferenceFixture(t)

	descriptionRule := &model.Rule{
		Id:          "repro-property-description",
		Description: "Every property must have a description",
		Message:     "Property is missing a description",
		Given:       []string{"$..properties.*", "$..items"},
		Resolved:    true,
		Severity:    model.SeverityError,
		Then: &model.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
	}
	requiredRule := &model.Rule{
		Id:          "repro-object-required",
		Description: "Every object must declare required fields",
		Message:     "Object is missing required",
		Given:       "$..[?(@ && @.type && @.type == 'object')]",
		Severity:    model.SeverityWarn,
		Then: &model.RuleAction{
			Field:    "required",
			Function: "truthy",
		},
	}
	enumRule := &model.Rule{
		Id:          "repro-enum-uppercase",
		Description: "Enum values must be uppercase",
		Message:     "Enum value must be uppercase",
		Given:       "$..enum[*]",
		Resolved:    true,
		Severity:    model.SeverityError,
		Then: &model.RuleAction{
			Function: "pattern",
			FunctionOptions: map[string]interface{}{
				"match": "^[A-Z_]+$",
			},
		},
	}
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{
		descriptionRule.Id: descriptionRule,
		requiredRule.Id:    requiredRule,
		enumRule.Id:        enumRule,
	}}

	expectedDescriptionPaths := []string{
		"$.components.schemas['deathFile'].properties['death'].allOf[0].description",
		"$.paths['/v1/deathAlert'].post.requestBody.content['application/json'].schema.properties['death'].allOf[0].description",
	}
	expectedItemsPaths := []string{
		"$.components.schemas['deathFile'].properties['contracts'].items",
		"$.paths['/v1/deathAlert'].post.requestBody.content['application/json'].schema.properties['contracts'].items",
	}
	expectedRequiredPaths := []string{
		"$.components.schemas['deathFile'].properties['death'].allOf[0].required",
		"$.paths['/v1/deathAlert'].post.requestBody.content['application/json'].schema.properties['death'].allOf[0].required",
	}
	expectedEnumPaths := []string{
		"$.components.schemas['request_c'].properties['type'].enum[0]",
		"$.paths['/v1/request'].post.requestBody.content['application/json'].schema.properties['type'].enum[0]",
	}

	previousProcs := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(previousProcs)

	for _, procs := range []int{1, 4} {
		runtime.GOMAXPROCS(procs)
		for i := 0; i < 50; i++ {
			results := ApplyRulesToRuleSet(&RuleSetExecution{
				RuleSet:           ruleSet,
				Spec:              specBytes,
				SpecFileName:      specPath,
				Base:              dir,
				AllowLookup:       true,
				NodeLookupTimeout: 5 * time.Second,
				SilenceLogs:       true,
			})

			require.Empty(t, results.Errors, "GOMAXPROCS=%d iteration %d", procs, i)

			descriptionResult := findResultByRuleAndPath(results.Results, descriptionRule.Id, expectedDescriptionPaths[0])
			if assert.NotNil(t, descriptionResult, "GOMAXPROCS=%d iteration %d", procs, i) {
				assert.Equal(t, expectedDescriptionPaths, descriptionResult.Paths, "GOMAXPROCS=%d iteration %d", procs, i)
			}

			itemsResult := findResultByRuleAndPath(results.Results, descriptionRule.Id, expectedItemsPaths[0])
			if assert.NotNil(t, itemsResult, "GOMAXPROCS=%d iteration %d", procs, i) {
				assert.Equal(t, expectedItemsPaths, itemsResult.Paths, "GOMAXPROCS=%d iteration %d", procs, i)
			}

			requiredResult := findResultByRuleAndPath(results.Results, requiredRule.Id, expectedRequiredPaths[0])
			if assert.NotNil(t, requiredResult, "GOMAXPROCS=%d iteration %d", procs, i) {
				assert.Equal(t, expectedRequiredPaths, requiredResult.Paths, "GOMAXPROCS=%d iteration %d", procs, i)
			}

			enumResult := findResultByRuleAndPath(results.Results, enumRule.Id, expectedEnumPaths[0])
			if assert.NotNil(t, enumResult, "GOMAXPROCS=%d iteration %d", procs, i) {
				assert.Equal(t, expectedEnumPaths, enumResult.Paths, "GOMAXPROCS=%d iteration %d", procs, i)
			}
		}
	}
}

func TestIssue879RecursiveNestedReferenceAliasesIncludeSiblingComponentUses(t *testing.T) {
	dir, specPath, specBytes := writeIssue879NestedReferenceAliasFixture(t)

	rule := &model.Rule{
		Id:          "repro-string-min-length",
		Description: "Every string must declare minLength",
		Message:     "String is missing minLength",
		Given:       "$..[?(@ && @.type && @.type == 'string')]",
		Resolved:    true,
		Severity:    model.SeverityError,
		Then: &model.RuleAction{
			Field:    "minLength",
			Function: "defined",
		},
	}
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}

	expectedPaths := []string{
		"$.components.schemas['vesselSurveyResult'].allOf[1].allOf[2].properties['hullIdentificationNumber'].allOf[0]",
		"$.paths['/v1/vesselSurveys'].post.requestBody.content['application/json'].schema.allOf[2].properties['hullIdentificationNumber'].allOf[0]",
		"$.paths['/v1/vesselSurveys'].post.responses['201'].content['application/json'].schema.allOf[1].allOf[2].properties['hullIdentificationNumber'].allOf[0]",
	}

	previousProcs := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(previousProcs)

	for _, procs := range []int{1, 4} {
		runtime.GOMAXPROCS(procs)
		for i := 0; i < 50; i++ {
			results := ApplyRulesToRuleSet(&RuleSetExecution{
				RuleSet:           ruleSet,
				Spec:              specBytes,
				SpecFileName:      specPath,
				Base:              dir,
				AllowLookup:       true,
				NodeLookupTimeout: 5 * time.Second,
				SilenceLogs:       true,
			})

			require.Empty(t, results.Errors, "GOMAXPROCS=%d iteration %d", procs, i)

			result := findResultByRuleAndPath(results.Results, rule.Id, expectedPaths[0])
			if assert.NotNil(t, result, "GOMAXPROCS=%d iteration %d", procs, i) {
				assert.Equal(t, expectedPaths, result.Paths, "GOMAXPROCS=%d iteration %d", procs, i)
			}
		}
	}
}

func TestIssue879ComponentSchemaReferenceWithAdditionalPropertiesPathsAreCompleteAndStable(t *testing.T) {
	dir, specPath, specBytes := writeIssue879AdditionalPropertiesComponentAliasFixture(t)

	rule := &model.Rule{
		Id:          "repro-schema-description",
		Description: "Every component schema must have a description",
		Message:     "Schema is missing a description",
		Given:       "$.components.*.*",
		Resolved:    true,
		Severity:    model.SeverityError,
		Then: &model.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
	}
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}

	expectedPaths := []string{
		"$.components.schemas['fraudy_description'].description",
		"$.paths['/v1/fraud-detection/analyze'].post.requestBody.content['application/json'].schema.description",
	}

	previousProcs := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(previousProcs)

	for _, procs := range []int{1, 4} {
		runtime.GOMAXPROCS(procs)
		for i := 0; i < 50; i++ {
			results := ApplyRulesToRuleSet(&RuleSetExecution{
				RuleSet:           ruleSet,
				Spec:              specBytes,
				SpecFileName:      specPath,
				Base:              dir,
				AllowLookup:       true,
				NodeLookupTimeout: 5 * time.Second,
				SilenceLogs:       true,
			})

			require.Empty(t, results.Errors, "GOMAXPROCS=%d iteration %d", procs, i)

			result := findResultByRuleAndPath(results.Results, rule.Id, expectedPaths[0])
			if assert.NotNil(t, result, "GOMAXPROCS=%d iteration %d", procs, i) {
				assert.Equal(t, expectedPaths, result.Paths, "GOMAXPROCS=%d iteration %d", procs, i)
			}
		}
	}
}

func TestIssue879ComponentSchemaAdditionalPropertiesReferencePathsArePreserved(t *testing.T) {
	dir, specPath, specBytes := writeIssue879AdditionalPropertiesSchemaReferenceFixture(t)

	rule := &model.Rule{
		Id:          "repro-schema-description",
		Description: "Every component schema must have a description",
		Message:     "Schema is missing a description",
		Given:       "$.components.*.*",
		Resolved:    true,
		Severity:    model.SeverityError,
		Then: &model.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
	}
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}

	expectedPaths := []string{
		"$.components.schemas['shared_value'].description",
		"$.paths['/v1/direct'].post.requestBody.content['application/json'].schema.description",
		"$.paths['/v1/map'].post.requestBody.content['application/json'].schema.additionalProperties.description",
	}

	previousProcs := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(previousProcs)

	for _, procs := range []int{1, 4} {
		runtime.GOMAXPROCS(procs)
		for i := 0; i < 50; i++ {
			results := ApplyRulesToRuleSet(&RuleSetExecution{
				RuleSet:           ruleSet,
				Spec:              specBytes,
				SpecFileName:      specPath,
				Base:              dir,
				AllowLookup:       true,
				NodeLookupTimeout: 5 * time.Second,
				SilenceLogs:       true,
			})

			require.Empty(t, results.Errors, "GOMAXPROCS=%d iteration %d", procs, i)

			result := findResultByRuleAndPath(results.Results, rule.Id, expectedPaths[0])
			if assert.NotNil(t, result, "GOMAXPROCS=%d iteration %d", procs, i) {
				assert.Equal(t, expectedPaths, result.Paths, "GOMAXPROCS=%d iteration %d", procs, i)
			}
		}
	}
}

func TestEquivalentResultReferenceTargetPathsStopsOnDescendantReferenceCycle(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`openapi: 3.0.3
info:
  title: descendant reference cycle
  version: 1.0.0
paths: {}
components:
  schemas:
    Loop:
      $ref: '#/components/schemas/Loop/properties/next'
      properties:
        next:
          type: object
`), &root))

	cfg := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&root, cfg)
	idx.BuildIndex()
	pathIndex := resultPathIndexForSpec(idx, make(map[*index.SpecIndex]*vacuumUtils.NodePathIndex))

	done := make(chan []string, 1)
	go func() {
		done <- equivalentResultReferenceTargetPaths(idx, "$.components.schemas['Loop']", pathIndex)
	}()

	select {
	case paths := <-done:
		require.NotEmpty(t, paths)
		assert.Contains(t, paths, "$.components.schemas['Loop']")
		assert.Contains(t, paths, "$.components.schemas['Loop'].properties['next']")
		assert.LessOrEqual(t, len(paths), maxResultReferenceAliasDepth+1)
	case <-time.After(time.Second):
		t.Fatal("equivalentResultReferenceTargetPaths did not terminate")
	}
}

func TestResultPathCacheDoesNotUseRootPositionForExternalOrigin(t *testing.T) {
	rootLocation := filepath.Join(t.TempDir(), "openapi.yaml")
	externalLocation := filepath.Join(t.TempDir(), "common.yaml")
	rootValue := &yaml.Node{
		Kind:   yaml.ScalarNode,
		Value:  "root",
		Line:   60,
		Column: 11,
	}
	root := testResultPathDocumentNode(testResultPathMappingNode("tags", rootValue))
	cache := newResultPathCache(root, rootLocation)

	rootOriginNode := &yaml.Node{
		Kind:   yaml.ScalarNode,
		Value:  "root origin",
		Line:   60,
		Column: 11,
	}
	rootResult := &model.RuleFunctionResult{
		Path: "unknown",
		Origin: &index.NodeOrigin{
			Node:             rootOriginNode,
			Line:             60,
			Column:           11,
			AbsoluteLocation: rootLocation,
		},
	}
	cache.reconcile(rootResult)
	assert.Equal(t, "$", rootResult.Path)

	externalNode := &yaml.Node{
		Kind:   yaml.ScalarNode,
		Value:  "external",
		Line:   60,
		Column: 11,
	}
	externalResult := &model.RuleFunctionResult{
		Path:      "unknown",
		StartNode: externalNode,
		Origin: &index.NodeOrigin{
			Node:             externalNode,
			Line:             60,
			Column:           11,
			AbsoluteLocation: externalLocation,
		},
	}
	cache.reconcile(externalResult)
	assert.Equal(t, "unknown", externalResult.Path)
}

func TestMergeResultPathCandidatesDropsDriftedPrimaryPath(t *testing.T) {
	candidates := []string{
		"$.paths['/v1/a'].get.responses['400'].content['*/*'].examples",
		"$.paths['/v1/b'].get.responses['400'].content['*/*'].examples",
	}
	result := &model.RuleFunctionResult{
		Path: "$.tags[1].examples",
		Paths: []string{
			"$.tags[1].examples",
			candidates[0],
		},
	}

	mergeResultPathCandidates(result, candidates)

	assert.Equal(t, candidates[0], result.Path)
	assert.Equal(t, candidates, result.Paths)
}

func TestDropRedundantAdditionalPropertiesFieldAliasesPreservesReferenceAliases(t *testing.T) {
	paths := []string{
		"$.components.schemas['shared_value'].description",
		"$.paths['/v1/map'].post.requestBody.content['application/json'].schema.additionalProperties.description",
		"$.paths['/v1/map'].post.requestBody.content['application/json'].schema.description",
	}

	assert.Equal(t, []string{
		"$.components.schemas['shared_value'].description",
		"$.paths['/v1/map'].post.requestBody.content['application/json'].schema.description",
	}, dropRedundantAdditionalPropertiesFieldAliases(paths, nil))

	assert.Equal(t, paths, dropRedundantAdditionalPropertiesFieldAliases(paths, map[string]struct{}{
		"$.paths['/v1/map'].post.requestBody.content['application/json'].schema.additionalProperties": {},
	}))
}

func TestShouldCompleteAliasedResultPathsFromReferencesSkipsRootGivenComponentResults(t *testing.T) {
	result := &model.RuleFunctionResult{
		Path:      "$.components.schemas['shared_value'].description",
		StartNode: &yaml.Node{Kind: yaml.MappingNode},
		Rule: &model.Rule{
			Given:    "$",
			Resolved: true,
		},
	}

	assert.False(t, shouldCompleteAliasedResultPathsFromReferences(result))
}

func TestShouldCompleteAliasedResultPathsFromReferencesAllowsComponentGivenResults(t *testing.T) {
	result := &model.RuleFunctionResult{
		Path:      "$.components.schemas['shared_value'].description",
		StartNode: &yaml.Node{Kind: yaml.MappingNode},
		Rule: &model.Rule{
			Given:    "$.components.*.*",
			Resolved: true,
		},
	}

	assert.True(t, shouldCompleteAliasedResultPathsFromReferences(result))
}

func TestIssue879SyntheticFixtureResultPathsAreStable(t *testing.T) {
	specPath := filepath.Join("test_data", "issue_879", "synthetic-openapi.yaml")
	rulesetPath := filepath.Join("test_data", "issue_879", "synthetic-ruleset.yaml")

	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	rulesetBytes, err := os.ReadFile(rulesetPath)
	require.NoError(t, err)

	suppliedRuleSet, err := rulesets.CreateRuleSetFromData(rulesetBytes)
	require.NoError(t, err)
	ruleSet := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(suppliedRuleSet)

	var expected []string
	for i := 0; i < 30; i++ {
		results := ApplyRulesToRuleSet(&RuleSetExecution{
			RuleSet:           ruleSet,
			Spec:              specBytes,
			SpecFileName:      specPath,
			Base:              filepath.Dir(specPath),
			AllowLookup:       true,
			NodeLookupTimeout: 5 * time.Second,
			SilenceLogs:       true,
		})

		require.Empty(t, results.Errors, "iteration %d", i)
		actual := resultPathSnapshot(results.Results)
		require.NotEmpty(t, actual, "iteration %d", i)
		if expected == nil {
			expected = actual
			continue
		}
		assert.Equal(t, expected, actual, "iteration %d", i)
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

func TestResultPathNeedsReconciliationDetectsSelectorFallbacks(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		pathFromRuleGiven bool
		want              bool
	}{
		{
			name: "empty",
			path: "",
			want: true,
		},
		{
			name: "unknown",
			path: "unknown",
			want: true,
		},
		{
			name: "wildcard",
			path: "$.paths[*][*].responses",
			want: true,
		},
		{
			name:              "recursive_filter_fallback",
			path:              "$..[?(@ && @.in == 'header')].name",
			pathFromRuleGiven: true,
			want:              true,
		},
		{
			name:              "nested_recursive_fallback",
			path:              "$.paths..summary",
			pathFromRuleGiven: true,
			want:              true,
		},
		{
			name: "explicit_recursive_filter",
			path: "$..[?(@ && @.in == 'header')].name",
			want: false,
		},
		{
			name: "concrete",
			path: "$.paths['/pets'].get.parameters[0].name",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &model.RuleFunctionResult{
				Path:              tt.path,
				PathFromRuleGiven: tt.pathFromRuleGiven,
			}
			assert.Equal(t, tt.want, resultPathNeedsReconciliation(result))
		})
	}
}

func TestAppendSelectorTerminalSegment(t *testing.T) {
	tests := []struct {
		name          string
		canonicalPath string
		selectorPath  string
		want          string
	}{
		{
			name:          "dot_terminal",
			canonicalPath: "$.paths['/pets'].get.parameters[0]",
			selectorPath:  "$..[?(@ && @.in == 'header')].name",
			want:          "$.paths['/pets'].get.parameters[0].name",
		},
		{
			name:          "quoted_terminal",
			canonicalPath: "$.paths['/pets'].get.parameters[0]",
			selectorPath:  "$..[?(@ && @.in == 'header')]['name']",
			want:          "$.paths['/pets'].get.parameters[0].name",
		},
		{
			name:          "already_terminal",
			canonicalPath: "$.paths['/pets'].get.parameters[0].name",
			selectorPath:  "$..[?(@ && @.in == 'header')].name",
			want:          "$.paths['/pets'].get.parameters[0].name",
		},
		{
			name:          "filter_only",
			canonicalPath: "$.paths['/pets'].get.parameters[0]",
			selectorPath:  "$..[?(@ && @.in == 'header')]",
			want:          "$.paths['/pets'].get.parameters[0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, appendSelectorTerminalSegment(tt.canonicalPath, tt.selectorPath))
		})
	}
}

func TestResultSelectorPathFallsBackToRuleGiven(t *testing.T) {
	result := &model.RuleFunctionResult{
		Path:              "unknown",
		PathFromRuleGiven: true,
		Rule: &model.Rule{
			Given: "$..[?(@ && @.in == 'header')].name",
		},
	}

	assert.Equal(t, "$..[?(@ && @.in == 'header')].name", resultSelectorPath(result))
}

func TestUpgradeSelectorTerminalPathsUsesRuleGiven(t *testing.T) {
	results := []model.RuleFunctionResult{
		{
			Path:              "$.paths['/pets'].get.parameters[0]",
			PathFromRuleGiven: true,
			Rule: &model.Rule{
				Given: "$..[?(@ && @.in == 'header')].name",
			},
		},
	}

	upgradeSelectorTerminalPaths(results)

	assert.Equal(t, "$.paths['/pets'].get.parameters[0].name", results[0].Path)
}

func TestIssue907RecursiveFilterPathlessJSResultUsesMatchedPath(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: issue 907 repro
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: X-Trace
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
`)
	ruleFunc := jsplugin.NewJSRuleFunction("issue907", `
function runRule(input) {
  if (!input) {
    return [];
  }
  return [{ message: "header name issue" }];
}
`)
	require.NoError(t, ruleFunc.CheckScript())

	rule := &model.Rule{
		Id:           "issue-907-filter-path",
		Description:  "Header names should be checked at the matched node",
		Given:        "$..[?(@ && @.in == 'header')].name",
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         rulesets.Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "issue907",
		},
	}
	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: &rulesets.RuleSet{
			Rules: map[string]*model.Rule{rule.Id: rule},
		},
		Spec:        spec,
		SilenceLogs: true,
		CustomFunctions: map[string]model.RuleFunction{
			"issue907": ruleFunc,
		},
	})

	require.Empty(t, results.Errors)
	require.Len(t, results.Results, 1)
	assert.Equal(t, "$.paths['/pets'].get.parameters[0].name", results.Results[0].Path)
	assert.NotContains(t, results.Results[0].Path, "$..")
	assert.NotContains(t, results.Results[0].Path, "[?")
}

func TestIssue907RecursiveFilterExplicitJSResultPathIsPreserved(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: issue 907 explicit path
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: X-Trace
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
`)
	ruleFunc := jsplugin.NewJSRuleFunction("issue907", `
function runRule(input) {
  if (!input) {
    return [];
  }
  return [{
    message: "header name issue",
    path: "$.info.title"
  }];
}
`)
	require.NoError(t, ruleFunc.CheckScript())

	rule := &model.Rule{
		Id:           "issue-907-explicit-path",
		Description:  "Explicit custom paths should be preserved",
		Given:        "$..[?(@ && @.in == 'header')].name",
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         rulesets.Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "issue907",
		},
	}
	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: &rulesets.RuleSet{
			Rules: map[string]*model.Rule{rule.Id: rule},
		},
		Spec:        spec,
		SilenceLogs: true,
		CustomFunctions: map[string]model.RuleFunction{
			"issue907": ruleFunc,
		},
	})

	require.Empty(t, results.Errors)
	require.Len(t, results.Results, 1)
	assert.Equal(t, "$.info.title", results.Results[0].Path)
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

func TestCanonicalizeResultAliasPathQuotesComponentSchemaKeys(t *testing.T) {
	path := "$.components.schemas.lossEventDeclarationResult_c.allOf[2].properties.persons"

	canonical := canonicalizeResultAliasPath(path)

	assert.Equal(t, "$.components.schemas['lossEventDeclarationResult_c'].allOf[2].properties['persons']", canonical)
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

func resultPathSnapshot(results []model.RuleFunctionResult) []string {
	snapshot := make([]string, 0, len(results))
	for _, result := range results {
		snapshot = append(snapshot, result.RuleId+"|"+result.Path+"|"+strings.Join(result.Paths, ","))
	}
	sort.Strings(snapshot)
	return snapshot
}

func findResultByRuleAndPath(results []model.RuleFunctionResult, ruleID, path string) *model.RuleFunctionResult {
	for i := range results {
		if results[i].RuleId == ruleID && results[i].Path == path {
			return &results[i]
		}
	}
	return nil
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

func writeIssue879AllOfSiblingReferenceFixture(t *testing.T) (string, string, []byte) {
	t.Helper()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi-test.yaml")

	specBytes := []byte(`openapi: 3.0.3
info:
  title: Vacuum issue 879 allOf sibling ref repro
  version: 1.0.0
paths:
  /v1/deathAlert:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/deathFile'
      responses:
        '200':
          description: ok
  /v1/request:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/request_c'
      responses:
        '200':
          description: ok
components:
  schemas:
    deathFile:
      type: object
      properties:
        death:
          type: object
          $ref: '#/components/schemas/dead'
        contracts:
          type: array
          items:
            $ref: '#/components/schemas/deadContract'
    dead:
      type: object
      properties:
        title:
          type: string
          description: Civilité
    deadContract:
      type: object
      properties:
        id:
          type: string
          description: Contract id
    request_c:
      type: object
      properties:
        type:
          type: string
          enum:
            - pending
`)
	require.NoError(t, os.WriteFile(specPath, specBytes, 0644))
	return dir, specPath, specBytes
}

func writeIssue879NestedReferenceAliasFixture(t *testing.T) (string, string, []byte) {
	t.Helper()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi-test.yaml")

	specBytes := []byte(`openapi: 3.0.3
info:
  title: Vacuum issue 879 marine nested alias repro
  version: 1.0.0
paths:
  /v1/vesselSurveys:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/vesselSurvey'
      responses:
        '201':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/vesselSurveyResult'
components:
  schemas:
    vesselSurvey:
      allOf:
        - $ref: '#/components/schemas/vesselProfile'
        - $ref: '#/components/schemas/harborContact'
        - type: object
          properties:
            hullIdentificationNumber:
              description: Primary hull identifier for the vessel
              $ref: '#/components/schemas/nauticalIdentifier'
    vesselSurveyResult:
      allOf:
        - $ref: '#/components/schemas/surveyRecord'
        - $ref: '#/components/schemas/vesselSurvey'
    vesselProfile:
      type: object
      properties:
        vesselName:
          type: string
          minLength: 1
    harborContact:
      type: object
      properties:
        harborMaster:
          type: string
          minLength: 1
    surveyRecord:
      type: object
      properties:
        surveyId:
          type: string
          minLength: 1
    nauticalIdentifier:
      type: string
`)
	require.NoError(t, os.WriteFile(specPath, specBytes, 0644))
	return dir, specPath, specBytes
}

func writeIssue879AdditionalPropertiesComponentAliasFixture(t *testing.T) (string, string, []byte) {
	t.Helper()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi-test.yaml")

	specBytes := []byte(`openapi: 3.0.3
info:
  title: Vacuum issue 879 component additionalProperties repro
  version: 1.0.0
paths:
  /v1/fraud-detection/analyze:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/fraudy_description'
      responses:
        '200':
          description: ok
components:
  schemas:
    fraudy_description:
      additionalProperties: false
      type: object
      properties:
        score:
          type: string
          description: Fraud score label
`)
	require.NoError(t, os.WriteFile(specPath, specBytes, 0644))
	return dir, specPath, specBytes
}

func writeIssue879AdditionalPropertiesSchemaReferenceFixture(t *testing.T) (string, string, []byte) {
	t.Helper()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi-test.yaml")

	specBytes := []byte(`openapi: 3.0.3
info:
  title: Vacuum issue 879 additionalProperties reference repro
  version: 1.0.0
paths:
  /v1/direct:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/shared_value'
      responses:
        '200':
          description: ok
  /v1/map:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              additionalProperties:
                $ref: '#/components/schemas/shared_value'
      responses:
        '200':
          description: ok
components:
  schemas:
    shared_value:
      type: string
`)
	require.NoError(t, os.WriteFile(specPath, specBytes, 0644))
	return dir, specPath, specBytes
}

func writeIssue879RecursiveCustomRuleFixture(t *testing.T) (string, string, []byte) {
	t.Helper()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "openapi-test.yaml")
	commonDir := filepath.Join(dir, "common")
	require.NoError(t, os.MkdirAll(commonDir, 0755))
	commonPath := filepath.Join(commonDir, "error-response.yaml")

	require.NoError(t, os.WriteFile(commonPath, []byte(`ErrorResponse:
  description: Standard error response
  content:
    '*/*':
      schema:
        type: object
        properties:
          error-code:
            type: string
          error-message:
            type: string
          error-detail:
            type: string
`), 0644))

	specBytes := []byte(`openapi: 3.0.3
info:
  title: Vacuum issue 879 recursive custom rule repro
  version: 1.0.0
paths:
  /v1/orders:
    post:
      responses:
        '201':
          description: Order created
        '400':
          $ref: './common/error-response.yaml#/ErrorResponse'
        '500':
          $ref: './common/error-response.yaml#/ErrorResponse'
  /v1/orders/{orderId}:
    get:
      responses:
        '200':
          description: Order details
        '400':
          $ref: './common/error-response.yaml#/ErrorResponse'
        '404':
          $ref: './common/error-response.yaml#/ErrorResponse'
        '500':
          $ref: './common/error-response.yaml#/ErrorResponse'
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
