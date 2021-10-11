package openapi_functions

import (
	"github.com/daveshanley/vaccum/model"
	"gopkg.in/yaml.v3"
)

type HelloFunction struct {
}

func (hf HelloFunction) RunRule(nodes []*yaml.Node, options interface{},
	context *model.RuleFunctionContext) []model.RuleFunctionResult {

	return []model.RuleFunctionResult{{
		Message: "oh hello",
	}}
}
