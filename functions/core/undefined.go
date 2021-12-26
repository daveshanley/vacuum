package core

import (
	"fmt"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"gopkg.in/yaml.v3"
)

type Undefined struct {
}

func (u Undefined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{}
}

func (u Undefined) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	for _, node := range nodes {

		fieldNode, _ := utils.FindKeyNode(context.RuleAction.Field, node.Content)
		if fieldNode != nil {
			results = append(results, model.RuleFunctionResult{
				Message: fmt.Sprintf("'%s' must be undefined", context.RuleAction.Field),
			})
		}
	}

	return results
}
