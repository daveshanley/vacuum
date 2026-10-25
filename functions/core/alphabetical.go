// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package core

import (
	"strconv"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/utils"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Alphabetical checks source order. Strings use English collation and numbers
// use numeric order. keyedBy selects a property from each object value.
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

// RunRule checks adjacent values and reports the first unordered pair per container.
func (a Alphabetical) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	keyedBy := context.GetOptionsStringMap()["keyedBy"]
	var order *collate.Collator
	greater := func(left, right *yaml.Node) bool {
		if isAlphabeticalNumber(left) && isAlphabeticalNumber(right) {
			// Keep integer comparisons exact before using the mixed numeric path.
			if left.Tag == "!!int" && right.Tag == "!!int" {
				l, le := strconv.ParseInt(left.Value, 10, 64)
				r, re := strconv.ParseInt(right.Value, 10, 64)
				if le == nil && re == nil {
					return l > r
				}
			}
			l, lok := alphabeticalNumber(left)
			r, rok := alphabeticalNumber(right)
			return lok && rok && l > r
		}
		if left.Tag != "!!str" || right.Tag != "!!str" {
			return false
		}
		if order == nil {
			// A collator owns mutable scratch buffers, so each invocation owns one.
			order = collate.New(language.English)
		}
		return order.CompareString(left.Value, right.Value) > 0
	}
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			node = node.Content[0]
		}
		values := alphabeticalValues(node, keyedBy)
		for i := 0; i+1 < len(values); i++ {
			if !greater(values[i], values[i+1]) {
				continue
			}
			path, paths, objects := locateNodePaths(&context, node)
			start := node
			if keyedBy == "" && node.Kind == yaml.SequenceNode {
				start = values[i]
				path = model.GetStringTemplates().BuildArrayPath(path, i)
				for p := range paths {
					paths[p] = model.GetStringTemplates().BuildArrayPath(paths[p], i)
				}
			}
			message := model.GetStringTemplates().BuildAlphabeticalMessage(context.Rule.Description, values[i+1].Value, values[i].Value)
			if isAlphabeticalNumber(values[i]) && isAlphabeticalNumber(values[i+1]) {
				message = model.GetStringTemplates().BuildNumericalOrderingMessage(context.Rule.Description, values[i+1].Value, values[i].Value)
			}
			result := model.RuleFunctionResult{
				Rule: context.Rule, Path: path, StartNode: start,
				EndNode: vacuumUtils.BuildEndNode(start),
				Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message, message),
			}
			if len(paths) > 1 {
				result.Paths = paths
			}
			addResultToLocatedModel(objects, &result)
			results = append(results, result)
			break
		}
	}
	return results
}

func isAlphabeticalNumber(node *yaml.Node) bool {
	return node.Tag == "!!int" || node.Tag == "!!float"
}

func alphabeticalNumber(node *yaml.Node) (float64, bool) {
	value, err := strconv.ParseFloat(node.Value, 64)
	if err == nil {
		return value, true
	}
	// YAML also permits numeric spellings such as hexadecimal and .inf.
	err = node.Decode(&value)
	return value, err == nil
}

// Return values in source order. Object values with keyedBy remain supported.
func alphabeticalValues(node *yaml.Node, keyedBy string) []*yaml.Node {
	if node.Kind != yaml.MappingNode && node.Kind != yaml.SequenceNode {
		return nil
	}
	if keyedBy == "" && node.Kind == yaml.SequenceNode {
		for _, value := range node.Content {
			if value.Tag != "!!str" && !isAlphabeticalNumber(value) {
				return nil
			}
		}
		return node.Content
	}
	step, first := 1, 0
	if node.Kind == yaml.MappingNode {
		step = 2
		if keyedBy != "" {
			first = 1
		}
	}
	values := make([]*yaml.Node, 0, len(node.Content)/step)
	for i := first; i < len(node.Content); i += step {
		value := node.Content[i]
		if keyedBy != "" {
			if value.Kind != yaml.MappingNode {
				return nil
			}
			_, value = utils.FindKeyNodeTop(keyedBy, value.Content)
		}
		if value == nil || value.Tag != "!!str" && !isAlphabeticalNumber(value) {
			return nil
		}
		values = append(values, value)
	}
	return values
}
