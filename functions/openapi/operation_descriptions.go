package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

// OperationDescription will check if an operation has a description, and if the description is useful
type OperationDescription struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationDescription rule.
func (od OperationDescription) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "operation_description"}
}

// RunRule will execute the OperationDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (od OperationDescription) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// check supplied type
	props := utils.ConvertInterfaceIntoStringMap(context.Options)

	minWordsString := props["minWords"]
	minWords, _ := strconv.Atoi(minWordsString)

	// check operations first.
	ops := GetOperationsFromRoot(nodes)

	type copyPasta struct {
		value string
		node  *yaml.Node
	}

	//seen := make(map[string]copyPasta)
	// TODO: explode out copyPasta into new function for a new rule.

	var opPath, opMethod string
	for i, op := range ops {
		if i%2 == 0 {
			opPath = op.Value
			continue
		}

		for m, method := range op.Content {

			if m%2 == 0 {
				opMethod = method.Value
				continue
			}

			basePath := fmt.Sprintf("$.paths.%s.%s", opPath, opMethod)

			_, descNode := utils.FindKeyNode("description", method.Content)

			if descNode == nil {

				res := model.BuildFunctionResultString(fmt.Sprintf("Operation '%s' at path '%s' is missing a description",
					opMethod, opPath))

				res.StartNode = method
				res.EndNode = method
				res.Path = basePath
				results = append(results, res)

				continue
			}

			// check if description is above a certain length of words
			words := strings.Split(descNode.Value, " ")
			if len(words) < minWords {

				res := model.BuildFunctionResultString(fmt.Sprintf("Operation '%s' description at path '%s' must be "+
					"at least %d words long", opMethod, opPath, minWords))

				res.StartNode = descNode
				res.EndNode = descNode
				res.Path = basePath
				results = append(results, res)
				continue
			}

			// check if description is a copy paste

		}

	}

	return results

}
