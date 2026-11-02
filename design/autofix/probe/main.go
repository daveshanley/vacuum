package main

import (
	"encoding/json"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
)

func main() {
	for _, mode := range []string{"noop", "return-replacement", "insert-json-field"} {
		rule := &model.Rule{Id: "audit", Given: "$.info.description", Severity: "warn", AutoFixFunction: "fix", Then: &model.RuleAction{Function: "truthy"}}
		e := &motor.RuleSetExecution{Spec: []byte(`{"openapi":"3.0.0","info":{"title":"Test","version":"1.0.0","description":""},"paths":{}}`), SpecFileName: "test.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"audit": rule}}, ApplyAutoFixes: true, AutoFixFunctions: map[string]model.AutoFixFunction{"fix": func(n, doc *yaml.Node, _ *model.RuleFunctionContext) (*yaml.Node, error) {
			if mode == "return-replacement" {
				return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "A valid description"}, nil
			}
			if mode == "insert-json-field" {
				n.Value = "A valid description"
				root := doc.Content[0]
				root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x-audit"}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "value"})
			}
			return n, nil
		}}}
		r := motor.ApplyRulesToRuleSet(e)
		fmt.Printf("%s fixed=%d json=%t bytes=%s\n", mode, len(r.FixedResults), json.Valid(r.ModifiedSpec), r.ModifiedSpec)
		r.ReleaseOwnedResources()
	}
}
