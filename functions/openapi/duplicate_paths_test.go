// Copyright 2024 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestDuplicatePaths_GetSchema(t *testing.T) {
	def := DuplicatePaths{}
	assert.Equal(t, "duplicatePaths", def.GetSchema().Name)
}

func TestDuplicatePaths_RunRule_NoDuplicates(t *testing.T) {
	def := DuplicatePaths{}
	ctx := model.RuleFunctionContext{}

	yml := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths:
  /api/users:
    get:
      summary: Get users
  /api/posts:
    get:
      summary: Get posts`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, err)

	nodes := []*yaml.Node{&rootNode}
	res := def.RunRule(nodes, ctx)
	assert.Len(t, res, 0)
}

func TestDuplicatePaths_RunRule_WithDuplicates(t *testing.T) {
	def := DuplicatePaths{}
	ctx := model.RuleFunctionContext{}

	yml := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths:
  /api/endpoint/{id}:
    get:
      summary: Get endpoint
  /api/endpoint/{id}:
    post:
      summary: Create endpoint`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, err)

	nodes := []*yaml.Node{&rootNode}
	res := def.RunRule(nodes, ctx)
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "duplicate path '/api/endpoint/{id}' found")
	assert.Contains(t, res[0].Message, "only the last definition will be used")
}

func TestDuplicatePaths_RunRule_MultipleDuplicates(t *testing.T) {
	def := DuplicatePaths{}
	ctx := model.RuleFunctionContext{}

	yml := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths:
  /api/users:
    get:
      summary: Get users - first
  /api/users:
    post:
      summary: Create user
  /api/users:
    delete:
      summary: Delete user - last
  /api/posts:
    get:
      summary: Get posts - first
  /api/posts:
    post:
      summary: Create post - last`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, err)

	nodes := []*yaml.Node{&rootNode}
	res := def.RunRule(nodes, ctx)
	assert.Len(t, res, 3) // 2 duplicates for /api/users + 1 duplicate for /api/posts

	// Check that all expected duplicates are found
	messages := make([]string, len(res))
	for i, result := range res {
		messages[i] = result.Message
	}

	// Should find duplicates for both paths
	usersCount := 0
	postsCount := 0
	for _, msg := range messages {
		if strings.Contains(msg, "/api/users") {
			usersCount++
		}
		if strings.Contains(msg, "/api/posts") {
			postsCount++
		}
	}
	assert.Equal(t, 2, usersCount, "Expected 2 duplicate reports for /api/users")
	assert.Equal(t, 1, postsCount, "Expected 1 duplicate report for /api/posts")
}

func TestDuplicatePaths_RunRule_NoPaths(t *testing.T) {
	def := DuplicatePaths{}
	ctx := model.RuleFunctionContext{}

	yml := `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, err)

	nodes := []*yaml.Node{&rootNode}
	res := def.RunRule(nodes, ctx)
	assert.Len(t, res, 0)
}

func TestDuplicatePaths_RunRule_EmptyNodes(t *testing.T) {
	def := DuplicatePaths{}
	ctx := model.RuleFunctionContext{}

	res := def.RunRule([]*yaml.Node{}, ctx)
	assert.Len(t, res, 0)
}

