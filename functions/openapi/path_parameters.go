// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"gopkg.in/yaml.v3"
)

// PathParameters is a rule that checks path level and operation level parameters for correct paths. The rule is
// one of the more complex, so here is a little detail as to what is happening.
//-- normalize paths to replace vars with % and duplicate check.
//-- check for duplicate param names in paths
//-- check for any unknown params (no name)
//-- check if required is set, that it's set to true only.
//-- check no duplicate params
//-- operation paths only
//-- all params in path must be defined
//-- all defined path params must be in path.
type PathParameters struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the PathParameters rule.
func (pp PathParameters) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "path_parameters",
	}
}

// RunRule will execute the PathParameters rule, based on supplied context and a supplied []*yaml.Node slice.
func (pp PathParameters) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	return results

}
