// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

func locateModelPaths(context model.RuleFunctionContext, node *yaml.Node, fallbackPath string) (string, []string, []v3.Foundational) {
	locatedPath := fallbackPath
	var allPaths []string
	var locatedObjects []v3.Foundational
	if context.DrDocument == nil {
		return locatedPath, allPaths, locatedObjects
	}
	located, err := context.DrDocument.LocateModel(node)
	if err != nil || located == nil {
		return locatedPath, allPaths, locatedObjects
	}
	locatedObjects = located
	for i, obj := range locatedObjects {
		if i == 0 {
			locatedPath = obj.GenerateJSONPath()
		}
		allPaths = append(allPaths, obj.GenerateJSONPath())
	}
	return locatedPath, allPaths, locatedObjects
}

func addResultToLocatedModel(locatedObjects []v3.Foundational, result *model.RuleFunctionResult) {
	if len(locatedObjects) == 0 {
		return
	}
	if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
		arr.AddRuleFunctionResult(v3.ConvertRuleResult(result))
	}
}
