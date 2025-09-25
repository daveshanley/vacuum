// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
	"strconv"
	"strings"
)

// Enumeration is a rule that will check that a set of values meet the supplied 'values' supplied via functionOptions.
type Enumeration struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Enumeration rule.
func (e Enumeration) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:     "enumeration",
		Required: []string{"values"},
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "values",
				Description: "'enumeration' requires a set of values to operate against",
			},
		},
		MinProperties: 1,
		ErrorMessage:  "'enumerate' needs 'values' to operate. A valid example of 'values' are: 'cake, egg, milk'",
	}
}

// GetCategory returns the category of the Enumeration rule.
func (e Enumeration) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Enumeration rule, based on supplied context and a supplied []*yaml.Node slice.
func (e Enumeration) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) != 1 { // there can only be a single node passed in to this function.
		return nil
	}

	var results []model.RuleFunctionResult
	var values []string

	message := context.Rule.Message

	// check supplied values (required)
	m, ok := context.Options.(map[string]any)
	if !ok {
		return nil
	}
	optValues, ok := m["values"]
	if !ok {
		return nil
	}
	switch value := optValues.(type) {
	case string:
		values = strings.Split(value, ",")
	case []any:
		values = make([]string, 0, len(value))
		for i := range value {
			switch v := value[i].(type) {
			case string:
				values = append(values, v)
			case int:
				values = append(values, strconv.Itoa(v))
			case int64:
				values = append(values, strconv.FormatInt(v, 10))
			case float64:
				values = append(values, strconv.FormatFloat(v, 'f', -1, 64))
			case bool:
				values = append(values, strconv.FormatBool(v))
			default:
				// Fallback to fmt.Sprintf for complex types
				values = append(values, fmt.Sprintf("%v", v))
			}
		}
	}

	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	for _, node := range nodes {
		if !e.checkValueAgainstAllowedValues(node.Value, values) {

			locatedObjects, err := context.DrDocument.LocateModel(node)
			locatedPath := pathValue
			var allPaths []string
			if err == nil && locatedObjects != nil {
				for s, obj := range locatedObjects {
					if s == 0 {
						locatedPath = obj.GenerateJSONPath()
					}
					allPaths = append(allPaths, obj.GenerateJSONPath())
				}
			}

			result := model.RuleFunctionResult{
				Message: vacuumUtils.SuppliedOrDefault(message,
					model.GetStringTemplates().BuildEnumValidationMessage(ruleMessage, node.Value, values)),
				StartNode: node,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Path:      locatedPath,
				Rule:      context.Rule,
			}
			if len(allPaths) > 1 {
				result.Paths = allPaths
			}
			results = append(results, result)
			if len(locatedObjects) > 0 {
				if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
					arr.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				}
			}
		}
	}
	return results
}

func (e Enumeration) checkValueAgainstAllowedValues(value string, allowed []string) bool {
	found := false
	for _, allowedValue := range allowed {
		if strings.TrimSpace(allowedValue) == strings.TrimSpace(value) {
			found = true
			break
		}
	}
	return found
}
