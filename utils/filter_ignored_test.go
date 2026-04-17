// Copyright 2025 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package utils

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestFilterIgnoredResults(t *testing.T) {
	results := []model.RuleFunctionResult{
		{Path: "a/b/c", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a/b", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a/b/c", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a/b", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a/b/c", Rule: &model.Rule{Id: "ZZZ"}},
		{Path: "a/b", Rule: &model.Rule{Id: "ZZZ"}},
		{Path: "a", Rule: &model.Rule{Id: "ZZZ"}},
	}

	igItems := model.IgnoredItems{
		"XXX": []string{"a/b/c"},
		"YYY": []string{"a/b"},
	}

	filtered := FilterIgnoredResults(results, igItems)

	assert.Len(t, filtered, 7)

	// Check that the ignored items are not in the result
	for _, r := range filtered {
		if r.Rule.Id == "XXX" {
			assert.NotEqual(t, "a/b/c", r.Path)
		}
		if r.Rule.Id == "YYY" {
			assert.NotEqual(t, "a/b", r.Path)
		}
	}
}

func TestFilterIgnoredResultsPtr(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{Path: "a/b/c", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a/b", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a/b/c", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a/b", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a/b/c", Rule: &model.Rule{Id: "ZZZ"}},
		{Path: "a/b", Rule: &model.Rule{Id: "ZZZ"}},
		{Path: "a", Rule: &model.Rule{Id: "ZZZ"}},
	}

	igItems := model.IgnoredItems{
		"XXX": []string{"a/b/c"},
		"YYY": []string{"a/b"},
	}

	filtered := FilterIgnoredResultsPtr(results, igItems)

	assert.Len(t, filtered, 7)

	// Check that the ignored items are not in the result
	for _, r := range filtered {
		if r.Rule.Id == "XXX" {
			assert.NotEqual(t, "a/b/c", r.Path)
		}
		if r.Rule.Id == "YYY" {
			assert.NotEqual(t, "a/b", r.Path)
		}
	}
}

func TestFilterIgnoredResultsWithPaths(t *testing.T) {
	results := []model.RuleFunctionResult{
		{Path: "main", Paths: []string{"a/b/c", "d/e/f"}, Rule: &model.Rule{Id: "XXX"}},
		{Path: "main", Paths: []string{"g/h/i"}, Rule: &model.Rule{Id: "XXX"}},
		{Path: "main", Rule: &model.Rule{Id: "YYY"}},
	}

	igItems := model.IgnoredItems{
		"XXX": []string{"d/e/f"},
		"YYY": []string{"main"},
	}

	filtered := FilterIgnoredResults(results, igItems)

	// First result should be filtered because one of its paths matches
	// Second result should not be filtered
	// Third result should be filtered because its path matches
	assert.Len(t, filtered, 1)
	assert.Equal(t, "XXX", filtered[0].Rule.Id)
	assert.Contains(t, filtered[0].Paths, "g/h/i")
}

func TestFilterIgnoredResultsWithOptions_ExpressionMatchesWildcard(t *testing.T) {
	spec := []byte(`
openapi: 3.1.0
info:
  title: Test API
  version: "1"
paths:
  /users:
    get:
      responses:
        "200":
          description: ok
  /orders:
    get:
      responses:
        "200":
          description: ok
  /users/create:
    post:
      responses:
        "200":
          description: ok
`)
	root := parseIgnoreMatcherRoot(t, spec)

	results := []model.RuleFunctionResult{
		{Path: "$.paths['/users'].get", Rule: &model.Rule{Id: "OP"}},
		{Path: "$.paths['/orders'].get", Rule: &model.Rule{Id: "OP"}},
		{Path: "$.paths['/users/create'].post", Rule: &model.Rule{Id: "OP"}},
		{Path: "$.paths['/users'].get", Rule: &model.Rule{Id: "OTHER"}},
	}

	filtered := FilterIgnoredResultsWithOptions(results, model.IgnoredItems{
		"OP": []string{"$.paths[*].get"},
	}, IgnoreMatcherOptions{
		RootNode: root,
	})

	assert.Len(t, filtered, 2)
	assert.Equal(t, "$.paths['/users/create'].post", filtered[0].Path)
	assert.Equal(t, "$.paths['/users'].get", filtered[1].Path)
	assert.Equal(t, "OTHER", filtered[1].Rule.Id)
}

func TestFilterIgnoredResultsWithOptions_ExpressionMatchesUsingSpecBytesFallback(t *testing.T) {
	spec := []byte(`
openapi: 3.1.0
info:
  title: Test API
  version: "1"
paths:
  /users:
    get:
      responses:
        "200":
          description: ok
  /orders:
    get:
      responses:
        "200":
          description: ok
`)

	results := []model.RuleFunctionResult{
		{Path: "$.paths['/users'].get", Rule: &model.Rule{Id: "OP"}},
		{Path: "$.paths['/orders'].get", Rule: &model.Rule{Id: "OP"}},
	}

	filtered := FilterIgnoredResultsWithOptions(results, model.IgnoredItems{
		"OP": []string{"$.paths[*].get"},
	}, IgnoreMatcherOptions{
		SpecBytes: spec,
	})

	assert.Empty(t, filtered)
}

func TestFilterIgnoredResultsWithOptions_ExpressionMatchesAlternatePaths(t *testing.T) {
	spec := []byte(`
openapi: 3.1.0
info:
  title: Test API
  version: "1"
paths:
  /users:
    get:
      parameters:
        - name: q
          in: query
          required: false
          schema:
            type: string
      responses:
        "200":
          description: ok
`)

	results := []model.RuleFunctionResult{
		{
			Path:  "$.components.parameters['Shared']",
			Paths: []string{"$.paths['/users'].get.parameters[0]"},
			Rule:  &model.Rule{Id: "PARAM"},
		},
	}

	filtered := FilterIgnoredResultsWithOptions(results, model.IgnoredItems{
		"PARAM": []string{"$.paths[*].get.parameters[*]"},
	}, IgnoreMatcherOptions{
		SpecBytes: spec,
	})

	assert.Empty(t, filtered)
}

func TestFilterIgnoredResultsWithOptions_InvalidExpressionStillSupportsLiteralMatches(t *testing.T) {
	spec := []byte(`
openapi: 3.1.0
info:
  title: Test API
  version: "1"
`)

	results := []model.RuleFunctionResult{
		{Path: "a/b/c", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a/b", Rule: &model.Rule{Id: "XXX"}},
	}

	filtered := FilterIgnoredResultsWithOptions(results, model.IgnoredItems{
		"XXX": []string{"a/b/c"},
	}, IgnoreMatcherOptions{
		SpecBytes: spec,
	})

	assert.Len(t, filtered, 1)
	assert.Equal(t, "a/b", filtered[0].Path)
}

func parseIgnoreMatcherRoot(t *testing.T, spec []byte) *yaml.Node {
	t.Helper()

	var root yaml.Node
	err := yaml.Unmarshal(spec, &root)
	assert.NoError(t, err)
	return &root
}
