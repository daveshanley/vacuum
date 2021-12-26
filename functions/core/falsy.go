package core

import (
	"fmt"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"gopkg.in/yaml.v3"
)

type Falsy struct {
}

func (f Falsy) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{}
}

func (f Falsy) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	for _, node := range nodes {

		fieldNode, fieldNodeValue := utils.FindKeyNode(context.RuleAction.Field, node.Content)
		if (fieldNode != nil && fieldNodeValue != nil) &&
			(fieldNodeValue.Value != "" && fieldNodeValue.Value != "false" || fieldNodeValue.Value != "0") {
			results = append(results, model.RuleFunctionResult{
				Message: fmt.Sprintf("'%s' must be falsy", context.RuleAction.Field),
			})
		}
	}

	return results
}
