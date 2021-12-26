package openapi_functions

import (
	"fmt"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

type PostResponseSuccess struct {
}

func (prs PostResponseSuccess) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{}
}

func (prs PostResponseSuccess) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	props := utils.ExtractValueFromInterfaceMap("properties", context.Options)
	values := utils.ConvertInterfaceArrayToStringArray(props)
	found := false

	for _, propVal := range values {
		key, _ := utils.FindFirstKeyNode(propVal, nodes)
		if key != nil {
			found = true
			break
		}
	}

	var results []model.RuleFunctionResult

	if !found {
		results = append(results, model.RuleFunctionResult{
			Message: fmt.Sprintf("operations must define a success response with one of the following codes: %s",
				strings.Join(values, ", ")),
		})
	}
	return results
}
