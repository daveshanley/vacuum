package core

import (
	"fmt"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

type Enumeration struct{}

func (e Enumeration) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) != 1 { // there can only be a single node passed in to this function.
		return nil
	}

	var results []model.RuleFunctionResult
	var values []string

	// check supplied values (required)
	props := utils.ConvertInterfaceIntoStringMap(context.Options)
	if props["values"] == "" {
		return nil
	} else {
		values = strings.Split(props["values"], ",")
	}

	for _, node := range nodes {
		if !e.checkValueAgainstAllowedValues(node.Value, values) {
			results = append(results, model.RuleFunctionResult{
				Message: fmt.Sprintf("'%s' must equal to one of the following: %v", node.Value, values),
			})
		}
	}
	return results
}

func (e Enumeration) checkValueAgainstAllowedValues(value string, allowed []string) bool {
	found := false
	for _, allowedValue := range allowed {
		if strings.TrimSpace(allowedValue) == strings.TrimSpace(value) {
			found = true
			break
		}
	}
	return found
}
