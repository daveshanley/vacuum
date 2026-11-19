// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package core

import (
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/utils"
)

// Alphabetical checks source order. Strings use lexical order and numbers
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

// RunRule reports unordered adjacent values. Plain maps report their first unordered pair.
func (a Alphabetical) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	keyedBy, keyed := context.GetOptionsStringMap()["keyedBy"]
	if context.Contract != model.OpenLintV1Draft {
		keyed = keyedBy != ""
	}
	var ordering *collate.Collator
	if context.Contract == model.OpenLintV1Draft {
		ordering = collate.New(language.English)
	}
	greater := func(left, right *yaml.Node) bool {
		if left == nil || right == nil {
			return false
		}
		if keyedBy == "" && isAlphabeticalNumber(left) && isAlphabeticalNumber(right) {
			return alphabeticalNumbersGreater(left, right)
		}
		if keyedBy == "" && (left.Tag != "!!str" || right.Tag != "!!str") {
			return false
		}
		return left.Value > right.Value
	}
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			node = node.Content[0]
		}
		values := alphabeticalValues(node, keyedBy, keyed)
		for i := 0; i+1 < len(values); i++ {
			if context.Contract != model.OpenLintV1Draft && !alphabeticalComparable(values[i], values[i], keyedBy != "") {
				continue
			}
			j := i + 1
			for context.Contract != model.OpenLintV1Draft && j < len(values) && !alphabeticalComparable(values[i], values[j], keyedBy != "") {
				j++
			}
			if j == len(values) {
				continue
			}
			outOfOrder := false
			if context.Contract == model.OpenLintV1Draft {
				// keyedBy addresses array elements, not the keys of a mapping.
				outOfOrder = node.Kind == yaml.MappingNode && keyed ||
					openLintAlphabeticalGreater(values[i], values[j], ordering, node.Kind == yaml.MappingNode && !keyed)
			} else {
				outOfOrder = greater(values[i], values[j])
			}
			if !outOfOrder {
				continue
			}
			path, paths, objects := locateNodePaths(&context, node)
			start := node
			if !keyed && node.Kind == yaml.SequenceNode {
				start = values[i]
				path = model.GetStringTemplates().BuildArrayPath(path, i)
				for p := range paths {
					paths[p] = model.GetStringTemplates().BuildArrayPath(paths[p], i)
				}
			}
			message := "values must be numbers or strings in ascending order"
			if values[i] != nil && values[j] != nil && (context.Contract != model.OpenLintV1Draft || alphabeticalComparable(values[i], values[i], false) && alphabeticalComparable(values[j], values[j], false)) {
				message = model.GetStringTemplates().BuildAlphabeticalMessage(context.Rule.Description, values[j].Value, values[i].Value)
				if keyedBy == "" && isAlphabeticalNumber(values[i]) && isAlphabeticalNumber(values[j]) {
					message = model.GetStringTemplates().BuildNumericalOrderingMessage(context.Rule.Description, values[j].Value, values[i].Value)
				}
			} else if keyed {
				message = "each element must be an object with property " + strconv.Quote(keyedBy)
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
			if context.Contract == model.OpenLintV1Draft || node.Kind == yaml.MappingNode && keyedBy == "" {
				break
			}
		}
	}
	return results
}

func isAlphabeticalNumber(node *yaml.Node) bool {
	return node.Tag == "!!int" || node.Tag == "!!float"
}

func alphabeticalNumbersGreater(left, right *yaml.Node) bool {
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
func alphabeticalValues(node *yaml.Node, keyedBy string, keyed bool) []*yaml.Node {
	if node.Kind != yaml.MappingNode && node.Kind != yaml.SequenceNode {
		return nil
	}
	if !keyed && node.Kind == yaml.SequenceNode {
		return node.Content
	}
	step, first := 1, 0
	if node.Kind == yaml.MappingNode {
		step = 2
		if keyed {
			first = 1
		}
	}
	values := make([]*yaml.Node, 0, len(node.Content)/step)
	for i := first; i < len(node.Content); i += step {
		value := node.Content[i]
		if keyed {
			if value.Kind == yaml.MappingNode {
				_, value = utils.FindKeyNodeTop(keyedBy, value.Content)
			} else {
				value = nil
			}
		}
		// Preserve source indices while comparisons skip missing values.
		values = append(values, value)
	}
	return values
}

func alphabeticalComparable(left, right *yaml.Node, keyed bool) bool {
	if left == nil || right == nil {
		return false
	}
	if keyed {
		return left.Kind == yaml.ScalarNode && right.Kind == yaml.ScalarNode
	}
	return left.Tag == "!!str" && right.Tag == "!!str" || isAlphabeticalNumber(left) && isAlphabeticalNumber(right)
}

// OpenLint compares number/string pairs asymmetrically, following its comparison
// table. Native mode retains its own handling of unsupported and missing values.
func openLintAlphabeticalGreater(left, right *yaml.Node, ordering *collate.Collator, keys bool) bool {
	if left == nil || right == nil {
		return true
	}
	if !keys && (!alphabeticalComparable(left, left, false) || !alphabeticalComparable(right, right, false)) {
		return true
	}
	if keys || left.Tag == "!!str" && right.Tag == "!!str" {
		_, leftNumeric := openLintNumericString(left.Value)
		_, rightNumeric := openLintNumericString(right.Value)
		return (leftNumeric || !rightNumeric) && ordering.CompareString(left.Value, right.Value) > 0
	}
	if !isAlphabeticalNumber(left) {
		return false
	}
	if right.Tag != "!!str" {
		return alphabeticalNumbersGreater(left, right)
	}
	l, lok := alphabeticalNumber(left)
	r, rok := openLintNumericString(right.Value)
	return lok && rok && l > r
}

var openLintDecimalString = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)

// openLintNumericString accepts ECMAScript StringNumericLiteral, not Go or YAML
// numeric syntax. In particular, separators and signed radix literals are invalid.
func openLintNumericString(value string) (float64, bool) {
	value = strings.TrimFunc(value, func(r rune) bool {
		return r == '\t' || r == '\n' || r == '\v' || r == '\f' || r == '\r' ||
			r == '\u2028' || r == '\u2029' || r == '\ufeff' || unicode.Is(unicode.Zs, r)
	})
	switch value {
	case "":
		return 0, true
	case "Infinity", "+Infinity":
		return math.Inf(1), true
	case "-Infinity":
		return math.Inf(-1), true
	}
	if len(value) > 2 && value[0] == '0' {
		base := 0
		switch value[1] {
		case 'x', 'X':
			base = 16
		case 'o', 'O':
			base = 8
		case 'b', 'B':
			base = 2
		}
		if base != 0 {
			integer, ok := new(big.Int).SetString(value[2:], base)
			if !ok || value[2] == '+' || value[2] == '-' {
				return 0, false
			}
			number, _ := integer.Float64()
			return number, true
		}
	}
	if !openLintDecimalString.MatchString(value) {
		return 0, false
	}
	number, err := strconv.ParseFloat(value, 64)
	// Overflow is a valid numeric string whose value is Infinity.
	return number, err == nil || math.IsInf(number, 0)
}
