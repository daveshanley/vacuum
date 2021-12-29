package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

type Defined struct {
}

func (d Defined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:          "defined",
		RequiresField: true,
	}
}

func (d Defined) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	for _, node := range nodes {

		fieldNode, _ := utils.FindKeyNode(context.RuleAction.Field, node.Content)
		if fieldNode == nil {
			results = append(results, model.RuleFunctionResult{
				Message: fmt.Sprintf("'%s' must be defined", context.RuleAction.Field),
			})
		}
	}

	return results
}
