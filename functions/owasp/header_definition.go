package owasp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type message struct {
	responseCode int
	headersSets  [][]string
}

type HeaderDefinition struct {
}

func (m message) String() string {
	oout := ""
	for _, headerSet := range m.headersSets {
		oout += "{" + strings.Join(headerSet, ", ") + "} "
	}

	return fmt.Sprintf(`Response with code %d, must contain one of the defined 'headers': {%s}`, m.responseCode, oout)
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the HeaderDefinition rule.
func (cd HeaderDefinition) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "header_definition"}
}

// RunRule will execute the HeaderDefinition rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd HeaderDefinition) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) == 0 {
		return nil
	}

	var headers [][]string
	methodsMap := utils.ExtractValueFromInterfaceMap("headers", context.Options)
	if castedHeaders, ok := methodsMap.([][]string); ok {
		headers = castedHeaders
	}

	var responseCode = -1
	var results []model.RuleFunctionResult
	for i, node := range nodes[0].Content {
		if i%2 == 0 {
			responseCode, _ = strconv.Atoi(node.Value)
		} else if responseCode >= 200 && responseCode < 300 || responseCode >= 400 && responseCode < 500 {
			result := cd.getResult(responseCode, node, context, headers)
			results = append(results, result...)
			responseCode = 0
		}
	}

	return results
}

func (cd HeaderDefinition) getResult(responseCode int, node *yaml.Node, context model.RuleFunctionContext, headersSets [][]string) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	numberOfHeaders := 0

	for i, headersNode := range node.Content {
		if headersNode.Value == "headers" {
			numberOfHeaders++
			if !(len(node.Content) > i+1) || !cd.validateNode(node.Content[i+1], headersSets) {
				results = append(results, model.RuleFunctionResult{
					Message:   message{responseCode: responseCode, headersSets: headersSets}.String(),
					StartNode: headersNode,
					EndNode:   utils.FindLastChildNodeWithLevel(headersNode, 0),
					Path:      fmt.Sprintf("$.paths.responses.%d.headers", responseCode),
					Rule:      context.Rule,
				})
			}
		}
	}

	// headers parameter not found
	if numberOfHeaders == 0 {
		results = append(results, model.RuleFunctionResult{
			Message:   message{responseCode: responseCode, headersSets: headersSets}.String(),
			StartNode: node,
			EndNode:   utils.FindLastChildNodeWithLevel(node, 0),
			Path:      fmt.Sprintf("$.paths.responses.%d", responseCode),
			Rule:      context.Rule,
		})
	}

	return results
}

// RunRule will execute the HeaderDefinition rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd HeaderDefinition) validateNode(node *yaml.Node, headers [][]string) bool {
	var nodeHeaders []string
	for i, nodeHeader := range node.Content {
		if i%2 == 0 {
			nodeHeaders = append(nodeHeaders, nodeHeader.Value)
		}
	}

	for _, set := range headers {
		if belong(set, nodeHeaders) {
			return true
		}
	}

	return false
}

func belong(set []string, nodeHeaders []string) bool {
	for _, header := range set {
		if !slices.Contains(nodeHeaders, header) {
			return false
		}
	}

	return true
}
