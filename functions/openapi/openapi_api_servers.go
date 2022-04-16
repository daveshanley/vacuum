// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"net/url"
	"strings"
)

// APIServers checks that each operation only has a single tag.
type APIServers struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the APIServers rule.
func (as APIServers) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "api_servers",
	}
}

// RunRule will execute the APIServers rule, based on supplied context and a supplied []*yaml.Node slice.
func (as APIServers) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	rootServers := context.Index.GetAllRootServers()
	rootServersNode := context.Index.GetRootServersNode()

	// does the 'servers' node exist?
	if rootServersNode == nil && rootServers == nil {
		results = append(results, model.RuleFunctionResult{
			Message:   "No servers defined for the specification",
			StartNode: context.Index.GetRootNode(),
			EndNode:   utils.FindLastChildNode(context.Index.GetRootNode()),
			Path:      "$.servers",
			Rule:      context.Rule,
		})
	}

	// check that we actually have something defined by the node.
	if rootServersNode != nil && len(rootServers) < 1 {
		results = append(results, model.RuleFunctionResult{
			Message:   "Servers definition is empty, contains no servers!",
			StartNode: rootServersNode,
			EndNode:   rootServersNode,
			Path:      "$.servers",
			Rule:      context.Rule,
		})
	}

	// check servers contains a URL and the URL is valid.
	for i, serverRef := range rootServers {
		urlLabelNode, urlNode := utils.FindKeyNode("url", serverRef.Node.Content)
		if urlNode == nil {
			results = append(results, model.RuleFunctionResult{
				Message:   "Server definition is missing a URL",
				StartNode: serverRef.Node,
				EndNode:   utils.FindLastChildNode(serverRef.Node),
				Path:      fmt.Sprintf("$.servers[%d]", i),
				Rule:      context.Rule,
			})
			continue
		}

		// check url is valid
		parsed, err := url.Parse(urlNode.Value)
		if err != nil {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("Server URL cannot be parsed: %s", err.Error()),
				StartNode: urlLabelNode,
				EndNode:   urlNode,
				Path:      fmt.Sprintf("$.servers[%d].url", i),
				Rule:      context.Rule,
			})
			continue
		}

		// check the host and the path are not empty.
		if parsed.Host == "" && parsed.Path == "" {
			msg := "Server URL is not valid: no hostname or path provided"
			results = append(results, model.RuleFunctionResult{
				Message:   msg,
				StartNode: urlLabelNode,
				EndNode:   urlNode,
				Path:      fmt.Sprintf("$.servers[%d].url", i),
				Rule:      context.Rule,
			})
			continue
		}

		// check the path doesn't have a trailing slash.
		if parsed.Path != "" && strings.LastIndex(parsed.Path, "/") == len(parsed.Path)-1 {
			msg := "Server URL is not valid: must not have a trailing slash"
			results = append(results, model.RuleFunctionResult{
				Message:   msg,
				StartNode: urlLabelNode,
				EndNode:   urlNode,
				Path:      fmt.Sprintf("$.servers[%d].url", i),
				Rule:      context.Rule,
			})
			continue
		}
	}

	// TODO: check operation server references, the above needs to be broken down into a function
	// and repeated for each operation server override, remember each path and verb can override servers.

	return results
}
