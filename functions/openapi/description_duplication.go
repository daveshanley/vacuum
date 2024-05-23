// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"crypto/md5"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

type copyPasta struct {
	value string
	node  *yaml.Node
}

// DescriptionDuplication will check if a description has been duplicated (copy/paste)
type DescriptionDuplication struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DescriptionDuplication rule.
func (dd DescriptionDuplication) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "oasDescriptionDuplication"}
}

// GetCategory returns the category of the DescriptionDuplication rule.
func (dd DescriptionDuplication) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the DescriptionDuplication rule, based on supplied context and a supplied []*yaml.Node slice.
func (dd DescriptionDuplication) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	seenDescriptions := make(map[string]*copyPasta)
	seenSummaries := make(map[string]*copyPasta)

	// extract all descriptions and summaries
	descriptions := context.Index.GetAllDescriptions()
	summaries := context.Index.GetAllSummaries()

	for _, description := range descriptions {

		data := []byte(description.Node.Value)
		md5String := fmt.Sprintf("%x", md5.Sum(data))
		cp := copyPasta{
			value: description.Node.Value,
			node:  description.Node,
		}

		checkDescriptions(seenDescriptions, md5String, description.Node, &results, cp, description.Path, context)

	}

	// look through summaries
	for _, summary := range summaries {
		if summary.Node != nil {
			data := []byte(summary.Node.Value)
			md5String := fmt.Sprintf("%x", md5.Sum(data))
			cp := copyPasta{
				value: summary.Node.Value,
				node:  summary.Node,
			}

			checkSummaries(seenSummaries, md5String, summary.Node, &results, cp, summary.Path, context)
			if len(seenDescriptions) > 0 {
				checkDescriptions(seenDescriptions, md5String, summary.Node, &results, cp, summary.Path, context)
			}
		}
	}
	return results
}

func checkSummaries(seenSummaries map[string]*copyPasta, md5String string, summary *yaml.Node,
	results *[]model.RuleFunctionResult, cp copyPasta, path string, context model.RuleFunctionContext) {
	if seenSummaries[md5String] != nil {
		// duplicate
		res := model.BuildFunctionResultString(fmt.Sprintf("Summary at line `%d` is a duplicate of line `%d`",
			summary.Line, seenSummaries[md5String].node.Line))
		res.StartNode = summary
		res.EndNode = vacuumUtils.BuildEndNode(summary)
		res.Path = path
		res.Rule = context.Rule
		*results = append(*results, res)

	} else {
		seenSummaries[md5String] = &cp
	}
}

func checkDescriptions(seenDescriptions map[string]*copyPasta, md5String string, description *yaml.Node,
	results *[]model.RuleFunctionResult, cp copyPasta, path string, context model.RuleFunctionContext) {

	if seenDescriptions[md5String] != nil {
		// duplicate
		res := model.BuildFunctionResultString(fmt.Sprintf("Description at line `%d` is a duplicate of line `%d`",
			description.Line, seenDescriptions[md5String].node.Line))
		res.StartNode = description
		res.EndNode = vacuumUtils.BuildEndNode(description)
		res.Path = path
		res.Rule = context.Rule
		*results = append(*results, res)

	} else {
		seenDescriptions[md5String] = &cp
	}
}
