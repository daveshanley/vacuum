package owasp

import (
	"fmt"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
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

	//var responseCode = -1
	var results []model.RuleFunctionResult

	doc := context.Document
	if doc == nil {
		return results
	}

	if doc.GetSpecInfo().VersionNumeric <= 2 {
		return results
	}
	m, _ := doc.BuildV3Model()

	for pathPairs := m.Model.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
		for opPairs := pathPairs.Value().GetOperations().First(); opPairs != nil; opPairs = opPairs.Next() {
			opValue := opPairs.Value()
			responses := opValue.Responses.Codes
			var node *yaml.Node

			for respPairs := responses.First(); respPairs != nil; respPairs = respPairs.Next() {
				resp := respPairs.Value()
				respCode := respPairs.Key()
				code, _ := strconv.Atoi(respCode)

				if code >= 200 && code < 300 || code >= 400 && code < 500 {

					lowCodes := opValue.Responses.GoLow().Codes
					for lowCodePairs := lowCodes.First(); lowCodePairs != nil; lowCodePairs = lowCodePairs.Next() {
						lowCodeKey := lowCodePairs.Key()
						codeCodeVal, _ := strconv.Atoi(lowCodeKey.KeyNode.Value)
						if codeCodeVal == code {
							node = lowCodeKey.KeyNode
						}
					}
					if resp.Headers != nil {
						result := cd.getResult(code, node, resp, context, headers)
						results = append(results, result...)
					} else {

						results = append(results, model.RuleFunctionResult{
							Message:   message{responseCode: code, headersSets: headers}.String(),
							StartNode: node,
							EndNode:   node,
							Path:      fmt.Sprintf("$.paths.responses.%d", code),
							Rule:      context.Rule,
						})

					}

				}
			}
		}
	}
	return results
}

func (cd HeaderDefinition) getResult(responseCode int, codeNode *yaml.Node,
	response *v3.Response, context model.RuleFunctionContext, headersSets [][]string) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult
	numberOfHeaders := 0

	headers := response.GoLow().Headers
	var headerKeys []string

	for headerPairs := headers.Value.First(); headerPairs != nil; headerPairs = headerPairs.Next() {
		numberOfHeaders++
		headerKey := headerPairs.Key()
		headerKeys = append(headerKeys, headerKey.KeyNode.Value)
	}
	b := false
	for _, set := range headersSets {
		if belong(set, headerKeys) {
			b = true
		}
	}

	if !b {
		results = append(results, model.RuleFunctionResult{
			Message:   message{responseCode: responseCode, headersSets: headersSets}.String(),
			StartNode: headers.KeyNode,
			EndNode:   headers.KeyNode,
			Path:      fmt.Sprintf("$.paths.responses.%d.headers", responseCode),
			Rule:      context.Rule,
		})
	}
	return results
}

func belong(set []string, nodeHeaders []string) bool {
	for _, header := range set {
		if !slices.Contains(nodeHeaders, header) {
			return false
		}
	}
	return true
}
