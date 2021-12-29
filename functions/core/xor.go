package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

type Xor struct {
}

func (x Xor) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:     "xor",
		Required: []string{"properties"},
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "properties",
				Description: "'xor' requires two values",
			},
		},
		MinProperties: 2,
		MaxProperties: 2,
		ErrorMessage: "'xor' function has invalid options supplied. Example valid options are 'properties' = 'a, b'" +
			" or 'properties' = '1, 2'",
	}
}

func (x Xor) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	// check supplied properties, there can only be two
	props := utils.ConvertInterfaceIntoStringMap(context.Options)
	var properties []string

	if len(props) <= 0 {
		properties = utils.ConvertInterfaceToStringArray(context.Options)
	} else {
		properties = strings.Split(props["properties"], ",")
	}

	if len(properties) != 2 {
		return nil
	}

	var results []model.RuleFunctionResult
	seenCount := 0
	for _, node := range nodes {

		// look through our properties for a match (or no match), the end result needs to be exactly 1.
		for _, v := range properties {
			fieldNode, _ := utils.FindKeyNode(strings.TrimSpace(v), node.Content)

			if fieldNode != nil && fieldNode.Value == strings.TrimSpace(v) {
				seenCount++
			}
		}
	}

	if seenCount != 1 {
		results = append(results, model.RuleFunctionResult{
			Message: fmt.Sprintf("'%s' and '%s' must not be both defined or undefined",
				properties[0], properties[1]),
		})
	}

	return results
}
