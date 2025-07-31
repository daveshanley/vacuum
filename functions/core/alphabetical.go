// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
    "github.com/daveshanley/vacuum/model"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "github.com/pb33f/libopenapi/utils"
    "gopkg.in/yaml.v3"
    "sort"
    "strconv"
    "strings"
)

// Alphabetical is a rule that will check an array or object to determine if the values are in order.
// if the path is to an object, then the value function option 'keyedBy' must be used, to know how to sort the
// data.
type Alphabetical struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Alphabetical rule.
func (a Alphabetical) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "alphabetical",
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "keyedBy",
				Description: "this is the key of an object you want to use to sort objects. If not specified for maps, the map will be sorted by keys",
			},
		},
		ErrorMessage: "'alphabetical' function has invalid options supplied. To sort objects by property use 'keyedBy'" +
			" and decide which property on the array of objects you want to use. Maps without 'keyedBy' will be sorted by their keys.",
	}
}

// GetCategory returns the category of the Alphabetical rule.
func (a Alphabetical) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Alphabetical rule, based on supplied context and a supplied []*yaml.Node slice.
func (a Alphabetical) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	if len(nodes) <= 0 {
		return nil
	}

	var keyedBy string

	// check supplied type
	props := utils.ConvertInterfaceIntoStringMap(context.Options)
	if props["keyedBy"] != "" {
		keyedBy = props["keyedBy"]
	}

	for _, node := range nodes {
		pathValue := "unknown"
		if path, ok := context.Given.(string); ok {
			pathValue = path
		}

		if utils.IsNodeMap(node) {
			if keyedBy == "" {
				// Sort by map keys when keyedBy is not provided
				mapKeys := a.extractMapKeys(node)
				if len(mapKeys) > 0 && !sort.StringsAreSorted(mapKeys) {
					// Report one violation per unsorted map for deterministic behavior
					rs := a.reportMapKeyViolation(node, mapKeys, context)
					if rs != nil {
						results = append(results, *rs)
					}
				}
				continue
			}

			resultsFromKey := a.processMap(node, keyedBy, context)
			results = compareStringArray(node, resultsFromKey, context)
			results = model.MapPathAndNodesToResults(pathValue, node, node, results)
			continue
		}

		if utils.IsNodeArray(node) {
			if a.isValidArray(node) {
				if a.isValidStringArray(node) {
					rs := a.checkStringArrayIsSorted(node, context)
					results = append(results, rs...)
				}

				if a.isValidNumberArray(node) {
					rs := a.checkNumberArrayIsSorted(node, context)
					results = append(results, rs...)
				}

				if a.isValidMapArray(node) {
					resultsFromKey := a.processMap(node, keyedBy, context)
					results = compareStringArray(node, resultsFromKey, context)
				}
				results = model.MapPathAndNodesToResults(pathValue, node, node, results)

			}
			continue
		}

		// TODO: handle single value code

	}

	return results
}

func (a Alphabetical) processMap(node *yaml.Node, keyedBy string, _ model.RuleFunctionContext) []string {
	var resultsFromKey []string
	for x, v := range node.Content {
		// run odd numbers for values.
		if x == 0 || x%2 != 0 {
			if v.Tag == "!!map" {

				for y, kv := range v.Content {

					// check keys for keyedBy match
					if y%2 == 0 && keyedBy == kv.Value && y+1 < len(v.Content) {
						resultsFromKey = append(resultsFromKey, v.Content[y+1].Value)
					}
				}
			}
		}
	}
	return resultsFromKey
}

func (a Alphabetical) extractMapKeys(node *yaml.Node) []string {
	var keys []string
	// For maps, Content contains alternating keys and values (key1, value1, key2, value2, ...)
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Tag == "!!str" {
			keys = append(keys, node.Content[i].Value)
		}
	}
	return keys
}

func (a Alphabetical) reportMapKeyViolation(node *yaml.Node, mapKeys []string, context model.RuleFunctionContext) *model.RuleFunctionResult {
	// Find the first out-of-order pair to create a deterministic error message
	for i := 0; i < len(mapKeys)-1; i++ {
		if strings.Compare(mapKeys[i], mapKeys[i+1]) > 0 {
			locatedObjects, err := context.DrDocument.LocateModel(node)
			locatedPath := ""
			var allPaths []string
			if err == nil && locatedObjects != nil {
				for v, obj := range locatedObjects {
					if v == 0 {
						locatedPath = obj.GenerateJSONPath()
					}
					allPaths = append(allPaths, obj.GenerateJSONPath())
				}
			}

			result := model.RuleFunctionResult{
				Rule:      context.Rule,
				StartNode: node,
				Path:      locatedPath,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
					fmt.Sprintf("%s: `%s` must be placed before `%s` (alphabetical)",
						context.Rule.Description,
						mapKeys[i+1], mapKeys[i])),
			}
			if len(allPaths) > 1 {
				result.Paths = allPaths
			}
			if len(locatedObjects) > 0 {
				if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
					arr.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				}
			}
			return &result
		}
	}
	return nil
}

func (a Alphabetical) isValidArray(arr *yaml.Node) bool {
	for _, n := range arr.Content {
		switch n.Tag {
		case "!!bool":
			return false
		}
	}
	return true
}

func (a Alphabetical) isValidStringArray(arr *yaml.Node) bool {
	if len(arr.Content) == 0 {
		return false
	}
	return arr.Content[0].Tag == "!!str"
}

func (a Alphabetical) isValidNumberArray(arr *yaml.Node) bool {
	if len(arr.Content) == 0 {
		return false
	}
	return arr.Content[0].Tag == "!!int" || arr.Content[0].Tag == "!!float"
}

func (a Alphabetical) isValidMapArray(arr *yaml.Node) bool {
	if len(arr.Content) == 0 {
		return false
	}
	return arr.Content[0].Tag == "!!map"
}

func (a Alphabetical) checkStringArrayIsSorted(arr *yaml.Node,
	context model.RuleFunctionContext) []model.RuleFunctionResult {

	var strArr []string
	for _, n := range arr.Content {
		if n.Tag == "!!str" {
			strArr = append(strArr, n.Value)
		}
	}
	if sort.StringsAreSorted(strArr) {
		return nil
	}
	return compareStringArray(arr, strArr, context)
}

func compareStringArray(node *yaml.Node, strArr []string,
	context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult
	message := context.Rule.Message

	for x := 0; x < len(strArr); x++ {
		if x+1 < len(strArr) {
			s := strings.Compare(strArr[x], strArr[x+1])
			if s > 0 {

				locatedObjects, err := context.DrDocument.LocateModel(node)
				locatedPath := ""
				var allPaths []string
				if err == nil && locatedObjects != nil {
					for v, obj := range locatedObjects {
						if v == 0 {
							locatedPath = obj.GenerateJSONPath()
						}
						allPaths = append(allPaths, obj.GenerateJSONPath())
					}
				}

				result := model.RuleFunctionResult{
					Rule:      context.Rule,
					StartNode: node,
					Path:      locatedPath,
					EndNode:   vacuumUtils.BuildEndNode(node),
					Message: vacuumUtils.SuppliedOrDefault(message,
						model.GetStringTemplates().BuildAlphabeticalMessage(context.Rule.Description, strArr[x+1], strArr[x])),
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
	}
	return results
}

func (a Alphabetical) checkNumberArrayIsSorted(arr *yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	var intArray []int
	var floatArray []float64

	for _, n := range arr.Content {
		if n.Tag == "!!int" {
			intVal, _ := strconv.Atoi(n.Value)
			intArray = append(intArray, intVal)
		}
		if n.Tag == "!!float" {
			floatVal, _ := strconv.ParseFloat(n.Value, 64)
			floatArray = append(floatArray, floatVal)
		}
	}

	errmsg := "%s: `%v` is less than `%v`, they need to be swapped (numerical ordering)"

	if len(floatArray) > 0 {
		if !sort.Float64sAreSorted(floatArray) {
			results = a.evaluateFloatArray(arr, floatArray, errmsg, context)
		}
	}

	if len(intArray) > 0 {
		if !sort.IntsAreSorted(intArray) {
			results = append(results, a.evaluateIntArray(arr, intArray, errmsg, context)...)
		}
	}

	return results
}

func (a Alphabetical) evaluateIntArray(node *yaml.Node, intArray []int, errmsg string,
	context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult
	message := context.Rule.Message
	for x, n := range intArray {
		if x+1 < len(intArray) && n > intArray[x+1] {

			locatedObjects, err := context.DrDocument.LocateModel(node)
			locatedPath := ""
			var allPaths []string
			if err == nil && locatedObjects != nil {
				for x, obj := range locatedObjects {
					if x == 0 {
						locatedPath = obj.GenerateJSONPath()
					}
					allPaths = append(allPaths, obj.GenerateJSONPath())
				}
			}

			result := model.RuleFunctionResult{
				Rule:      context.Rule,
				StartNode: node,
				Path:      locatedPath,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Message: vacuumUtils.SuppliedOrDefault(message,
					model.GetStringTemplates().BuildNumericalOrderingMessage(context.Rule.Description, 
						strconv.Itoa(intArray[x+1]), strconv.Itoa(intArray[x]))),
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

func (a Alphabetical) evaluateFloatArray(node *yaml.Node, floatArray []float64, errmsg string,
	context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	message := context.Rule.Message

	for x, n := range floatArray {
		if x+1 < len(floatArray) && n > floatArray[x+1] {
			locatedObjects, err := context.DrDocument.LocateModel(node)
			locatedPath := ""
			var allPaths []string
			if err == nil && locatedObjects != nil {
				for x, obj := range locatedObjects {
					if x == 0 {
						locatedPath = obj.GenerateJSONPath()
					}
					allPaths = append(allPaths, obj.GenerateJSONPath())
				}
			}

			result := model.RuleFunctionResult{
				Rule:      context.Rule,
				StartNode: node,
				Path:      locatedPath,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Message: vacuumUtils.SuppliedOrDefault(message,
					model.GetStringTemplates().BuildNumericalOrderingMessage(context.Rule.Description,
						strconv.FormatFloat(floatArray[x+1], 'g', -1, 64), strconv.FormatFloat(floatArray[x], 'g', -1, 64))),
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
