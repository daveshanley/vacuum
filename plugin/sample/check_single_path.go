package main

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"gopkg.in/yaml.v3"
)

// checkSinglePathExists is an example custom rule that checks only a single path exists.
type checkSinglePathExists struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Defined rule.
func (s checkSinglePathExists) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "checkSinglePathExists",
	}
}

// RunRule will execute the Sample rule, based on supplied context and a supplied []*yaml.Node slice.
func (s checkSinglePathExists) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	// get the index https://quobix.com/vacuum/api/spec-index/
	index := context.Index

	// get the paths node from the index.
	paths := index.GetPathsNode()

	// checks if there are more than two nodes present in the paths node, if so, more than one path is present.
	if len(paths.Content) > 2 {
		return []model.RuleFunctionResult{
			{
				Message:   fmt.Sprintf("more than a single path exists, there are %v", len(paths.Content)/2),
				StartNode: paths,
				EndNode:   paths,
				Path:      "$.paths",
				Rule:      context.Rule,
			},
		}
	}
	return nil
}
