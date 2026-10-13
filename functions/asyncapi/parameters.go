// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
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

// RunRule validates inline and referenced Security Scheme Objects.
func (s Security) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	if ctx.GetOptionsStringMap()["source"] != "owner" {
		return validateSecuritySchemes(nodes, ctx)
	}
	var results []model.RuleFunctionResult
	seen := make(map[*yaml.Node]bool)
	var visit func(*yaml.Node, model.RuleFunctionContext)
	visit = func(source *yaml.Node, scope model.RuleFunctionContext) {
		node, scope := resolveObject(source, scope)
		if node == nil || seen[node] {
			return
		}
		seen[node] = true
		_, security := mappingValue(node, "security")
		if security != nil {
			if security.Kind != yaml.SequenceNode {
				results = append(results, result(scope, security, "", "Security must be an array of Security Scheme Objects or references."))
			} else {
				results = append(results, validateSecuritySchemes(security.Content, scope)...)
			}
		}
		_, traits := mappingValue(node, "traits")
		if traits != nil && traits.Kind == yaml.SequenceNode {
			for _, trait := range traits.Content {
				visit(trait, scope)
			}
		}
	}
	for _, source := range nodes {
		if strings.HasPrefix(nodePath(ctx, source, ""), "$.components.securitySchemes") {
			results = append(results, validateSecuritySchemes([]*yaml.Node{source}, ctx)...)
		} else {
			visit(source, ctx)
		}
	}
	return results
}

func validateSecuritySchemes(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, source := range nodes {
		node, scope := resolveObject(source, ctx)
		if node == nil {
			results = append(results, result(ctx, source, "", fmt.Sprintf("Security scheme `%s` is not defined.", refValue(source))))
			continue
		}
		add := func(field, message string) {
			_, target := mappingValue(node, field)
			if target == nil {
				target = node
			}
			results = append(results, result(scope, target, "", message))
		}
		text := func(field string) string {
			_, value := mappingValue(node, field)
			if value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
				return ""
			}
			return value.Value
		}
		kind := text("type")
		switch kind {
		case "userPassword", "apiKey", "X509", "symmetricEncryption", "asymmetricEncryption", "httpApiKey", "http", "oauth2", "openIdConnect", "plain", "scramSha256", "scramSha512", "gssapi":
		default:
			add("type", "Security scheme must define a supported string `type`.")
			continue
		}
		switch kind {
		case "apiKey":
			if location := text("in"); location != "user" && location != "password" {
				add("in", "API key security scheme `in` must be `user` or `password`.")
			}
		case "httpApiKey":
			if text("name") == "" {
				add("name", "HTTP API key security scheme must define `name`.")
			}
			if location := text("in"); location != "header" && location != "query" && location != "cookie" {
				add("in", "HTTP API key security scheme `in` must be `header`, `query` or `cookie`.")
			}
		case "http":
			if text("scheme") == "" {
				add("scheme", "HTTP security scheme must define `scheme`.")
			}
		case "oauth2":
			_, flows := mappingValue(node, "flows")
			if flows == nil || flows.Kind != yaml.MappingNode {
				add("flows", "OAuth2 security scheme must define a `flows` object.")
			}
		case "openIdConnect":
			address, err := url.Parse(text("openIdConnectUrl"))
			if err != nil || address == nil || !address.IsAbs() {
				add("openIdConnectUrl", "OpenID Connect security scheme must define an absolute `openIdConnectUrl`.")
			}
		}
	}
	return results
}
