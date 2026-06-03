// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	"go.yaml.in/yaml/v4"
)

// ChannelParameters validates channel address parameter declarations.
type ChannelParameters struct{}

// GetSchema returns the AsyncAPI channel-parameters function schema.
func (c ChannelParameters) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiChannelParameters"}
}

// GetCategory returns the AsyncAPI function category.
func (c ChannelParameters) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule checks that every `{parameter}` in a channel address is declared and
// every declared parameter is used by the address.
func (c ChannelParameters) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		_, address := mappingValue(node, "address")
		_, parameters := mappingValue(node, "parameters")
		if address == nil {
			continue
		}
		results = append(results, validateTemplateVariables(context, node, address.Value, parameters, "Channel address")...)
	}
	return results
}

// ChannelServers validates channel server references.
type ChannelServers struct{}

// GetSchema returns the AsyncAPI channel-servers function schema.
func (c ChannelServers) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiChannelServers"}
}

// GetCategory returns the AsyncAPI function category.
func (c ChannelServers) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule checks that channel server references resolve to root or component servers.
func (c ChannelServers) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	root := rootNode(context)
	_, rootServersNode := mappingValue(root, "servers")
	rootServers := mappingKeys(rootServersNode)
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		_, servers := mappingValue(node, "servers")
		if servers == nil || servers.Kind != yaml.SequenceNode {
			continue
		}
		for _, server := range servers.Content {
			ref := refValue(server)
			if ref == "" {
				continue
			}
			if name := rootRefName(ref, "servers"); name != "" {
				if rootServers[name] == nil {
					results = append(results, result(context, server, nodePath(context, server, ""), fmt.Sprintf("Channel server `%s` is not defined.", name)))
				}
				continue
			}
			results = append(results, result(context, server, nodePath(context, server, ""), "Channel servers must reference `#/servers`."))
		}
	}
	return results
}

// ServerVariables validates server host and pathname variables.
type ServerVariables struct{}

// GetSchema returns the AsyncAPI server-variables function schema.
func (s ServerVariables) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiServerVariables"}
}

// GetCategory returns the AsyncAPI function category.
func (s ServerVariables) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule checks that variables used by server host/pathname templates are
// declared in the server variables map and that declared variables are used.
func (s ServerVariables) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		_, host := mappingValue(node, "host")
		_, pathname := mappingValue(node, "pathname")
		_, variables := mappingValue(node, "variables")
		template := ""
		if host != nil {
			template += host.Value
		}
		if pathname != nil {
			template += pathname.Value
		}
		if template == "" {
			continue
		}
		results = append(results, validateTemplateVariables(context, node, template, variables, "Server")...)
	}
	return results
}

// Security validates AsyncAPI security scheme references.
type Security struct{}

// GetSchema returns the AsyncAPI security function schema.
func (s Security) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiSecurity"}
}

// GetCategory returns the AsyncAPI function category.
func (s Security) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule checks each security entry against components.securitySchemes.
func (s Security) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	root := rootNode(context)
	knownSchemes := mappingKeys(componentMap(root, "securitySchemes"))
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		if ref := refValue(node); ref != "" {
			name := componentRefName(ref, "securitySchemes")
			if name == "" {
				results = append(results, result(context, node, nodePath(context, node, ""), "Security scheme references must target `#/components/securitySchemes`."))
				continue
			}
			if knownSchemes[name] == nil {
				results = append(results, result(context, node, nodePath(context, node, ""), fmt.Sprintf("Security scheme `%s` is not defined.", name)))
			}
			continue
		}

		for _, entry := range mappingEntries(node) {
			name := entry[0].Value
			if knownSchemes[name] == nil {
				results = append(results, result(context, entry[0], nodePath(context, entry[0], ""), fmt.Sprintf("Security scheme `%s` is not defined.", name)))
			}
		}
	}
	return results
}
