package owasp

import (
	"fmt"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	drV3 "github.com/pb33f/doctor/model/high/v3"
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
	return fmt.Sprintf("response with code `%d`, must contain one of the defined headers: `%s`", m.responseCode, oout)
}

// GetCategory returns the category of the HeaderDefinition rule.
func (cd HeaderDefinition) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the HeaderDefinition rule.
func (cd HeaderDefinition) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspHeaderDefinition"}
}

// RunRule will execute the HeaderDefinition rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd HeaderDefinition) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	var headers []string
	methodsMap := utils.ExtractValueFromInterfaceMap("headers", context.Options)
	if castedHeaders, ok := methodsMap.([]interface{}); ok {
		for _, header := range castedHeaders {
			headers = append(headers, header.(string))
		}
	}
	if castedHeaders, ok := methodsMap.([]string); ok {
		headers = append(headers, castedHeaders...)

	}
	// compose header sets from header inputs
	var headerSets [][]string
	for _, header := range headers {
		headerSets = append(headerSets, strings.Split(header, "||"))
	}

	if context.DrDocument.V3Document != nil && context.DrDocument.V3Document.Paths != nil && context.DrDocument.V3Document.Paths.PathItems != nil {
		for pathPairs := context.DrDocument.V3Document.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
			for opPairs := pathPairs.Value().GetOperations().First(); opPairs != nil; opPairs = opPairs.Next() {
				opValue := opPairs.Value()

				if opValue.Responses != nil && opValue.Responses.Codes != nil {
					responses := opValue.Responses.Codes
					var node *yaml.Node

					for respPairs := responses.First(); respPairs != nil; respPairs = respPairs.Next() {
						resp := respPairs.Value()
						respCode := respPairs.Key()
						code, _ := strconv.Atoi(respCode)

						if code >= 200 && code < 300 || code >= 400 && code < 500 {

							lowCodes := opValue.Responses.Value.GoLow().Codes
							for lowCodePairs := lowCodes.First(); lowCodePairs != nil; lowCodePairs = lowCodePairs.Next() {
								lowCodeKey := lowCodePairs.Key()
								codeCodeVal, _ := strconv.Atoi(lowCodeKey.KeyNode.Value)
								if codeCodeVal == code {
									node = lowCodeKey.KeyNode
								}
							}
							if resp.Headers != nil {
								result := cd.getResult(code, resp, context, headerSets)
								results = append(results, result...)
							} else {

								results = append(results, model.RuleFunctionResult{
									Message:   message{responseCode: code, headersSets: headerSets}.String(),
									StartNode: node,
									EndNode:   vacuumUtils.BuildEndNode(node),
									Path:      vacuumUtils.SuppliedOrDefault(context.Rule.Message, resp.GenerateJSONPath()),
									Rule:      context.Rule,
								})
							}
						}
					}
				}
			}
		}
	}
	return results
}

func (cd HeaderDefinition) getResult(responseCode int,
	response *drV3.Response, context model.RuleFunctionContext, headersSets [][]string) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult
	numberOfHeaders := 0

	headers := response.Value.GoLow().Headers
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
			Path:      fmt.Sprintf("%s.headers", response.GenerateJSONPath()),
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
