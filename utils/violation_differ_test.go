// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func makeResult(ruleId, path, message string) model.RuleFunctionResult {
	return model.RuleFunctionResult{
		RuleId:  ruleId,
		Path:    path,
		Message: message,
		Rule:    &model.Rule{Id: ruleId},
	}
}

func makeResultPtr(ruleId, path, message string) *model.RuleFunctionResult {
	r := makeResult(ruleId, path, message)
	return &r
}

func makeOriginResult(ruleId, path, message, location string, line, column int) model.RuleFunctionResult {
	result := makeResult(ruleId, path, message)
	result.Origin = &index.NodeOrigin{
		AbsoluteLocation: location,
		Line:             line,
		Column:           column,
	}
	return result
}

func makeOriginResultPtr(ruleId, path, message, location string, line, column int) *model.RuleFunctionResult {
	result := makeOriginResult(ruleId, path, message, location, line, column)
	return &result
}

func makeIndexedSourceResult(t *testing.T, ruleId, path, message string) model.RuleFunctionResult {
	return makeIndexedSourceResultAtLocation(t, ruleId, path, message, "common.yaml", 0)
}

func makeIndexedSourceResultAtLocation(t *testing.T, ruleId, path, message, location string, lineOffset int) model.RuleFunctionResult {
	t.Helper()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`
components:
  schemas:
    Error:
      type: object
      properties:
        error-code:
          type: string
`), &root))

	doc := root.Content[0]
	idx := index.NewSpecIndexWithConfig(doc, index.CreateOpenAPIIndexConfig())
	node := findScalarNodeForTest(doc, "error-code")
	require.NotNil(t, node)

	result := makeResult(ruleId, path, message)
	result.Paths = []string{path}
	result.Origin = &index.NodeOrigin{
		AbsoluteLocation: location,
		Index:            idx,
		Node:             node,
		Line:             node.Line + lineOffset,
		Column:           node.Column,
	}
	return result
}

func findScalarNodeForTest(node *yaml.Node, value string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.ScalarNode && node.Value == value {
		return node
	}
	for _, child := range node.Content {
		if found := findScalarNodeForTest(child, value); found != nil {
			return found
		}
	}
	return nil
}

func TestDiffViolationsValues_BothEmpty(t *testing.T) {
	result, stats := DiffViolationsValues(nil, nil)
	assert.Empty(t, result)
	assert.Equal(t, 0, stats.TotalResultsBefore)
	assert.Equal(t, 0, stats.TotalResultsAfter)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValues_OriginalEmpty(t *testing.T) {
	newResults := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"),
		makeResult("rule-2", "$.info", "missing contact"),
	}
	result, stats := DiffViolationsValues(nil, newResults)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, stats.TotalResultsBefore)
	assert.Equal(t, 2, stats.TotalResultsAfter)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValues_NewEmpty(t *testing.T) {
	original := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"),
	}
	result, stats := DiffViolationsValues(original, nil)
	assert.Empty(t, result)
	assert.Equal(t, 0, stats.TotalResultsBefore)
	assert.Equal(t, 0, stats.TotalResultsAfter)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValues_NoOverlap(t *testing.T) {
	original := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./old", "old violation"),
	}
	newResults := []model.RuleFunctionResult{
		makeResult("rule-2", "$.paths./new", "new violation"),
		makeResult("rule-3", "$.info", "another new"),
	}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Len(t, result, 2)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValues_FullOverlap(t *testing.T) {
	violations := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"),
		makeResult("rule-2", "$.info.contact", "missing name"),
	}
	result, stats := DiffViolationsValues(violations, violations)
	assert.Empty(t, result)
	assert.Equal(t, 2, stats.ResultsDropped)
	assert.Len(t, stats.RulesFullyFiltered, 2)
}

func TestDiffViolationsValues_PartialOverlap(t *testing.T) {
	original := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"),
	}
	newResults := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"), // same — suppressed
		makeResult("rule-2", "$.info", "missing contact"),                  // new — kept
	}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Len(t, result, 1)
	assert.Equal(t, "rule-2", result[0].RuleId)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValues_SameRuleIdPathDifferentMessage(t *testing.T) {
	// info-contact-properties scenario: same (RuleId, Path), different Message
	original := []model.RuleFunctionResult{
		makeResult("info-contact-properties", "$.info.contact", "missing name"),
	}
	newResults := []model.RuleFunctionResult{
		makeResult("info-contact-properties", "$.info.contact", "missing name"), // same — suppressed
		makeResult("info-contact-properties", "$.info.contact", "missing url"),  // different message — kept
	}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Len(t, result, 1)
	assert.Equal(t, "missing url", result[0].Message)
	assert.Equal(t, 1, stats.ResultsDropped)
	assert.Equal(t, 1, stats.RulesPartialFiltered["info-contact-properties"])
}

func TestDiffViolationsValues_SameRuleIdPathMessage_Suppressed(t *testing.T) {
	v := makeResult("rule-1", "$.paths./items.post", "missing description")
	original := []model.RuleFunctionResult{v}
	newResults := []model.RuleFunctionResult{v}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValues_DuplicateCount(t *testing.T) {
	// Original has 2, new has 3 at same key → 1 reported
	v := makeResult("rule-1", "$.paths./items.post", "missing description")
	original := []model.RuleFunctionResult{v, v}
	newResults := []model.RuleFunctionResult{v, v, v}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Len(t, result, 1)
	assert.Equal(t, 2, stats.ResultsDropped)
	assert.Equal(t, 2, stats.RulesPartialFiltered["rule-1"])
}

func TestDiffViolationsValues_EmptyPathFallsBackToPaths(t *testing.T) {
	original := []model.RuleFunctionResult{
		{
			RuleId:  "rule-1",
			Path:    "",
			Paths:   []string{"$.fallback.path"},
			Message: "test",
			Rule:    &model.Rule{Id: "rule-1"},
		},
	}
	newResults := []model.RuleFunctionResult{
		{
			RuleId:  "rule-1",
			Path:    "",
			Paths:   []string{"$.fallback.path"},
			Message: "test",
			Rule:    &model.Rule{Id: "rule-1"},
		},
	}
	result, _ := DiffViolationsValues(original, newResults)
	assert.Empty(t, result) // should be matched and suppressed
}

func TestDiffViolationsValues_PrimaryPathCanDifferWhenPathsMatch(t *testing.T) {
	original := []model.RuleFunctionResult{
		{
			RuleId: "owasp-string-restricted",
			Path:   "$.paths['/a'].patch.requestBody.content['application/json'].schema.items.properties['path']",
			Paths: []string{
				"$.paths['/a'].patch.requestBody.content['application/json'].schema.items.properties['path']",
				"$.paths['/b'].patch.requestBody.content['application/json'].schema.items.properties['path']",
			},
			Message: "schema of type `string` must specify `format`, `const`, `enum` or `pattern`",
			Rule:    &model.Rule{Id: "owasp-string-restricted"},
		},
	}
	newResults := []model.RuleFunctionResult{
		{
			RuleId: "owasp-string-restricted",
			Path:   "$.paths['/b'].patch.requestBody.content['application/json'].schema.items.properties['path']",
			Paths: []string{
				"$.paths['/b'].patch.requestBody.content['application/json'].schema.items.properties['path']",
				"$.paths['/a'].patch.requestBody.content['application/json'].schema.items.properties['path']",
			},
			Message: "schema of type `string` must specify `format`, `const`, `enum` or `pattern`",
			Rule:    &model.Rule{Id: "owasp-string-restricted"},
		},
	}

	result, stats := DiffViolationsValues(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValues_PathsSuppressExternalRefPrimaryDrift(t *testing.T) {
	message := "schema of type `string` must specify `format`, `const`, `enum` or `pattern`"
	pathA := "$.paths['/a'].patch.requestBody.content['application/json'].schema.items.properties['path']"
	pathB := "$.paths['/b'].patch.requestBody.content['application/json'].schema.items.properties['path']"
	original := []model.RuleFunctionResult{
		{
			RuleId:  "owasp-string-restricted",
			Path:    pathA,
			Paths:   []string{pathA, pathB},
			Message: message,
			Rule:    &model.Rule{Id: "owasp-string-restricted"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "api-common.yaml",
				Line:             258,
				Column:           11,
			},
		},
	}
	newResults := []model.RuleFunctionResult{
		{
			RuleId:  "owasp-string-restricted",
			Path:    pathB,
			Paths:   []string{pathB, pathA},
			Message: message,
			Rule:    &model.Rule{Id: "owasp-string-restricted"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "api-common.yaml",
				Line:             258,
				Column:           11,
			},
		},
	}

	result, stats := DiffViolationsValues(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValues_PathIntersectionSuppressesAliasedLineShiftDrift(t *testing.T) {
	message := "examples must be set"
	sharedPath := "$.paths['/a'].get.responses['400'].content['*/*'].examples"
	original := []model.RuleFunctionResult{
		{
			RuleId: "cnp-p0043-examples-must-exist",
			Path:   "$.tags[1].examples",
			Paths: []string{
				"$.tags[1].examples",
				sharedPath,
			},
			Message: message,
			Rule:    &model.Rule{Id: "cnp-p0043-examples-must-exist"},
		},
	}
	newResults := []model.RuleFunctionResult{
		{
			RuleId: "cnp-p0043-examples-must-exist",
			Path:   "$.examples",
			Paths: []string{
				"$.examples",
				sharedPath,
			},
			Message: message,
			Rule:    &model.Rule{Id: "cnp-p0043-examples-must-exist"},
		},
	}

	result, stats := DiffViolationsValues(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValues_PathIdentitySuppressesOriginLineShift(t *testing.T) {
	message := "operation method `GET` at path `/pets` is missing a description or summary"
	original := []model.RuleFunctionResult{
		makeOriginResult(
			"operation-description",
			"$.paths['/pets'].get",
			message,
			"/workspace/api.yaml",
			14,
			7,
		),
	}
	newResults := []model.RuleFunctionResult{
		makeOriginResult(
			"operation-description",
			"$.paths['/pets'].get",
			message,
			"/workspace/api.yaml",
			22,
			7,
		),
	}

	result, stats := DiffViolationsValues(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValues_PathIdentityKeepsNewAliasPathDistinctWhenSourceMatches(t *testing.T) {
	message := "media type schema property `error-code` is missing `examples` or `example`"
	pathA := "$.paths['/a'].get.responses['400'].content['*/*'].schema.properties['error-code']"
	pathB := "$.paths['/b'].get.responses['400'].content['*/*'].schema.properties['error-code']"

	original := []model.RuleFunctionResult{
		makeIndexedSourceResult(t, "oas3-missing-example", pathA, message),
	}
	newResults := []model.RuleFunctionResult{
		makeIndexedSourceResult(t, "oas3-missing-example", pathB, message),
	}

	result, stats := DiffViolationsValues(original, newResults)
	require.Len(t, result, 1)
	assert.Equal(t, pathB, result[0].Path)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValuesWithOriginBases_RootSourceSuppressesDriftedPath(t *testing.T) {
	message := "service type must be defined"
	originalSpecPath := "/workspace/original/openapi.yaml"
	newSpecPath := "/workspace/shifted/openapi.yaml"
	original := []model.RuleFunctionResult{
		makeIndexedSourceResultAtLocation(
			t,
			"cnp-p0045-x-service-type-must-exist",
			"$.components.schemas['lossEventDeclarationCommon'].properties['collectivity'].x-service-type",
			message,
			originalSpecPath,
			0,
		),
	}
	newResults := []model.RuleFunctionResult{
		makeIndexedSourceResultAtLocation(
			t,
			"cnp-p0045-x-service-type-must-exist",
			"$.components.schemas['wrongLine'].properties['collectivity'].x-service-type",
			message,
			newSpecPath,
			3,
		),
	}

	result, stats := DiffViolationsValuesWithOriginBases(original, newResults, originalSpecPath, newSpecPath)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValuesWithOriginBases_RootSourceKeepsNewAliasPathDistinctWhenSourceMatches(t *testing.T) {
	message := "media type schema property `error-code` is missing `examples` or `example`"
	originalSpecPath := "/workspace/original/openapi.yaml"
	newSpecPath := "/workspace/changed/openapi.yaml"
	pathA := "$.components.schemas.Error.properties['error-code']"
	pathB := "$.paths['/b'].get.responses['400'].content['*/*'].schema.properties['error-code']"

	original := []model.RuleFunctionResult{
		makeIndexedSourceResultAtLocation(t, "oas3-missing-example", pathA, message, originalSpecPath, 0),
	}
	newResults := []model.RuleFunctionResult{
		makeIndexedSourceResultAtLocation(t, "oas3-missing-example", pathB, message, newSpecPath, 0),
	}

	result, stats := DiffViolationsValuesWithOriginBases(original, newResults, originalSpecPath, newSpecPath)
	require.Len(t, result, 1)
	assert.Equal(t, pathB, result[0].Path)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValues_PathsSuppressMirroredDirectoryPrimaryDrift(t *testing.T) {
	message := "minLength must be defined"
	pathA := "$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']"
	pathB := "$.paths['/b'].get.responses['200'].content['application/json'].schema.properties['name']"
	original := []model.RuleFunctionResult{
		{
			RuleId:  "check-string-attribute-minlength",
			Path:    pathA,
			Paths:   []string{pathA, pathB},
			Message: message,
			Rule:    &model.Rule{Id: "check-string-attribute-minlength"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "/workspace/folder1/openapi-3.0/test/common/test-common.yaml",
				Line:             64,
				Column:           11,
			},
		},
	}
	newResults := []model.RuleFunctionResult{
		{
			RuleId:  "check-string-attribute-minlength",
			Path:    pathB,
			Paths:   []string{pathB, pathA},
			Message: message,
			Rule:    &model.Rule{Id: "check-string-attribute-minlength"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "/workspace/folder2/openapi-3.0/test/common/test-common.yaml",
				Line:             64,
				Column:           11,
			},
		},
	}

	result, stats := DiffViolationsValues(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
	assert.Equal(t, "/workspace/folder1/openapi-3.0/test/common/test-common.yaml", original[0].Origin.AbsoluteLocation)
	assert.Equal(t, "/workspace/folder2/openapi-3.0/test/common/test-common.yaml", newResults[0].Origin.AbsoluteLocation)
}

func TestDiffViolationsValues_CanonicalOriginKeepsDifferentMirroredFilesDistinct(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		{
			RuleId:  "check-string-attribute-minlength",
			Path:    "$.components.schemas.Customer.properties.name",
			Message: message,
			Rule:    &model.Rule{Id: "check-string-attribute-minlength"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "/workspace/folder1/openapi-3.0/test/common/customer.yaml",
				Line:             64,
				Column:           11,
			},
		},
		{
			RuleId:  "check-string-attribute-minlength",
			Path:    "$.components.schemas.Order.properties.name",
			Message: message,
			Rule:    &model.Rule{Id: "check-string-attribute-minlength"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "/workspace/folder1/openapi-3.0/test/common/order.yaml",
				Line:             64,
				Column:           11,
			},
		},
	}
	newResults := []model.RuleFunctionResult{
		{
			RuleId:  "check-string-attribute-minlength",
			Path:    "$.components.schemas.Order.properties.name",
			Message: message,
			Rule:    &model.Rule{Id: "check-string-attribute-minlength"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "/workspace/folder2/openapi-3.0/test/common/order.yaml",
				Line:             64,
				Column:           11,
			},
		},
		{
			RuleId:  "check-string-attribute-minlength",
			Path:    "$.components.schemas.Invoice.properties.name",
			Message: message,
			Rule:    &model.Rule{Id: "check-string-attribute-minlength"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "/workspace/folder2/openapi-3.0/test/common/invoice.yaml",
				Line:             64,
				Column:           11,
			},
		},
	}

	result, stats := DiffViolationsValues(original, newResults)
	require.Len(t, result, 1)
	assert.Equal(t, "/workspace/folder2/openapi-3.0/test/common/invoice.yaml", result[0].Origin.AbsoluteLocation)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValues_FallbackOriginPreservesDirectoryContext(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"",
			message,
			"/old/common/schema.yaml",
			64,
			11,
		),
	}
	newResults := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"",
			message,
			"/new/other/schema.yaml",
			64,
			11,
		),
	}

	result, stats := DiffViolationsValues(original, newResults)

	require.Len(t, result, 1)
	assert.Equal(t, "/new/other/schema.yaml", result[0].Origin.AbsoluteLocation)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValuesWithOriginBases_SuppressesMirroredFileWhenFileSetsDiffer(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']",
			message,
			"/workspace/folder1/common/a.yaml",
			64,
			11,
		),
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.paths['/b'].get.responses['200'].content['application/json'].schema.properties['name']",
			message,
			"/workspace/folder1/other/b.yaml",
			72,
			11,
		),
	}
	newResults := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']",
			message,
			"/workspace/folder2/common/a.yaml",
			64,
			11,
		),
	}

	result, stats := DiffViolationsValuesWithOriginBases(
		original,
		newResults,
		"/workspace/folder1/openapi.yaml",
		"/workspace/folder2/openapi.yaml",
	)

	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValuesWithOriginBases_KeepsSameBasenameInDifferentMirroredDirectory(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.components.schemas.Customer.properties.name",
			message,
			"/workspace/folder1/common/a.yaml",
			64,
			11,
		),
	}
	newResults := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.components.schemas.Order.properties.name",
			message,
			"/workspace/folder2/other/a.yaml",
			64,
			11,
		),
	}

	result, stats := DiffViolationsValuesWithOriginBases(
		original,
		newResults,
		"/workspace/folder1/openapi.yaml",
		"/workspace/folder2/openapi.yaml",
	)

	require.Len(t, result, 1)
	assert.Equal(t, "/workspace/folder2/other/a.yaml", result[0].Origin.AbsoluteLocation)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValuesWithOriginBases_SuppressesSiblingExternalRefOutsideSpecDirectory(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']",
			message,
			"/workspace/folder1/common/schema.yaml",
			64,
			11,
		),
	}
	newResults := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']",
			message,
			"/workspace/folder2/common/schema.yaml",
			64,
			11,
		),
	}

	result, stats := DiffViolationsValuesWithOriginBases(
		original,
		newResults,
		"/workspace/folder1/apis/openapi.yaml",
		"/workspace/folder2/apis/openapi.yaml",
	)

	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValuesWithOriginBases_KeepsSiblingExternalRefsInDifferentDirectoriesDistinct(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.components.schemas.Customer.properties.name",
			message,
			"/workspace/folder1/common/a.yaml",
			64,
			11,
		),
	}
	newResults := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.components.schemas.Order.properties.name",
			message,
			"/workspace/folder2/other/a.yaml",
			64,
			11,
		),
	}

	result, stats := DiffViolationsValuesWithOriginBases(
		original,
		newResults,
		"/workspace/folder1/apis/openapi.yaml",
		"/workspace/folder2/apis/openapi.yaml",
	)

	require.Len(t, result, 1)
	assert.Equal(t, "/workspace/folder2/other/a.yaml", result[0].Origin.AbsoluteLocation)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValues_OriginFallsBackToStartNodePosition(t *testing.T) {
	message := "schema property `error-code` is missing `examples` or `example`"
	original := []model.RuleFunctionResult{
		{
			RuleId:    "oas3-missing-example",
			Path:      "",
			Message:   message,
			Rule:      &model.Rule{Id: "oas3-missing-example"},
			StartNode: &yaml.Node{Line: 321, Column: 9},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "api-common.yaml",
			},
		},
	}
	newResults := []model.RuleFunctionResult{
		{
			RuleId:    "oas3-missing-example",
			Path:      "",
			Message:   message,
			Rule:      &model.Rule{Id: "oas3-missing-example"},
			StartNode: &yaml.Node{Line: 321, Column: 9},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "api-common.yaml",
			},
		},
	}

	result, stats := DiffViolationsValues(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValues_StatsCorrect(t *testing.T) {
	original := []model.RuleFunctionResult{
		makeResult("rule-1", "$.a", "msg-a"),
		makeResult("rule-2", "$.b", "msg-b"),
		makeResult("rule-2", "$.c", "msg-c"),
	}
	newResults := []model.RuleFunctionResult{
		makeResult("rule-1", "$.a", "msg-a"), // suppressed
		makeResult("rule-2", "$.b", "msg-b"), // suppressed
		makeResult("rule-2", "$.c", "msg-c"), // suppressed
		makeResult("rule-2", "$.d", "msg-d"), // new
		makeResult("rule-3", "$.e", "msg-e"), // new
	}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Len(t, result, 2)
	assert.Equal(t, 5, stats.TotalResultsBefore)
	assert.Equal(t, 2, stats.TotalResultsAfter)
	assert.Equal(t, 3, stats.ResultsDropped)

	// rule-1: 1 before, 0 after (fully filtered)
	assert.Contains(t, stats.RulesFullyFiltered, "rule-1")

	// rule-2: 3 before, 1 after (partial: 2 dropped)
	assert.Equal(t, 2, stats.RulesPartialFiltered["rule-2"])
}

func TestDiffViolationsMixed_Basic(t *testing.T) {
	original := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"),
	}
	newResults := []*model.RuleFunctionResult{
		makeResultPtr("rule-1", "$.paths./items.post", "missing description"), // suppressed
		makeResultPtr("rule-2", "$.info", "missing contact"),                  // new
	}
	result, stats := DiffViolationsMixed(original, newResults)
	require.Len(t, result, 1)
	assert.Equal(t, "rule-2", result[0].RuleId)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsMixed_PathsSuppressExternalRefPrimaryDrift(t *testing.T) {
	message := "schema of type `string` must specify `format`, `const`, `enum` or `pattern`"
	pathA := "$.paths['/a'].patch.requestBody.content['application/json'].schema.items.properties['path']"
	pathB := "$.paths['/b'].patch.requestBody.content['application/json'].schema.items.properties['path']"
	original := []model.RuleFunctionResult{
		{
			RuleId:  "owasp-string-restricted",
			Path:    pathA,
			Paths:   []string{pathA, pathB},
			Message: message,
			Rule:    &model.Rule{Id: "owasp-string-restricted"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "api-common.yaml",
				Line:             258,
				Column:           11,
			},
		},
	}
	newResults := []*model.RuleFunctionResult{
		{
			RuleId:  "owasp-string-restricted",
			Path:    pathB,
			Paths:   []string{pathB, pathA},
			Message: message,
			Rule:    &model.Rule{Id: "owasp-string-restricted"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "api-common.yaml",
				Line:             258,
				Column:           11,
			},
		},
	}

	result, stats := DiffViolationsMixed(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsMixed_PathIdentitySuppressesOriginLineShift(t *testing.T) {
	message := "operation method `GET` at path `/pets` is missing a description or summary"
	original := []model.RuleFunctionResult{
		makeOriginResult(
			"operation-description",
			"$.paths['/pets'].get",
			message,
			"/workspace/api.yaml",
			14,
			7,
		),
	}
	newResults := []*model.RuleFunctionResult{
		makeOriginResultPtr(
			"operation-description",
			"$.paths['/pets'].get",
			message,
			"/workspace/api.yaml",
			22,
			7,
		),
	}

	result, stats := DiffViolationsMixed(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsMixed_PathsSuppressMirroredDirectoryPrimaryDrift(t *testing.T) {
	message := "minLength must be defined"
	pathA := "$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']"
	pathB := "$.paths['/b'].get.responses['200'].content['application/json'].schema.properties['name']"
	original := []model.RuleFunctionResult{
		{
			RuleId:  "check-string-attribute-minlength",
			Path:    pathA,
			Paths:   []string{pathA, pathB},
			Message: message,
			Rule:    &model.Rule{Id: "check-string-attribute-minlength"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "/workspace/folder1/openapi-3.0/test/common/test-common.yaml",
				Line:             64,
				Column:           11,
			},
		},
	}
	newResults := []*model.RuleFunctionResult{
		{
			RuleId:  "check-string-attribute-minlength",
			Path:    pathB,
			Paths:   []string{pathB, pathA},
			Message: message,
			Rule:    &model.Rule{Id: "check-string-attribute-minlength"},
			Origin: &index.NodeOrigin{
				AbsoluteLocation: "/workspace/folder2/openapi-3.0/test/common/test-common.yaml",
				Line:             64,
				Column:           11,
			},
		},
	}

	result, stats := DiffViolationsMixed(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
	assert.Equal(t, "/workspace/folder1/openapi-3.0/test/common/test-common.yaml", original[0].Origin.AbsoluteLocation)
	assert.Equal(t, "/workspace/folder2/openapi-3.0/test/common/test-common.yaml", newResults[0].Origin.AbsoluteLocation)
}

func TestDiffViolationsMixed_FallbackOriginPreservesDirectoryContext(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"",
			message,
			"/old/common/schema.yaml",
			64,
			11,
		),
	}
	newResults := []*model.RuleFunctionResult{
		makeOriginResultPtr(
			"check-string-attribute-minlength",
			"",
			message,
			"/new/other/schema.yaml",
			64,
			11,
		),
	}

	result, stats := DiffViolationsMixed(original, newResults)

	require.Len(t, result, 1)
	assert.Equal(t, "/new/other/schema.yaml", result[0].Origin.AbsoluteLocation)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsMixedWithOriginBases_SuppressesMirroredFileWhenFileSetsDiffer(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']",
			message,
			"/workspace/folder1/common/a.yaml",
			64,
			11,
		),
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.paths['/b'].get.responses['200'].content['application/json'].schema.properties['name']",
			message,
			"/workspace/folder1/other/b.yaml",
			72,
			11,
		),
	}
	newResults := []*model.RuleFunctionResult{
		makeOriginResultPtr(
			"check-string-attribute-minlength",
			"$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']",
			message,
			"/workspace/folder2/common/a.yaml",
			64,
			11,
		),
	}

	result, stats := DiffViolationsMixedWithOriginBases(
		original,
		newResults,
		"/workspace/folder1/openapi.yaml",
		"/workspace/folder2/openapi.yaml",
	)

	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsMixedWithOriginBases_SuppressesSiblingExternalRefOutsideSpecDirectory(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']",
			message,
			"/workspace/folder1/common/schema.yaml",
			64,
			11,
		),
	}
	newResults := []*model.RuleFunctionResult{
		makeOriginResultPtr(
			"check-string-attribute-minlength",
			"$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']",
			message,
			"/workspace/folder2/common/schema.yaml",
			64,
			11,
		),
	}

	result, stats := DiffViolationsMixedWithOriginBases(
		original,
		newResults,
		"/workspace/folder1/apis/openapi.yaml",
		"/workspace/folder2/apis/openapi.yaml",
	)

	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsMixed_NilInNew(t *testing.T) {
	original := []model.RuleFunctionResult{}
	newResults := []*model.RuleFunctionResult{
		nil,
		makeResultPtr("rule-1", "$.a", "msg"),
	}
	result, stats := DiffViolationsMixed(original, newResults)
	// nil entries produce an empty key; since original also has no such entry, it stays
	assert.Len(t, result, 2)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestExtractPath(t *testing.T) {
	assert.Equal(t, "$.a", extractPath("$.a", nil))
	assert.Equal(t, "$.a\x00$.b", extractPath("$.a", []string{"$.b"}))
	assert.Equal(t, "$.b", extractPath("", []string{"$.b"}))
	assert.Equal(t, "", extractPath("", nil))
	assert.Equal(t, "", extractPath("", []string{}))
	assert.Equal(t, "$.a\x00$.b", extractPath("$.b", []string{"$.a", "$.b"}))
}
