// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// GetAllOperationsJSONPath wil return a string that can be used as a query for extracting all OpenAPI operations.
func GetAllOperationsJSONPath() string {
	return "$.paths[*]['get','put','post','delete','options','head','patch','trace']"
}

// GetTagsFromRoot will extract all tag nodes from the root of an OpenAPI document.
func GetTagsFromRoot(nodes []*yaml.Node) []*yaml.Node {
	for _, node := range nodes {
		_, tags := utils.FindFirstKeyNode("tags", node.Content, 0)
		if tags != nil && len(tags.Content) > 0 {
			return tags.Content
		}
	}
	return nil
}

// GetOperationsFromRoot will extract all operation (paths nodes) from the root of an OpenAPI document.
func GetOperationsFromRoot(nodes []*yaml.Node) []*yaml.Node {
	for _, node := range nodes {
		_, paths := utils.FindFirstKeyNode("paths", node.Content, 0)
		if paths != nil && len(paths.Content) > 0 {
			return paths.Content
		}
	}
	return nil
}
