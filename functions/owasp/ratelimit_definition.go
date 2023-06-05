package owasp

import (
	"fmt"
	"strconv"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type RateLimitDefinition struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the RateLimitDefinition rule.
func (cd RateLimitDefinition) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "ratelimit_definition"}
}

// RunRule will execute the RateLimitDefinition rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd RateLimitDefinition) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var responseCode = -1
	var results []model.RuleFunctionResult
	for i, node := range nodes[0].Content {
		if i%2 == 0 {
			responseCode, _ = strconv.Atoi(node.Value)
		} else if responseCode >= 200 && responseCode < 300 || responseCode >= 400 && responseCode < 500 {
			result := cd.getResult(responseCode, node, context)
			results = append(results, result...)
			responseCode = 0
		}
	}

	return results
}

func (cd RateLimitDefinition) getResult(responseCode int, node *yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	numberOfHeaders := 0

	for i, headersNode := range node.Content {
		if headersNode.Value == "headers" {
			numberOfHeaders++
			if !(len(node.Content) > i+1) || !cd.validateNode(node.Content[i+1]) {
				results = append(results, model.RuleFunctionResult{
					Message:   "Operation must define at least one 4xx error response", // TODO
					StartNode: headersNode,
					EndNode:   utils.FindLastChildNodeWithLevel(headersNode, 0),
					Path:      fmt.Sprintf("$.paths.responses.%d.headers", responseCode),
					Rule:      context.Rule,
				})
			}
		}
	}

	if numberOfHeaders == 0 {
		results = append(results, model.RuleFunctionResult{
			Message:   "Operation must define headers", // TODO
			StartNode: node,
			EndNode:   utils.FindLastChildNodeWithLevel(node, 0),
			Path:      fmt.Sprintf("$.paths.responses.%d", responseCode),
			Rule:      context.Rule,
		})
	}

	return results
}

// RunRule will execute the RateLimitDefinition rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd RateLimitDefinition) validateNode(node *yaml.Node) bool {
	var headers []string
	for i, headerNode := range node.Content {
		if i%2 == 0 {
			headers = append(headers, headerNode.Value)
		}
	}

	if slices.Contains(headers, "X-RateLimit-Limit") {
		return true
	}

	if slices.Contains(headers, "X-Rate-Limit-Limit") {
		return true
	}

	if slices.Contains(headers, "RateLimit-Limit") && slices.Contains(headers, "RateLimit-Reset") {
		return true
	}

	return false
}
