// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package motor

import (
	"net/url"
	"sync"

	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/mitchellh/mapstructure"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v2 "github.com/pb33f/libopenapi/datamodel/high/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type ruleContext struct {
	rule             *model.Rule
	specNode         *yaml.Node
	builtinFunctions functions.Functions
	ruleResults      *[]model.RuleFunctionResult
	wg               *sync.WaitGroup
	errors           *[]error
	index            *index.SpecIndex
	specInfo         *datamodel.SpecInfo
	customFunctions  map[string]model.RuleFunction
	panicFunc        func(p any)
	silenceLogs      bool
	document         libopenapi.Document
}

// RuleSetExecution is an instruction set for executing a ruleset. It's a convenience structure to allow the signature
// of ApplyRulesToRuleSet to change, without a huge refactor. The ApplyRulesToRuleSet function only returns a single error also.
type RuleSetExecution struct {
	RuleSet         *rulesets.RuleSet             // The RuleSet in which to apply
	Spec            []byte                        // The raw bytes of the OpenAPI specification.
	SpecInfo        *datamodel.SpecInfo           // Pre-parsed spec-info.
	CustomFunctions map[string]model.RuleFunction // custom functions loaded from plugin.
	PanicFunction   func(p any)                   // In case of emergency, do this thing here.
	SilenceLogs     bool                          // Prevent any warnings about rules/rule-sets being printed.
	Base            string                        // The base path or URL of the specification, used for resolving relative or remote paths.
	Document        libopenapi.Document           // a ready to render model.
}

// RuleSetExecutionResult returns the results of running the ruleset against the supplied spec.
type RuleSetExecutionResult struct {
	RuleSetExecution *RuleSetExecution          // The execution struct that was used invoking the result.
	Results          []model.RuleFunctionResult // The results of the execution.
	Index            *index.SpecIndex           // The index that was created from the specification, used by the rules.
	SpecInfo         *datamodel.SpecInfo        // A reference to the SpecInfo object, used by all the rules.
	Errors           []error                    // Any errors that were returned.

}

// todo: move copy into virtual file system or some kind of map.
const CircularReferencesFix string = "Circular references are created by schemas that reference back to themselves somewhere " +
	"in the chain. The link could be very deep, or it could be super shallow. Sometimes it's hard to know what is looping " +
	"without resolving the references. This model is looping, Remove the looping link in the chain. This can also appear with missing or " +
	"references that cannot be located or resolved correctly."

// ApplyRulesToRuleSet is a replacement for ApplyRules. This function was created before trying to use
// vacuum as an API. The signature is not sufficient, but is embedded everywhere. This new method
// uses a message structure, to allow the signature to grow, without breaking anything.
func ApplyRulesToRuleSet(execution *RuleSetExecution) *RuleSetExecutionResult {

	builtinFunctions := functions.MapBuiltinFunctions()
	var ruleResults []model.RuleFunctionResult
	var ruleWaitGroup sync.WaitGroup
	if execution.RuleSet != nil && execution.RuleSet.Rules != nil {
		ruleWaitGroup.Add(len(execution.RuleSet.Rules))
	}

	var specResolved *yaml.Node
	var specUnresolved *yaml.Node

	// create new configurations
	config := index.CreateClosedAPIIndexConfig()

	docConfig := datamodel.NewClosedDocumentConfiguration()
	docConfig.AllowFileReferences = true
	config.AllowFileLookup = true

	if execution.Base != "" {
		// check if this is a URL or not
		u, e := url.Parse(execution.Base)
		if e == nil && u.Scheme != "" && u.Host != "" {
			config.BaseURL = u
			config.BasePath = ""
			docConfig.BaseURL = u
			docConfig.BasePath = ""
		} else {
			config.BasePath = execution.Base
			docConfig.BasePath = execution.Base
		}
		config.AllowRemoteLookup = true
	}

	var specInfo, specInfoUnresolved *datamodel.SpecInfo
	var doc libopenapi.Document
	var err error

	// create a new document.
	doc, err = libopenapi.NewDocumentWithConfiguration(execution.Spec, docConfig)

	if err != nil {
		// Done.
		return &RuleSetExecutionResult{Errors: []error{err}}
	}

	// build model
	var docModelErrors []error
	var modelIndex *index.SpecIndex

	version := doc.GetVersion()
	switch version[0] {
	case '2':
		var docModel *libopenapi.DocumentModel[v2.Swagger]
		docModel, docModelErrors = doc.BuildV2Model()
		if execution.SpecInfo == nil {
			specInfo = doc.GetSpecInfo()
			specInfoUnresolved, _ = datamodel.ExtractSpecInfo(execution.Spec)
		} else {
			specInfo = execution.SpecInfo
			specInfoUnresolved = execution.SpecInfo
		}
		if docModel != nil {
			modelIndex = docModel.Index
		}
	case '3':
		var docModel *libopenapi.DocumentModel[v3.Document]
		docModel, docModelErrors = doc.BuildV3Model()
		if execution.SpecInfo == nil {
			specInfo = doc.GetSpecInfo()
			specInfoUnresolved, _ = datamodel.ExtractSpecInfo(execution.Spec)
		} else {
			specInfo = execution.SpecInfo
			specInfoUnresolved = execution.SpecInfo
		}
		if docModel != nil {
			modelIndex = docModel.Index
		}
	}

	specUnresolved = specInfoUnresolved.RootNode
	specResolved = specInfo.RootNode

	var indexResolved, indexUnresolved *index.SpecIndex

	// create resolved and un-resolved indexes.
	if modelIndex != nil {
		indexResolved = modelIndex
	} else {
		indexResolved = index.NewSpecIndexWithConfig(specResolved, config)
	}
	indexUnresolved = index.NewSpecIndexWithConfig(specUnresolved, config)

	// create a resolver
	resolverInstance := resolver.NewResolver(indexResolved)

	// resolve the doc
	resolvingErrors := resolverInstance.Resolve()
	for i := range docModelErrors {
		if m, ok := docModelErrors[i].(*resolver.ResolvingError); ok {
			resolvingErrors = append(resolvingErrors, m)
		}
	}

	// check references can be resolved correctly and are not infinite loops.
	resolvingRule := &model.Rule{
		Name:         "Check references can be resolved correctly",
		Id:           "resolving-references",
		Description:  "$ref values must be resolvable and locatable within a local or remote document.",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         "validation",
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "blank",
		},
		HowToFix: "Ensure that all $ref values are resolvable and locatable within a local or remote document. " + CircularReferencesFix,
	}

	// add all resolving errors to the results.
	for _, er := range resolvingErrors {
		res := model.RuleFunctionResult{
			RuleId:    "resolving-references",
			Rule:      resolvingRule,
			StartNode: er.Node,
			EndNode:   er.Node,
			Message:   er.Error(),
			Path:      er.Path,
		}
		ruleResults = append(ruleResults, res)
	}

	for _, er := range indexResolved.GetReferenceIndexErrors() {
		idxError := er.(*index.IndexingError)
		res := model.RuleFunctionResult{
			RuleId:    "resolving-references",
			Rule:      resolvingRule,
			StartNode: idxError.Node,
			EndNode:   idxError.Node,
			Message:   idxError.Error(),
			Path:      idxError.Path,
		}
		ruleResults = append(ruleResults, res)
	}

	// run all rules.
	var errors []error

	if execution.RuleSet != nil {
		for _, rule := range execution.RuleSet.Rules {
			ruleSpec := specResolved
			ruleIndex := indexResolved
			if !rule.Resolved {
				ruleSpec = specUnresolved
				ruleIndex = indexUnresolved
			}

			// this list of things is most likely going to grow a bit, so we use a nice clean message design.
			ctx := ruleContext{
				rule:             rule,
				specNode:         ruleSpec,
				builtinFunctions: builtinFunctions,
				ruleResults:      &ruleResults,
				wg:               &ruleWaitGroup,
				errors:           &errors,
				specInfo:         specInfo,
				index:            ruleIndex,
				document:         doc,
				customFunctions:  execution.CustomFunctions,
				silenceLogs:      execution.SilenceLogs,
			}
			if execution.PanicFunction != nil {
				ctx.panicFunc = execution.PanicFunction
			}
			go runRule(ctx)
		}

		ruleWaitGroup.Wait()
	}

	return &RuleSetExecutionResult{
		RuleSetExecution: execution,
		Results:          ruleResults,
		Index:            indexResolved,
		SpecInfo:         specInfo,
		Errors:           errors,
	}
}

func runRule(ctx ruleContext) {

	if ctx.panicFunc != nil {
		defer func() {
			if r := recover(); r != nil {
				ctx.panicFunc(r)
			}
		}()
	}
	defer ctx.wg.Done()

	var givenPaths []string
	if x, ok := ctx.rule.Given.(string); ok {
		givenPaths = append(givenPaths, x)
	}

	if x, ok := ctx.rule.Given.([]interface{}); ok {
		for _, gpI := range x {
			if gp, ok := gpI.(string); ok {
				givenPaths = append(givenPaths, gp)
			}
			// TODO: come back and clean this up if it proves to be required.
			// Not sure why I added this check for a given field, it's always a string path.
			//if gp, ok := gpI.(int); ok { //
			//	givenPaths = append(givenPaths, fmt.Sprintf("%v", gp))
			//}
		}
	}

	// for hardcoded Given statements, use []string directly
	if x, ok := ctx.rule.Given.([]string); ok {
		givenPaths = x
	}

	for _, givenPath := range givenPaths {

		var nodes []*yaml.Node
		var err error
		if givenPath != "$" {
			nodes, err = utils.FindNodesWithoutDeserializing(ctx.specNode, givenPath)
		} else {
			// if we're looking for the root, don't bother looking, we already have it.
			nodes = []*yaml.Node{ctx.specNode}
		}

		if err != nil {
			*ctx.errors = append(*ctx.errors, err)
			return
		}
		if len(nodes) <= 0 {
			continue
		}

		// check for a single action
		var ruleAction model.RuleAction
		err = mapstructure.Decode(ctx.rule.Then, &ruleAction)

		if err == nil {

			ctx.ruleResults = buildResults(ctx, ruleAction, nodes)

		} else {

			// check for multiple actions.
			var ruleActions []model.RuleAction
			err = mapstructure.Decode(ctx.rule.Then, &ruleActions)

			if err == nil {
				for _, rAction := range ruleActions {
					ctx.ruleResults = buildResults(ctx, rAction, nodes)
				}
			}
		}
	}
}

var lock sync.Mutex

func buildResults(ctx ruleContext, ruleAction model.RuleAction, nodes []*yaml.Node) *[]model.RuleFunctionResult {

	ruleFunction := ctx.builtinFunctions.FindFunction(ruleAction.Function)
	// not found, check if it's been registered as a custom function
	if ruleFunction == nil {
		if ctx.customFunctions != nil {
			if ctx.customFunctions[ruleAction.Function] != nil {
				ruleFunction = ctx.customFunctions[ruleAction.Function]
			}
		}
	}

	if ruleFunction != nil {

		rfc := model.RuleFunctionContext{
			Options:    ruleAction.FunctionOptions,
			RuleAction: &ruleAction,
			Rule:       ctx.rule,
			Given:      ctx.rule.Given,
			Index:      ctx.index,
			SpecInfo:   ctx.specInfo,
			Document:   ctx.document,
		}

		if ctx.specInfo.SpecFormat == "" && ctx.specInfo.Version == "" {
			if !ctx.silenceLogs {
				pterm.Warning.Printf("Specification version not detected, cannot apply rule `%s`\n", ctx.rule.Id)
			}
			return ctx.ruleResults
		}

		// validate the rule is configured correctly before running it.
		res, errs := model.ValidateRuleFunctionContextAgainstSchema(ruleFunction, rfc)
		if !res {
			for _, e := range errs {
				lock.Lock()
				*ctx.ruleResults = append(*ctx.ruleResults, model.RuleFunctionResult{Message: e})
				lock.Unlock()
			}
		} else {

			// iterate through nodes and supply them one at a time so we don't pollute each run
			for _, node := range nodes {

				// if this rule is designed for a different version, skip it.
				if len(ctx.rule.Formats) > 0 {
					match := false
					for _, format := range ctx.rule.Formats {
						if format == ctx.specInfo.SpecFormat {
							match = true
						}
					}
					if ctx.specInfo.SpecFormat != "" && !match {
						continue // does not apply to this spec.
					}
				}

				runRuleResults := ruleFunction.RunRule([]*yaml.Node{node}, rfc)

				// because this function is running in multiple threads, we need to sync access to the final result
				// list, otherwise things can get a bit random.
				lock.Lock()
				*ctx.ruleResults = append(*ctx.ruleResults, runRuleResults...)
				lock.Unlock()
			}

		}
	}
	return ctx.ruleResults
}
