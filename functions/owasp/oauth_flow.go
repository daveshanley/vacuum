package owasp

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	libutils "github.com/pb33f/libopenapi/utils"
)

const oauthFlowOptionsError = "'owaspOAuthFlow' requires 'flow' to be 'password' or 'implicit'"

// OAuthFlow checks for a configured insecure OAuth flow.
type OAuthFlow struct{}

// GetSchema defines the required flow option.
func (OAuthFlow) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:     "owaspOAuthFlow",
		Required: []string{"flow"},
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "flow",
				Description: "OAuth flow to reject: password or implicit",
			},
		},
		ErrorMessage: oauthFlowOptionsError,
	}
}

// GetCategory returns the OAuthFlow function category.
func (OAuthFlow) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule reports the selected OAuth flow without changing HTTP scheme checks.
func (OAuthFlow) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.DrDocument == nil || context.DrDocument.V3Document == nil {
		return nil
	}
	var flowName string
	if casted, ok := libutils.ExtractValueFromInterfaceMap("flow", context.Options).(string); ok {
		flowName = casted
	}
	if flowName != "password" && flowName != "implicit" {
		node := context.DrDocument.V3Document.Document.GoLow().Version.KeyNode
		return []model.RuleFunctionResult{{
			Message:   oauthFlowOptionsError,
			StartNode: node,
			EndNode:   utils.BuildEndNode(node),
			Path:      "$",
			Rule:      context.Rule,
		}}
	}
	if context.DrDocument.V3Document.Components == nil {
		return nil
	}
	var results []model.RuleFunctionResult
	for pair := context.DrDocument.V3Document.Components.SecuritySchemes.First(); pair != nil; pair = pair.Next() {
		scheme := pair.Value()
		if scheme.Value.Type != "oauth2" || scheme.Flows == nil {
			continue
		}
		var flow *v3.OAuthFlow
		switch flowName {
		case "password":
			flow = scheme.Flows.Password
		case "implicit":
			flow = scheme.Flows.Implicit
		}
		if flow == nil {
			continue
		}
		node := flow.Value.GoLow().RootNode
		result := model.RuleFunctionResult{
			Message:   utils.SuppliedOrDefault(context.Rule.Message, fmt.Sprintf("OAuth %s flow should not be used; use authorization code with PKCE for user authorization", flowName)),
			StartNode: node,
			EndNode:   utils.BuildEndNode(node),
			Path:      flow.GenerateJSONPath(),
			Rule:      context.Rule,
		}
		flow.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		results = append(results, result)
	}
	return results
}
