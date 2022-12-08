// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package motor

import (
	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/mitchellh/mapstructure"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
	"sync"
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
}

// RuleSetExecution is an instruction set for executing a ruleset. It's a convenience structure to allow the signature
// of ApplyRules to change, without a huge refactor. The ApplyRules function only returns a single error also.
type RuleSetExecution struct {
	RuleSet         *rulesets.RuleSet             // The RuleSet in which to apply
	Spec            []byte                        // The raw bytes of the OpenAPI specification.
	SpecInfo        *datamodel.SpecInfo           // Pre-parsed spec-info.
	CustomFunctions map[string]model.RuleFunction // custom functions loaded from plugin.
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

	var specInfo, specInfoUnresolved *datamodel.SpecInfo
	var err error
	if execution.SpecInfo == nil {
		// extract spec info, make this available to rule context.
		specInfo, err = datamodel.ExtractSpecInfo(execution.Spec)
		if err != nil || specInfo == nil {
			if specInfo == nil || specInfo.RootNode == nil {
				return &RuleSetExecutionResult{Errors: []error{err}}
			}
		}
		specInfoUnresolved, _ = datamodel.ExtractSpecInfo(execution.Spec)
	} else {
		specInfo = execution.SpecInfo
		specInfoUnresolved = execution.SpecInfo
	}

	specUnresolved = specInfoUnresolved.RootNode
	specResolved = specInfo.RootNode

	// create resolved and un-resolved indexes.
	indexResolved := index.NewSpecIndex(specResolved)
	indexUnresolved := index.NewSpecIndex(specUnresolved)

	// create a resolver
	resolverInstance := resolver.NewResolver(indexResolved)

	// resolve the doc
	resolverInstance.Resolve()

	// any errors (circular or lookup) from resolving spec.
	errs := resolverInstance.GetResolvingErrors()

	// create circular rule, it's blank, but we need a rule for a result.
	circularRule := &model.Rule{
		Name:         "Check for circular or missing references",
		Id:           "circular-references",
		Description:  "Specification schemas contain circular or missing references",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         "validation",
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "blank",
		},
		HowToFix: CircularReferencesFix,
	}

	// add all circular references to results.
	for _, er := range errs {
		res := model.RuleFunctionResult{
			Rule:      circularRule,
			StartNode: er.Node,
			EndNode:   er.Node,
			Message:   er.Error(),
			Path:      er.Path,
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
				customFunctions:  execution.CustomFunctions,
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

// Deprecated: ApplyRules will apply a loaded model.RuleSet against an OpenAPI specification.
// Please use ApplyRulesToRuleSet instead of this function, the signature needs to change.
func ApplyRules(ruleSet *rulesets.RuleSet, spec []byte) ([]model.RuleFunctionResult, error) {

	builtinFunctions := functions.MapBuiltinFunctions()
	var ruleResults []model.RuleFunctionResult
	var ruleWaitGroup sync.WaitGroup
	if ruleSet != nil && ruleSet.Rules != nil {
		ruleWaitGroup.Add(len(ruleSet.Rules))
	}

	var specResolved yaml.Node
	var specUnresolved yaml.Node

	// extract spec info, make this available to rule context.
	specInfo, err := datamodel.ExtractSpecInfo(spec)
	if err != nil || specInfo == nil {
		if specInfo == nil || specInfo.RootNode == nil {
			return nil, err
		}
	}

	specUnresolved = *specInfo.RootNode
	specResolved = specUnresolved

	// create an index
	index := index.NewSpecIndex(&specResolved)

	// create a resolver
	resolverInstance := resolver.NewResolver(index)

	// resolve the doc
	resolverInstance.Resolve()

	// any errors (circular or lookup) from resolving spec.
	errs := resolverInstance.GetResolvingErrors()

	// create circular rule, it's blank, but we need a rule for a result.
	circularRule := &model.Rule{
		Name:         "Check for circular references",
		Id:           "circular-references",
		Description:  "Specification schemas contain circular references",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         "validation",
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "blank",
		},
		HowToFix: CircularReferencesFix,
	}

	// add all circular references to results.
	for _, er := range errs {
		res := model.RuleFunctionResult{
			Rule:      circularRule,
			StartNode: er.Node,
			EndNode:   er.Node,
			Message:   er.Error(),
			Path:      er.Path,
		}
		ruleResults = append(ruleResults, res)
	}

	// run all rules.
	var errors []error

	if ruleSet != nil {
		for _, rule := range ruleSet.Rules {
			ruleSpec := &specResolved
			if !rule.Resolved {
				ruleSpec = &specUnresolved
			}

			// this list of things is most likely going to grow a bit, so we use a nice clean message design.
			ctx := ruleContext{
				rule:             rule,
				specNode:         ruleSpec,
				builtinFunctions: builtinFunctions,
				ruleResults:      &ruleResults,
				wg:               &ruleWaitGroup,
				errors:           &errors,
				index:            index,
				specInfo:         specInfo,
			}
			go runRule(ctx)
		}

		ruleWaitGroup.Wait()
	}

	return ruleResults, nil
}

func runRule(ctx ruleContext) {

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
		}

		if ctx.specInfo.SpecFormat == "" && ctx.specInfo.Version == "" {
			pterm.Warning.Printf("Specification version not detected, cannot apply rule `%s`\n", ctx.rule.Id)
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
