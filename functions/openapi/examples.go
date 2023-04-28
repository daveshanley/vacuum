// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
    "fmt"
    "github.com/daveshanley/vacuum/model"
    "github.com/daveshanley/vacuum/model/reports"
    "github.com/daveshanley/vacuum/parser"
    validationErrors "github.com/pb33f/libopenapi-validator/errors"
    highBase "github.com/pb33f/libopenapi/datamodel/high/base"
    "github.com/pb33f/libopenapi/utils"
    "gopkg.in/yaml.v3"
    "strings"
    "sync"
    "time"
)

// Examples is a rule that checks that examples are being correctly used.
type Examples struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the TagDefined rule.
func (ex Examples) GetSchema() model.RuleFunctionSchema {
    return model.RuleFunctionSchema{
        Name: "examples",
    }
}

func checkExampleAsync(wg *sync.WaitGroup, rbNode *yaml.Node, basePath string, results *[]model.RuleFunctionResult, context model.RuleFunctionContext) {
    checkExamples(rbNode, basePath, results, context)
    wg.Done()
}

func analyzeExampleAsync(wg *sync.WaitGroup, nameNodeValue string, mediaTypeNode *yaml.Node, basePath string, results *[]model.RuleFunctionResult, context model.RuleFunctionContext) {
    analyzeExample(nameNodeValue, mediaTypeNode, basePath, results, context)
    wg.Done()
}

type opExample struct {
    path  string
    node  *yaml.Node
    param *yaml.Node
}

// RunRule will execute the Examples rule, based on supplied context and a supplied []*yaml.Node slice.
func (ex Examples) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

    if len(nodes) <= 0 {
        return nil
    }

    var results = &[]model.RuleFunctionResult{}

    ops := context.Index.GetPathsNode()

    var opPath, opMethod string

    // collect responses to scan
    var responseBodyCollection []opExample
    var requestBodyCollection []opExample
    var paramCollection []opExample

    if ops != nil {
        for i, op := range ops.Content {
            if i%2 == 0 {
                opPath = op.Value
                continue
            }

            for m, method := range op.Content {

                if m%2 == 0 {
                    opMethod = method.Value
                    continue
                }

                basePath := fmt.Sprintf("$.paths.%s.%s", opPath, opMethod)

                //check requests.
                _, rbNode := utils.FindKeyNodeTop("requestBody", method.Content)

                //check responses
                _, respNode := utils.FindKeyNodeTop("responses", method.Content)

                if rbNode != nil {
                    requestBodyCollection = append(requestBodyCollection, opExample{
                        path: utils.BuildPath(basePath, []string{"requestBody"}),
                        node: rbNode,
                    })
                }

                // check parameters.
                _, paramsNode := utils.FindKeyNodeTop("parameters", method.Content)
                if paramsNode != nil && utils.IsNodeArray(paramsNode) {

                    for y, param := range paramsNode.Content {

                        // extract name from param
                        _, nameNode := utils.FindKeyNodeTop("name", param.Content)
                        if nameNode != nil {

                            paramCollection = append(paramCollection, opExample{
                                path:  utils.BuildPath(basePath, []string{fmt.Sprintf("%s[%d]", "parameters", y)}),
                                node:  nameNode,
                                param: param,
                            })
                        }
                    }
                }

                if respNode != nil {
                    var code string
                    //wg.Add()
                    for x, respCodeNode := range respNode.Content {

                        if x%2 == 0 {
                            code = respCodeNode.Value
                            continue
                        }

                        responseBodyCollection = append(
                            responseBodyCollection,
                            opExample{
                                node: respCodeNode,
                                path: utils.BuildPath(basePath, []string{fmt.Sprintf("%s.%s", "responses", code)}),
                            })
                    }
                }

            }
        }
    }

    // scan requests, responses as fast as we can asynchronously.
    var wg sync.WaitGroup
    wg.Add(len(responseBodyCollection))
    wg.Add(len(requestBodyCollection))
    wg.Add(len(paramCollection))
    for _, rb := range responseBodyCollection {
        go checkExampleAsync(&wg, rb.node, rb.path, results, context)
    }
    for _, rb := range requestBodyCollection {
        go checkExampleAsync(&wg, rb.node, rb.path, results, context)
    }
    for _, p := range paramCollection {
        go analyzeExampleAsync(&wg, p.node.Value, p.param, p.path, results, context)
    }
    wg.Wait()

    //check components.
    objNode := context.Index.GetSchemasNode()

    if context.SpecInfo.SpecFormat == model.OAS3 {
        results = checkAllDefinitionsForExamples([]*yaml.Node{objNode}, results, "$.components.schemas", context)
    }
    if context.SpecInfo.SpecFormat == model.OAS2 {
        results = checkAllDefinitionsForExamples([]*yaml.Node{objNode}, results, "$.definitions", context)
    }

    // check parameters
    var componentParamPath string
    if context.SpecInfo.SpecFormat == model.OAS3 {
        componentParamPath = "$.components.parameters"
    }
    if context.SpecInfo.SpecFormat == model.OAS3 {
        componentParamPath = "$.parameters"
    }

    paramsNode := context.Index.GetParametersNode()

    if paramsNode != nil && (utils.IsNodeArray(paramsNode) || utils.IsNodeMap(paramsNode)) {

        for x, param := range paramsNode.Content {

            // extract name from param
            _, nameNode := utils.FindKeyNodeTop("name", param.Content)
            if nameNode != nil {
                results = analyzeExample(nameNode.Value, param,
                    utils.BuildPath(componentParamPath, []string{fmt.Sprintf("%s[%d]", "parameters", x)}), results, context)
            }
        }
    }

    return *results
}

func checkAllDefinitionsForExamples(objNode []*yaml.Node,
    results *[]model.RuleFunctionResult, path string, context model.RuleFunctionContext) *[]model.RuleFunctionResult {

    lineRefs := context.Index.GetLinesWithReferences()

    if len(objNode) > 0 {
        if objNode[0] != nil {
            compNode := objNode[0]
            var compName string
            for n, schemaNode := range compNode.Content {
                if n%2 == 0 {
                    compName = schemaNode.Value
                    continue
                }
                results = checkDefinitionForExample(schemaNode, compName, results, path, context, lineRefs)
            }
        }
    }
    return results
}

// super lean DFS to check if example is circular.
func miniCircCheck(node *yaml.Node, seen map[*yaml.Node]bool, depth int) bool {
    if depth > 40 {
        return false // too deep, this is insane.
    }
    if seen[node] {
        return true
    }
    seen[node] = true
    circ := false
    for _, child := range node.Content {
        depth++
        circ = miniCircCheck(child, seen, depth)
    }
    return circ

}

func checkDefinitionForExample(componentNode *yaml.Node, compName string,
    results *[]model.RuleFunctionResult, path string, context model.RuleFunctionContext, lineRefs map[int]bool) *[]model.RuleFunctionResult {

    // extract properties and a top level example, if it exists.
    topExKey, topExValue := utils.FindKeyNode("example", []*yaml.Node{componentNode})
    _, pValue := utils.FindKeyNode("properties", []*yaml.Node{componentNode})
    var pName string

    // if no object level example exists, check for property examples.
    if topExKey == nil && topExValue == nil && pValue != nil {
        for n, prop := range pValue.Content {
            if n%2 == 0 {
                pName = prop.Value
                continue
            }
            if utils.IsNodeMap(prop) {

                // check for an example
                exKey, exValue := utils.FindKeyNode("example", prop.Content)
                _, typeValue := utils.FindKeyNode("type", prop.Content)
                _, enumValue := utils.FindKeyNode("enum", prop.Content)

                skip := false
                if typeValue != nil {
                    switch typeValue.Value {
                    case "boolean":
                        skip = true
                    }
                }
                if enumValue != nil {
                    skip = true
                }

                if lineRefs[prop.Line] {
                    continue
                }

                if exKey == nil && exValue == nil && !skip {

                    res := model.BuildFunctionResultString(fmt.Sprintf("Missing example for `%s` on component `%s`",
                        pName, compName))

                    res.StartNode = pValue.Content[n-1]
                    if len(prop.Content) > 0 {
                        res.EndNode = prop.Content[len(prop.Content)-1]
                    } else {
                        res.EndNode = res.StartNode
                    }
                    res.Path = utils.BuildPath(path, []string{compName, pName})
                    res.Rule = context.Rule
                    res.Range = buildRange(res.StartNode, res.EndNode)
                    t := time.Now()
                    res.Timestamp = &t
                    *results = append(*results, res)
                    continue

                } else {

                    // no point going forward here.
                    if exKey == nil && exValue == nil {
                        continue
                    }

                    // so there is an example, lets validate it.
                    var schema *highBase.Schema

                    // if this node is somehow circular, we won't be able to convert it into a schema.
                    if !miniCircCheck(prop, make(map[*yaml.Node]bool), 0) {
                        schema, _ = parser.ConvertNodeIntoJSONSchema(prop, context.Index)
                    } else {
                        continue // no point moving on past here.
                    }

                    var res bool
                    var isArr bool
                    var errs []*validationErrors.ValidationError
                    for i := range schema.Type {
                        if schema.Type[i] == "array" {
                            isArr = true
                        }
                    }

                    if schema != nil && schema.Type != nil && isArr && exValue != nil {
                        res, errs = parser.ValidateNodeAgainstSchema(schema, exValue, true)
                    }
                    if schema != nil && schema.Type != nil && !isArr && exValue != nil {
                        res, errs = parser.ValidateNodeAgainstSchema(schema, exValue, false)
                    }

                    // TODO: handle enums in here.

                    if !res {

                        // extract all validation errors.
                        for _, resError := range errs {

                            var buf strings.Builder
                            for i := range resError.SchemaValidationErrors {
                                buf.WriteString(resError.SchemaValidationErrors[i].Reason)
                                if i+1 < len(resError.SchemaValidationErrors) {
                                    buf.WriteString("\n")
                                }
                            }

                            // TODO: Diagnose examples of arrays of enums.

                            z := model.BuildFunctionResultString(fmt.Sprintf("Example for property `%s` is not valid: `%s`",
                                pName, buf.String()))
                            z.StartNode = exKey
                            z.EndNode = exValue
                            z.Rule = context.Rule
                            z.Path = utils.BuildPath(path, []string{compName, pName})
                            z.Range = buildRange(z.StartNode, z.EndNode)
                            t := time.Now()
                            z.Timestamp = &t
                            *results = append(*results, z)
                        }
                    }
                }
                continue
            }
        }
    } else {

        // we have an object level example here, so let's convert our properties into a schema and validate
        // this example against the schema.
        // don't start chasing polymorphic nodes. Converting the schema could end up in an endless loop.
        if !utils.IsNodePolyMorphic(componentNode) {

            schema, _ := parser.ConvertNodeIntoJSONSchema(componentNode, context.Index)

            var errorResults []*validationErrors.ValidationError
            if topExValue != nil {
                _, errorResults = parser.ValidateNodeAgainstSchema(schema, topExValue, false)
            }

            // extract all validation errors.
            for _, resError := range errorResults {

                var buf strings.Builder
                for i := range resError.SchemaValidationErrors {
                    buf.WriteString(resError.SchemaValidationErrors[i].Reason)
                    if i+1 < len(resError.SchemaValidationErrors) {
                        buf.WriteString("\n")
                    }
                }

                z := model.BuildFunctionResultString(fmt.Sprintf("Example for property `%s` is not valid: `%s`",
                    compName, buf.String()))
                z.StartNode = topExKey

                if len(topExValue.Content) > 0 {
                    z.EndNode = topExValue.Content[len(topExValue.Content)-1]
                } else {
                    z.EndNode = topExKey
                }
                z.Rule = context.Rule
                z.Range = buildRange(z.StartNode, z.EndNode)
                t := time.Now()
                z.Timestamp = &t
                *results = append(*results, z)
            }
        }
    }

    return results
}

func checkExamples(rbNode *yaml.Node, basePath string, results *[]model.RuleFunctionResult, context model.RuleFunctionContext) *[]model.RuleFunctionResult {
    // don't bother if we can't see anything.
    if rbNode == nil {
        return results
    }
    for n, rbNodeChild := range rbNode.Content {
        if rbNodeChild.Value != "content" {
            continue
        } else {
            mediaTypeNode := rbNode.Content[n+1]
            var nameNodeValue string
            for b, nameValueNode := range mediaTypeNode.Content {
                if nameValueNode.Value == "" {
                    continue
                }
                if b+1 >= len(mediaTypeNode.Content) {
                    continue
                }
                nameNodeValue = nameValueNode.Value

                results = analyzeExample(nameNodeValue, mediaTypeNode.Content[b+1], basePath, results, context)
            }
        }
    }
    return results
}

func buildRange(start, end *yaml.Node) reports.Range {
    return reports.Range{
        Start: reports.RangeItem{
            Line: start.Line,
            Char: start.Column,
        },
        End: reports.RangeItem{
            Line: end.Line,
            Char: end.Column,
        },
    }
}

func analyzeExample(nameNodeValue string, mediaTypeNode *yaml.Node, basePath string, results *[]model.RuleFunctionResult, context model.RuleFunctionContext) *[]model.RuleFunctionResult {

    sLabel, sValue := utils.FindKeyNodeTop("schema", mediaTypeNode.Content)
    _, esValue := utils.FindKeyNodeTop("examples", mediaTypeNode.Content)
    _, eValue := utils.FindKeyNodeTop("example", mediaTypeNode.Content)

    // if there are no examples, anywhere then add a result.
    if sValue != nil && (esValue == nil && eValue == nil) {

        // check type is not a boolean
        _, typ := utils.FindKeyNodeTop("type", sValue.Content)
        if typ != nil && typ.Value != "boolean" && typ.Value != "number" {

            res := model.BuildFunctionResultString(fmt.Sprintf("Schema for `%s` does not "+
                "contain any examples or example data", nameNodeValue))

            res.StartNode = sLabel
            res.EndNode = sValue
            res.Path = basePath
            res.Rule = context.Rule
            res.Range = buildRange(sLabel, sValue)
            modifyExampleResults(results, &res)
        }
        return results
    }

    var schema *highBase.Schema

    // look through multiple examples and evaluate them.
    var exampleName string

    if esValue != nil {
        var exampleNameNode *yaml.Node
        for v, multiExampleNode := range esValue.Content {
            if v%2 == 0 {
                exampleName = multiExampleNode.Value
                exampleNameNode = multiExampleNode
                continue
            }

            nodePath := utils.BuildPath(basePath, []string{"content", nameNodeValue, "schema", "examples", exampleName, "value"})

            _, valueNode := utils.FindKeyNodeTop("value", multiExampleNode.Content)
            _, externalValueNode := utils.FindKeyNodeTop("externalValue", multiExampleNode.Content)

            if valueNode != nil {
                // check if the example validates against the convertedSchema
                // extract the convertedSchema
                convertedSchema, err := parser.ConvertNodeIntoJSONSchema(sValue, context.Index)

                if err != nil {
                    z := model.BuildFunctionResultString(fmt.Sprintf("Example `%s` is not valid: `%s`",
                        exampleName, err.Error()))
                    z.StartNode = exampleNameNode
                    z.EndNode = valueNode
                    z.Path = nodePath
                    z.Rule = context.Rule
                    z.Range = buildRange(exampleNameNode, exampleNameNode)
                    modifyExampleResults(results, &z)
                }

                if convertedSchema == nil {
                    z := model.BuildFunctionResultString(fmt.Sprintf("Example `%s` is not valid: `%s`",
                        exampleName, "no convertedSchema can be extracted, invalid convertedSchema"))
                    z.StartNode = exampleNameNode
                    z.EndNode = valueNode
                    z.Path = nodePath
                    z.Rule = context.Rule
                    z.Range = buildRange(exampleNameNode, exampleNameNode)
                    modifyExampleResults(results, &z)

                }

                res, errs := parser.ValidateNodeAgainstSchema(convertedSchema, valueNode, false)

                if !res {
                    // extract all validation errors.
                    for _, resError := range errs {

                        var buf strings.Builder
                        for i := range resError.SchemaValidationErrors {
                            buf.WriteString(resError.SchemaValidationErrors[i].Reason)
                            if i+1 < len(resError.SchemaValidationErrors) {
                                buf.WriteString("\n")
                            }
                        }

                        z := model.BuildFunctionResultString(fmt.Sprintf("Example `%s` is not valid: `%s`",
                            exampleName, buf.String()))
                        z.StartNode = exampleNameNode
                        z.EndNode = valueNode
                        z.Path = nodePath
                        z.Rule = context.Rule
                        z.Range = buildRange(exampleNameNode, exampleNameNode)
                        modifyExampleResults(results, &z)
                    }
                }

                // check if the example contains a summary
                _, summaryNode := utils.FindKeyNodeTop("summary", multiExampleNode.Content)
                if summaryNode == nil {
                    z := model.BuildFunctionResultString(fmt.Sprintf("Example `%s` (line %d) missing a `summary` "+
                        "- examples need explaining", exampleName, exampleNameNode.Line))
                    z.StartNode = exampleNameNode
                    z.EndNode = valueNode
                    z.Path = nodePath
                    z.Rule = context.Rule
                    z.Range = buildRange(exampleNameNode, exampleNameNode)
                    modifyExampleResults(results, &z)
                }

                // can`t both have a value and an external value set!
                if valueNode != nil && externalValueNode != nil {
                    z := model.BuildFunctionResultString(fmt.Sprintf("Example `%s` is not valid: cannot use"+
                        " both `value` and `externalValue` - choose one or the other",
                        exampleName))
                    z.StartNode = esValue
                    z.EndNode = valueNode
                    z.Path = nodePath
                    z.Rule = context.Rule
                    z.Range = buildRange(esValue, valueNode)
                    modifyExampleResults(results, &z)
                }

            }
        }
    }

    // handle single examples when a schema is used.
    if sValue != nil && eValue != nil {

        // ok, so let's check the object is valid against the schema.
        // extract the schema
        var err error
        if schema == nil {
            schema, err = parser.ConvertNodeIntoJSONSchema(sValue, context.Index)
            if err != nil {
                z := model.BuildFunctionResultString(fmt.Sprintf("Example for `%s` is not valid: `%s`",
                    nameNodeValue, err.Error()))
                z.StartNode = eValue
                if len(eValue.Content) > 0 {
                    z.EndNode = eValue.Content[len(eValue.Content)-1]
                } else {
                    z.EndNode = eValue
                }
                z.Rule = context.Rule
                z.Path = basePath
                z.Range = buildRange(eValue, eValue)
                modifyExampleResults(results, &z)
                return results
            }
        }

        //return results

        res, validateError := parser.ValidateNodeAgainstSchema(schema, eValue, false)

        var schemaErrors []*validationErrors.SchemaValidationFailure
        for i := range validateError {
            schemaErrors = append(schemaErrors, validateError[i].SchemaValidationErrors...)
        }

        if validateError != nil {

            var buf strings.Builder

            for i := range schemaErrors {
                buf.WriteString(schemaErrors[i].Reason)
                if i+1 < len(schemaErrors) {
                    buf.WriteString("\n")
                }
            }

            z := model.BuildFunctionResultString(fmt.Sprintf("Example for `%s` is not valid: `%s`",
                nameNodeValue, buf.String()))
            z.StartNode = eValue
            if len(eValue.Content) > 0 {
                z.EndNode = eValue.Content[len(eValue.Content)-1]
            } else {
                z.EndNode = eValue
            }
            z.Rule = context.Rule
            z.Path = basePath
            z.Range = buildRange(eValue, eValue)
            modifyExampleResults(results, &z)
            return results
        }

        if !res {
            // extract all validation errors.
            for _, resError := range schemaErrors {

                z := model.BuildFunctionResultString(fmt.Sprintf("Example for `%s` is not valid: `%s`",
                    nameNodeValue, resError.Reason))
                z.StartNode = eValue
                if len(eValue.Content) > 0 {
                    z.EndNode = eValue.Content[len(eValue.Content)-1]
                } else {
                    z.EndNode = eValue
                }
                z.Rule = context.Rule
                z.Path = basePath
                z.Range = buildRange(eValue, eValue)
                modifyExampleResults(results, &z)
            }
            return results
        }
    }

    ex := false
    if sValue != nil {

        _, propsNode := utils.FindKeyNodeTop("properties", []*yaml.Node{sValue})

        if propsNode != nil {
            for n, prop := range propsNode.Content {
                if n%2 != 0 {
                    _, exampleNode := utils.FindKeyNodeTop("example", []*yaml.Node{prop})
                    if exampleNode != nil {
                        ex = true
                    }
                }
            }
        }
    }
    if ex {
        if schema == nil && !utils.IsNodePolyMorphic(sValue) {
            var err error
            schema, err = parser.ConvertNodeIntoJSONSchema(sValue, context.Index)

            if err != nil {
                z := model.BuildFunctionResultString(err.Error())
                z.StartNode = sValue
                z.EndNode = sValue
                z.Rule = context.Rule
                z.Range = buildRange(sValue, sValue)
                modifyExampleResults(results, &z)
            }

        }
        if schema == nil {
            return results
        }
        exampleValidation := parser.ValidateExample(schema)
        if len(exampleValidation) > 0 {
            _, pNode := utils.FindKeyNodeTop("properties", sValue.Content)
            var endNode *yaml.Node
            if pNode != nil && len(pNode.Content) > 0 {
                endNode = pNode.Content[len(pNode.Content)-1]
            }
            for _, example := range exampleValidation {
                z := model.BuildFunctionResultString(example.Message)
                z.StartNode = pNode
                z.EndNode = endNode
                z.Rule = context.Rule
                z.Range = buildRange(pNode, endNode)
                modifyExampleResults(results, &z)
            }
        }
    }

    return results
}

var exampleLock sync.Mutex

func modifyExampleResults(results *[]model.RuleFunctionResult, result *model.RuleFunctionResult) {
    exampleLock.Lock()
    t := time.Now()
    result.Timestamp = &t
    *results = append(*results, *result)
    exampleLock.Unlock()
}
