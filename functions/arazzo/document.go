// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

// Package arazzo contains Arazzo rule functions backed by libopenapi-validator.
package arazzo

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
)

// Document exposes one group of findings from the execution's shared validation.
type Document struct{}

// GetSchema declares the validator diagnostic code selected by this rule.
func (Document) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "arazzoDocument", Required: []string{"code"}, Properties: []model.RuleFunctionProperty{{Name: "code"}}}
}

// GetCategory returns the Arazzo rule function category.
func (Document) GetCategory() string { return model.FunctionCategoryArazzo }

// RunRule attaches the configured rule to its precomputed validator findings.
func (Document) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.Arazzo == nil {
		return nil
	}
	findings := context.Arazzo.Results(context.GetOptionsStringMap()["code"])
	results := make([]model.RuleFunctionResult, len(findings))
	copy(results, findings)
	for i := range results {
		results[i].Rule = context.Rule
		results[i].RuleId = context.Rule.Id
	}
	return results
}
