package owasp

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

type CheckSecurity struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the CheckSecurity rule.
func (cd CheckSecurity) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspCheckSecurity"}
}

// GetCategory returns the category of the CheckSecurity rule.
func (cd CheckSecurity) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the CheckSecurity rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd CheckSecurity) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var nullable bool
	nullableMap := utils.ExtractValueFromInterfaceMap("nullable", context.Options)
	if castedNullable, ok := nullableMap.(bool); ok {
		nullable = castedNullable
	}

	var methods []string
	methodsMap := utils.ExtractValueFromInterfaceMap("methods", context.Options)
	if castedMethods, ok := methodsMap.([]string); ok {
		methods = castedMethods
	}

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}
	drDoc := context.DrDocument.V3Document
	globalSecurity := drDoc.Security

	if drDoc.Paths != nil && drDoc.Paths.PathItems != nil {

		for pathPairs := drDoc.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
			path := pathPairs.Key()
			pathItem := pathPairs.Value()
			for opPairs := pathItem.GetOperations().First(); opPairs != nil; opPairs = opPairs.Next() {
				opValue := opPairs.Value()
				opType := opPairs.Key()

				if !slices.Contains(methods, opType) {
					continue
				}

				var opNode *yaml.Node
				var op *drV3.Operation

				switch opType {
				case v3.GetLabel:
					opNode = pathPairs.Value().Value.GoLow().Get.KeyNode
					op = pathPairs.Value().Get
				case v3.PutLabel:
					opNode = pathPairs.Value().Value.GoLow().Put.KeyNode
					op = pathPairs.Value().Put
				case v3.PostLabel:
					opNode = pathPairs.Value().Value.GoLow().Post.KeyNode
					op = pathPairs.Value().Post
				case v3.DeleteLabel:
					opNode = pathPairs.Value().Value.GoLow().Delete.KeyNode
					op = pathPairs.Value().Delete
				case v3.OptionsLabel:
					opNode = pathPairs.Value().Value.GoLow().Options.KeyNode
					op = pathPairs.Value().Options
				case v3.HeadLabel:
					opNode = pathPairs.Value().Value.GoLow().Head.KeyNode
					op = pathPairs.Value().Head
				case v3.PatchLabel:
					opNode = pathPairs.Value().Value.GoLow().Patch.KeyNode
					op = pathPairs.Value().Patch
				case v3.TraceLabel:
					opNode = pathPairs.Value().Value.GoLow().Trace.KeyNode
					op = pathPairs.Value().Trace
				}

				if opValue.Security == nil && globalSecurity == nil {
					result := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							model.GetStringTemplates().BuildSecurityDefinedMessage(path, opType)),
						StartNode: opNode,
						EndNode:   vacuumUtils.BuildEndNode(opNode),
						Path:      op.GenerateJSONPath(),
						Rule:      context.Rule,
					}
					pathItem.AddRuleFunctionResult(drV3.ConvertRuleResult(&result))
					results = append(results, result)
					continue

				}

				if opValue.Security != nil && len(opValue.Security) <= 0 {
					result := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							model.GetStringTemplates().BuildSecurityEmptyMessage(path, opType)),
						StartNode: opNode,
						EndNode:   vacuumUtils.BuildEndNode(opNode),
						Path:      op.GenerateJSONPath(),
						Rule:      context.Rule,
					}
					opValue.AddRuleFunctionResult(drV3.ConvertRuleResult(&result))
					results = append(results, result)
				}

				if !nullable && len(opValue.Security) >= 1 {
					for i := range opValue.Security {
						if opValue.Security[i].Value.Requirements == nil || opValue.Security[i].Value.Requirements.Len() <= 0 {
							securityNode := opValue.Security[i].Value.GoLow().Requirements.ValueNode
							result := model.RuleFunctionResult{
								Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
									model.GetStringTemplates().BuildSecurityNullElementsMessage(path, opType)),
								StartNode: securityNode,
								EndNode:   vacuumUtils.BuildEndNode(securityNode),
								Path:      opValue.Security[i].GenerateJSONPath(),
								Rule:      context.Rule,
							}
							pathItem.AddRuleFunctionResult(drV3.ConvertRuleResult(&result))
							results = append(results, result)
							continue
						}
					}
				}

				if !nullable && opValue.Security == nil && len(globalSecurity) >= 1 {
					for i := range globalSecurity {
						if globalSecurity[i].Value.Requirements == nil || globalSecurity[i].Value.Requirements.Len() <= 0 {
							securityNode := globalSecurity[i].Value.GoLow().Requirements.ValueNode
							result := model.RuleFunctionResult{
								Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
									model.GetStringTemplates().BuildSecurityNullElementsMessage(path, opType)),
								StartNode: securityNode,
								EndNode:   vacuumUtils.BuildEndNode(securityNode),
								Path:      globalSecurity[i].GenerateJSONPath(),
								Rule:      context.Rule,
							}
							pathItem.AddRuleFunctionResult(drV3.ConvertRuleResult(&result))
							results = append(results, result)
							continue
						}
					}
				}
			}
		}
	}
	return results
}
