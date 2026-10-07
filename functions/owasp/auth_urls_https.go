package owasp

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
)

// AuthURLsHTTPS checks transport declarations for OAuth and OpenID endpoints.
type AuthURLsHTTPS struct{}

// GetSchema returns the schema of the AuthURLsHTTPS rule.
func (AuthURLsHTTPS) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspAuthURLsHTTPS"}
}

// GetCategory returns the OWASP function category.
func (AuthURLsHTTPS) GetCategory() string { return model.FunctionCategoryOWASP }

// RunRule checks declared endpoint URLs without contacting identity providers.
func (AuthURLsHTTPS) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.DrDocument == nil || context.DrDocument.V3Document == nil || context.DrDocument.V3Document.Components == nil {
		return nil
	}
	var results []model.RuleFunctionResult
	for pair := context.DrDocument.V3Document.Components.SecuritySchemes.First(); pair != nil; pair = pair.Next() {
		scheme := pair.Value()
		type endpoint struct {
			field low.NodeReference[string]
			path  string
		}
		var endpoints []endpoint
		if scheme.Value.Type == "openIdConnect" {
			endpoints = append(endpoints, endpoint{scheme.Value.GoLow().OpenIdConnectUrl, scheme.GenerateJSONPath()})
		}
		if scheme.Value.Type == "oauth2" && scheme.Flows != nil {
			for _, flow := range []*v3.OAuthFlow{scheme.Flows.Implicit, scheme.Flows.Password, scheme.Flows.ClientCredentials, scheme.Flows.AuthorizationCode, scheme.Flows.Device} {
				if flow == nil {
					continue
				}
				value := flow.Value.GoLow()
				for _, field := range []low.NodeReference[string]{value.AuthorizationUrl, value.TokenUrl, value.RefreshUrl} {
					endpoints = append(endpoints, endpoint{field, flow.GenerateJSONPath()})
				}
			}
		}
		for _, endpoint := range endpoints {
			field := endpoint.field
			if field.KeyNode == nil {
				continue
			}
			parsed, err := url.Parse(field.Value)
			if err != nil || parsed.Scheme == "" || strings.EqualFold(parsed.Scheme, "https") {
				continue
			}
			result := model.RuleFunctionResult{
				Message:   utils.SuppliedOrDefault(context.Rule.Message, "authentication endpoint URLs should use TLS (https)"),
				StartNode: field.KeyNode, EndNode: utils.BuildEndNode(field.KeyNode),
				Path: fmt.Sprintf("%s.%s", endpoint.path, field.KeyNode.Value), Rule: context.Rule,
			}
			scheme.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
			results = append(results, result)
		}
	}
	return results
}
