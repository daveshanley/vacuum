package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"gopkg.in/yaml.v3"
)

type PathParameters struct {
	tagNodes []*yaml.Node
	opsNodes []*yaml.Node
}

func (pp PathParameters) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "path_parameters",
	}
}

func (pp PathParameters) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	return results

}
