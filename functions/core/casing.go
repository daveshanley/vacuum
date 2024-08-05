// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"regexp"
	"strconv"
)

const (
	flat        string = "flat"
	camel       string = "camel"
	pascal      string = "pascal"
	pascalKebab string = "pascal-kebab"
	kebab       string = "kebab"
	cobol       string = "cobol"
	snake       string = "snake"
	macro       string = "macro"
)

// Casing is a rule that will check the value of a node to ensure it meets the required casing type.
type Casing struct {
	flat                  string
	camel                 string
	pascal                string
	pascalKebab           string
	kebab                 string
	cobol                 string
	snake                 string
	macro                 string
	separatorPattern      string
	ignoreDigits          bool
	separatorChar         string
	separatorAllowLeading bool
	compiled              bool
}

var casingTypes = []string{flat, camel, pascal, kebab, cobol, snake, macro}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Casing rule.
func (c Casing) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:          "casing",
		Required:      []string{"type"},
		MinProperties: 1,
		Properties: []model.RuleFunctionProperty{
			{
				Name: "type",
				Description: fmt.Sprintf("'casing' requires a 'type' to be supplied, which can be one of:"+
					" '%s'", casingTypes),
			},
			{
				Name:        "disallowDigits",
				Description: "don't allow digits in any matched pattern",
			},
			{
				Name:        "separator.char",
				Description: "use a separator character",
			},
			{
				Name:        "separator.allowLeading",
				Description: "Allow a leading separator or not",
			},
		},
		ErrorMessage: "'casing' function has invalid options supplied. Example valid options are 'type' = 'camel'" +
			" or 'disallowDigits' = true",
	}
}

// GetCategory returns the category of the Casing rule.
func (c Casing) GetCategory() string {
	return model.FunctionCategoryCore

}

// RunRule will execute the Casing rule, based on supplied context and a supplied []*yaml.Node slice.
func (c Casing) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	// We expect at least one node to be found from the match
	if len(nodes) == 0 {
		return nil
	}

	// If we matched more than one node (eg through a recursive JSONPATH search such as '$..properties')
	// Then recursively apply the casing to all nodes and bubble up all the results
	if len(nodes) > 1 {
		for _, n := range nodes {
			results = append(results, c.RunRule([]*yaml.Node{n}, context)...)
		}
		return results
	}

	// From here on out, we are processing a single node
	node := nodes[0]

	var casingType string

	message := context.Rule.Message

	// check supplied type
	props := utils.ConvertInterfaceIntoStringMap(context.Options)
	if props["type"] == "" {
		return nil
	}
	casingType = props["type"]

	// pull out props
	if props["disallowDigits"] != "" {
		c.ignoreDigits, _ = strconv.ParseBool(props["disallowDigits"])
	}

	if props["separator.char"] != "" {
		c.separatorChar = props["separator.char"]
	}

	if props["separator.allowLeading"] != "" {
		c.separatorAllowLeading, _ = strconv.ParseBool(props["separator.allowLeading"])
	}

	// if a separator is defined, and can be used as a leading char, and the node value is that
	// char (rune, what ever), then we're done.
	if len(node.Value) == 1 &&
		c.separatorChar != "" &&
		c.separatorAllowLeading &&
		c.separatorChar == node.Value {
		return nil
	}

	var pattern string

	if !c.compiled {
		c.compileExpressions()
	}

	switch casingType {
	case camel:
		pattern = c.camel
	case pascal:
		pattern = c.pascal
	case pascalKebab:
		pattern = c.pascalKebab
	case kebab:
		pattern = c.kebab
	case cobol:
		pattern = c.cobol
	case snake:
		pattern = c.snake
	case macro:
		pattern = c.macro
	case flat:
		pattern = c.flat
	}

	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	// If the matched node is an array or map, then we should apply the casing rule to all it's children nodes:
	// For maps, it applies to all field names and for arrays, it applies to all elements.
	nodesToMatch := c.unravelNode(node)

	var rx *regexp.Regexp
	if c.separatorChar == "" {
		rx = regexp.MustCompile(fmt.Sprintf("^%s$", pattern))

	} else {
		c.separatorPattern = fmt.Sprintf("[%s]", regexp.QuoteMeta(c.separatorChar))
		var leadingSepPattern string
		var leadingPattern string
		leadingSepPattern = c.separatorPattern
		if c.separatorAllowLeading {
			leadingPattern = fmt.Sprintf("^(?:%[1]s)?%[3]s(?:%[2]s%[3]s)*$", leadingSepPattern, c.separatorPattern, pattern)
		} else {
			leadingPattern = fmt.Sprintf("^(?:%[1]s)+(?:%[2]s%[1]s)*$", pattern, c.separatorPattern)
		}

		rx = regexp.MustCompile(leadingPattern)
	}

	// Go through each node and check if the casing is correct
	for _, n := range nodesToMatch {
		if !rx.MatchString(n.Value) {
			locatedObject, err := context.DrDocument.LocateModel(node)
			locatedPath := pathValue
			if err == nil && locatedObject != nil {
				locatedPath = locatedObject.GenerateJSONPath()
			}
			results = append(results, model.RuleFunctionResult{
				Message:   vacuumUtils.SuppliedOrDefault(message, fmt.Sprintf("%s: `%s` is not `%s` case", ruleMessage, n.Value, casingType)),
				StartNode: n,
				EndNode:   vacuumUtils.BuildEndNode(n),
				Path:      locatedPath,
				Rule:      context.Rule,
			})
		}
	}

	return results
}

// If a node refers to an object, return a list of it's fields. If a node refers to an array, return a list of it's elements
func (c Casing) unravelNode(node *yaml.Node) []*yaml.Node {
	var nodesToMatch []*yaml.Node

	if utils.IsNodeMap(node) {
		for ii := 0; ii < len(node.Content); ii += 2 {
			nodesToMatch = append(nodesToMatch, node.Content[ii])
		}
	} else if utils.IsNodeArray(node) {
		nodesToMatch = node.Content
	} else {
		nodesToMatch = append(nodesToMatch, node)
	}

	return nodesToMatch
}

func (c *Casing) compileExpressions() {

	digits := "0-9"
	if c.ignoreDigits {
		digits = ""
	}

	c.flat = fmt.Sprintf("[a-z][a-z%[1]s]*", digits)
	c.camel = fmt.Sprintf("[a-z][a-z%[1]s]*(?:[A-Z%[1]s](?:[a-z%[1]s]+|$))*", digits)
	c.pascal = fmt.Sprintf("[A-Z][a-z%[1]s]*(?:[A-Z%[1]s](?:[a-z%[1]s]+|$))*", digits)
	c.pascalKebab = fmt.Sprintf("[A-Z][a-z%[1]s]*(-[A-Z][a-z%[1]s]*)*", digits)
	c.kebab = fmt.Sprintf("[a-z%[1]s-]+", digits)
	c.cobol = fmt.Sprintf("[A-Z%[1]s-]+", digits)
	c.snake = fmt.Sprintf("[a-z%[1]s_]+", digits)
	c.macro = fmt.Sprintf("[A-Z%[1]s_]+", digits)
	c.compiled = true
}
