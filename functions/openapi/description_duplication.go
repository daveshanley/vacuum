// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"crypto/md5"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
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
	return model.RuleFunctionSchema{Name: "description_duplication"}
}

// RunRule will execute the DescriptionDuplication rule, based on supplied context and a supplied []*yaml.Node slice.
func (dd DescriptionDuplication) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	seenDescriptions := make(map[string]*copyPasta)
	seenSummaries := make(map[string]*copyPasta)

	var results []model.RuleFunctionResult

	// extract all descriptions and summaries
	for _, root := range nodes {
		descriptions, _ := utils.FindNodesWithoutDeserializing(root, "$..description")
		summaries, _ := utils.FindNodesWithoutDeserializing(root, "$..summary")

		for _, description := range descriptions {

			data := []byte(description.Value)
			md5String := fmt.Sprintf("%x", md5.Sum(data))
			cp := copyPasta{
				value: description.Value,
				node:  description,
			}

			checkDescriptions(seenDescriptions, md5String, description, &results, cp)

		}

		// look through summaries
		for _, summary := range summaries {

			data := []byte(summary.Value)
			md5String := fmt.Sprintf("%x", md5.Sum(data))
			cp := copyPasta{
				value: summary.Value,
				node:  summary,
			}

			checkSummaries(seenSummaries, md5String, summary, &results, cp)
			if len(seenDescriptions) > 0 {
				checkDescriptions(seenDescriptions, md5String, summary, &results, cp)
			}

		}

	}
	return results

}

func checkSummaries(seenSummaries map[string]*copyPasta, md5String string, summary *yaml.Node,
	results *[]model.RuleFunctionResult, cp copyPasta) {
	if seenSummaries[md5String] != nil {
		// duplicate
		res := model.BuildFunctionResultString(fmt.Sprintf("Summary at line '%d' is a duplicate of line '%d'",
			summary.Line, seenSummaries[md5String].node.Line))
		res.StartNode = summary
		res.EndNode = summary
		res.Path = "$..summary"
		*results = append(*results, res)

	} else {
		seenSummaries[md5String] = &cp
	}
}

func checkDescriptions(seenDescriptions map[string]*copyPasta, md5String string, description *yaml.Node,
	results *[]model.RuleFunctionResult, cp copyPasta) {

	if seenDescriptions[md5String] != nil {
		// duplicate
		res := model.BuildFunctionResultString(fmt.Sprintf("Description at line '%d' is a duplicate of line '%d'",
			description.Line, seenDescriptions[md5String].node.Line))
		res.StartNode = description
		res.EndNode = description
		res.Path = "$..description"
		*results = append(*results, res)

	} else {
		seenDescriptions[md5String] = &cp
	}
}
