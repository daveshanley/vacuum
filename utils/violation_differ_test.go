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

func TestDiffViolationsValues_OriginSuppressesExternalRefPathDrift(t *testing.T) {
	message := "schema of type `string` must specify `format`, `const`, `enum` or `pattern`"
	original := []model.RuleFunctionResult{
		{
			RuleId:  "owasp-string-restricted",
			Path:    "$.paths['/a'].patch.requestBody.content['application/json'].schema.items.properties['path']",
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
			Path:    "$.paths['/b'].patch.requestBody.content['application/json'].schema.items.properties['path']",
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

func TestDiffViolationsValues_CanonicalOriginSuppressesMirroredDirectoryDrift(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		{
			RuleId:  "check-string-attribute-minlength",
			Path:    "$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']",
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
			Path:    "$.paths['/b'].get.responses['200'].content['application/json'].schema.properties['name']",
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
			"$.components.schemas.Customer.properties.name",
			message,
			"/old/common/schema.yaml",
			64,
			11,
		),
	}
	newResults := []model.RuleFunctionResult{
		makeOriginResult(
			"check-string-attribute-minlength",
			"$.components.schemas.Customer.properties.name",
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
			Path:      "$.paths['/old'].get.responses['400'].content['*/*'].schema.properties['error-code']",
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
			Path:      "$.paths['/new'].get.responses['400'].content['*/*'].schema.properties['error-code']",
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

func TestDiffViolationsMixed_OriginSuppressesExternalRefPathDrift(t *testing.T) {
	message := "schema of type `string` must specify `format`, `const`, `enum` or `pattern`"
	original := []model.RuleFunctionResult{
		{
			RuleId:  "owasp-string-restricted",
			Path:    "$.paths['/a'].patch.requestBody.content['application/json'].schema.items.properties['path']",
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
			Path:    "$.paths['/b'].patch.requestBody.content['application/json'].schema.items.properties['path']",
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

func TestDiffViolationsMixed_CanonicalOriginSuppressesMirroredDirectoryDrift(t *testing.T) {
	message := "minLength must be defined"
	original := []model.RuleFunctionResult{
		{
			RuleId:  "check-string-attribute-minlength",
			Path:    "$.paths['/a'].get.responses['200'].content['application/json'].schema.properties['name']",
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
			Path:    "$.paths['/b'].get.responses['200'].content['application/json'].schema.properties['name']",
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
			"$.components.schemas.Customer.properties.name",
			message,
			"/old/common/schema.yaml",
			64,
			11,
		),
	}
	newResults := []*model.RuleFunctionResult{
		makeOriginResultPtr(
			"check-string-attribute-minlength",
			"$.components.schemas.Customer.properties.name",
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
