package openapi_functions

import (
	"fmt"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"gopkg.in/yaml.v3"
)

type HelloFunction struct {
}

func (hf HelloFunction) RunRule(nodes []*yaml.Node, options interface{},
	context *model.RuleFunctionContext) []model.RuleFunctionResult {

	// get title node and return it.
	_, title := utils.FindKeyNode("title", nodes)

	return []model.RuleFunctionResult{{
		Message: fmt.Sprintf("oh hello '%s'", title.Value),
	}}
}
