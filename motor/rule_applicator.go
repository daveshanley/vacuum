// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package motor

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/mitchellh/mapstructure"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type ruleContext struct {
	rule              *model.Rule
	specNode          *yaml.Node
	builtinFunctions  functions.Functions
	ruleResults       *[]model.RuleFunctionResult
	wg                *sync.WaitGroup
	errors            *[]error
	index             *index.SpecIndex
	specInfo          *datamodel.SpecInfo
	customFunctions   map[string]model.RuleFunction
	panicFunc         func(p any)
	silenceLogs       bool
	document          libopenapi.Document
	skipDocumentCheck bool
	logger            *slog.Logger
}

// RuleSetExecution is an instruction set for executing a ruleset. It's a convenience structure to allow the signature
// of ApplyRulesToRuleSet to change, without a huge refactor. The ApplyRulesToRuleSet function only returns a single error also.
type RuleSetExecution struct {
	RuleSet           *rulesets.RuleSet             // The RuleSet in which to apply
	SpecFileName      string                        // The name of the specification file, used to correctly label location
	Spec              []byte                        // The raw bytes of the OpenAPI specification.
	SpecInfo          *datamodel.SpecInfo           // Pre-parsed spec-info.
	CustomFunctions   map[string]model.RuleFunction // custom functions loaded from plugin.
	PanicFunction     func(p any)                   // In case of emergency, do this thing here.
	SilenceLogs       bool                          // Prevent any warnings about rules/rule-sets being printed.
	Base              string                        // The base path or URL of the specification, used for resolving relative or remote paths.
	AllowLookup       bool                          // Allow remote lookup of files or links
	Document          libopenapi.Document           // a ready to render model.
	SkipDocumentCheck bool                          // Skip the document check, useful for fragments and non openapi specs.
	Logger            *slog.Logger                  // A custom logger.
}

// RuleSetExecutionResult returns the results of running the ruleset against the supplied spec.
type RuleSetExecutionResult struct {
	RuleSetExecution *RuleSetExecution          // The execution struct that was used invoking the result.
	Results          []model.RuleFunctionResult // The results of the execution.
	Index            *index.SpecIndex           // The index that was created from the specification, used by the rules.
	SpecInfo         *datamodel.SpecInfo        // A reference to the SpecInfo object, used by all the rules.
	Errors           []error                    // Any errors that were returned.
	FilesProcessed   int                        // number of files extracted by the rolodex
	FileSize         int64                      // total filesize loaded by the rolodex
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
	indexConfig := index.CreateClosedAPIIndexConfig()
	indexConfigUnresolved := index.CreateClosedAPIIndexConfig()

	// avoid building the index, we don't need it to run yet.
	indexConfig.AvoidBuildIndex = true
	indexConfig.AvoidCircularReferenceCheck = true
	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.SkipCircularReferenceCheck = true

	// add new pretty logger.
	if execution.Logger == nil {
		var logger *slog.Logger
		if execution.SilenceLogs {
			// logger that goes to no-where
			logger = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))
		} else {
			handler := pterm.NewSlogHandler(&pterm.Logger{
				Formatter: pterm.LogFormatterColorful,
				Writer:    os.Stdout,
				Level:     pterm.LogLevelError,
				ShowTime:  false,
				MaxWidth:  280,
				KeyStyles: map[string]pterm.Style{
					"error":  *pterm.NewStyle(pterm.FgRed, pterm.Bold),
					"err":    *pterm.NewStyle(pterm.FgRed, pterm.Bold),
					"caller": *pterm.NewStyle(pterm.FgGray, pterm.Bold),
				},
			})
			logger = slog.New(handler)
			pterm.DefaultLogger.Level = pterm.LogLevelError
		}
		docConfig.Logger = logger
		indexConfig.Logger = logger

	} else {
		docConfig.Logger = execution.Logger
		indexConfig.Logger = execution.Logger
	}

	if execution.Base != "" {
		// check if this is a URL or not
		u, e := url.Parse(execution.Base)
		if e == nil && u.Scheme != "" && u.Host != "" {
			indexConfig.BaseURL = u
			indexConfig.BasePath = ""
			indexConfigUnresolved.BaseURL = u
			indexConfigUnresolved.BasePath = ""
			docConfig.BaseURL = u
			docConfig.BasePath = ""
			indexConfig.AllowRemoteLookup = true
			indexConfigUnresolved.AllowRemoteLookup = true

		} else {
			indexConfig.AllowFileLookup = true
			indexConfig.BasePath = execution.Base
			indexConfigUnresolved.AllowFileLookup = true
			indexConfigUnresolved.BasePath = execution.Base
			docConfig.BasePath = execution.Base
		}
	}

	if execution.AllowLookup {
		if indexConfig.BasePath == "" {
			indexConfig.BasePath = "."
			docConfig.BasePath = "."
		}
		indexConfig.AllowFileLookup = true
		indexConfigUnresolved.AllowFileLookup = true
		indexConfig.AllowRemoteLookup = true
		indexConfigUnresolved.AllowRemoteLookup = true
		docConfig.AllowRemoteReferences = true
	}

	if execution.SkipDocumentCheck {
		docConfig.BypassDocumentCheck = true
	}

	docResolved := execution.Document
	var docUnresolved libopenapi.Document

	// If no docResolved is supplied (default) then create a new one.
	// otherwise update the configuration with the supplied document.
	// and build it.

	var specInfo, specInfoUnresolved *datamodel.SpecInfo
	if docResolved == nil {
		var err error
		// create a new document.

		done := make(chan bool)

		go func() {
			docResolved, err = libopenapi.NewDocumentWithConfiguration(execution.Spec, docConfig)
			done <- true
		}()

		go func() {
			docUnresolved, _ = libopenapi.NewDocumentWithConfiguration(execution.Spec, docConfig)
			done <- true
		}()

		complete := 0
		for complete < 2 {
			<-done
			complete++
		}

		if err != nil {
			// Done here, we can't do anything else.
			return &RuleSetExecutionResult{Errors: []error{err}}
		}

		specInfo = docResolved.GetSpecInfo()
		specInfoUnresolved = docUnresolved.GetSpecInfo()
		indexConfig.SpecInfo = specInfo

	} else {

		var uErr error
		docUnresolved, uErr = libopenapi.NewDocumentWithConfiguration(*docResolved.GetSpecInfo().SpecBytes, docConfig)
		if uErr != nil {
			// Done here, we can't do anything else.
			return &RuleSetExecutionResult{Errors: []error{uErr}}
		}

		specInfo = docResolved.GetSpecInfo()
		specInfoUnresolved = docUnresolved.GetSpecInfo()

		suppliedDocConfig := docResolved.GetConfiguration()
		docConfig.BaseURL = suppliedDocConfig.BaseURL
		docConfig.BasePath = suppliedDocConfig.BasePath
		docConfig.IgnorePolymorphicCircularReferences = suppliedDocConfig.IgnorePolymorphicCircularReferences
		docConfig.IgnoreArrayCircularReferences = suppliedDocConfig.IgnoreArrayCircularReferences
		docConfig.AvoidIndexBuild = suppliedDocConfig.AvoidIndexBuild
		indexConfig.SpecInfo = specInfo
		indexConfig.AvoidBuildIndex = suppliedDocConfig.AvoidIndexBuild
		indexConfig.IgnorePolymorphicCircularReferences = suppliedDocConfig.IgnorePolymorphicCircularReferences
		indexConfig.IgnoreArrayCircularReferences = suppliedDocConfig.IgnoreArrayCircularReferences
		indexConfigUnresolved.SpecInfo = specInfoUnresolved
		indexConfigUnresolved.AvoidBuildIndex = suppliedDocConfig.AvoidIndexBuild
		indexConfigUnresolved.IgnorePolymorphicCircularReferences = suppliedDocConfig.IgnorePolymorphicCircularReferences
		indexConfigUnresolved.IgnoreArrayCircularReferences = suppliedDocConfig.IgnoreArrayCircularReferences
	}

	// build model
	var resolvedModelErrors []error
	var indexResolved *index.SpecIndex
	var indexUnresolved *index.SpecIndex

	version := docResolved.GetVersion()

	var resolvingErrors []*index.ResolvingError
	var circularReferences []*index.CircularReferenceResult

	var rolodexResolved, rolodexUnresolved *index.Rolodex

	if version != "" {
		switch version[0] {
		case '2':
			_, resolvedModelErrors = docResolved.BuildV2Model()
			rolodexResolved = docResolved.GetRolodex()

			_, _ = docUnresolved.BuildV2Model()
			rolodexUnresolved = docUnresolved.GetRolodex()

			indexResolved = rolodexResolved.GetRootIndex()
			indexUnresolved = rolodexUnresolved.GetRootIndex()

			// we only resolve one.
			rolodexResolved.Resolve()

			specResolved = rolodexResolved.GetRootIndex().GetRootNode()
			specUnresolved = rolodexUnresolved.GetRootIndex().GetRootNode()

			if rolodexResolved != nil && rolodexResolved.GetRootIndex() != nil {
				resolvingErrors = rolodexResolved.GetRootIndex().GetResolver().GetResolvingErrors()
				circularReferences = rolodexResolved.GetRootIndex().GetResolver().GetCircularReferences()
			}

		case '3':
			_, resolvedModelErrors = docResolved.BuildV3Model()
			rolodexResolved = docResolved.GetRolodex()

			_, _ = docUnresolved.BuildV3Model()
			rolodexUnresolved = docUnresolved.GetRolodex()

			indexResolved = rolodexResolved.GetRootIndex()
			indexUnresolved = rolodexUnresolved.GetRootIndex()

			// we only resolve one.
			rolodexResolved.Resolve()

			specResolved = rolodexResolved.GetRootIndex().GetRootNode()
			specUnresolved = rolodexUnresolved.GetRootIndex().GetRootNode()

			if rolodexResolved != nil && rolodexResolved.GetRootIndex() != nil {
				resolvingErrors = rolodexResolved.GetRootIndex().GetResolver().GetResolvingErrors()
				circularReferences = rolodexResolved.GetRootIndex().GetResolver().GetCircularReferences()
			}

		}
	} else {

		unresRoloConfig := *indexConfig
		resRoloConfig := *indexConfig

		completeChan := make(chan bool)

		go func() {
			// create an index for the unresolved spec.
			rolodexResolved, _ = BuildRolodexFromIndexConfig(&resRoloConfig)
			rolodexResolved.SetRootNode(resRoloConfig.SpecInfo.RootNode)

			_ = rolodexResolved.IndexTheRolodex()
			rolodexResolved.Resolve()
			completeChan <- true
		}()

		go func() {
			unResInfo, _ := datamodel.ExtractSpecInfo(*specInfo.SpecBytes)
			rolodexUnresolved, _ = BuildRolodexFromIndexConfig(&unresRoloConfig)
			rolodexUnresolved.SetRootNode(unResInfo.RootNode)

			_ = rolodexUnresolved.IndexTheRolodex()
			completeChan <- true
		}()

		completedBuilds := 0
		for completedBuilds < 2 {
			<-completeChan
			completedBuilds++
		}

		indexResolved = rolodexResolved.GetRootIndex()
		indexUnresolved = rolodexUnresolved.GetRootIndex()

		specResolved = rolodexResolved.GetRootNode()
		specUnresolved = rolodexUnresolved.GetRootNode()

		if rolodexResolved != nil && rolodexResolved.GetRootIndex() != nil {
			resolvingErrors = rolodexResolved.GetRootIndex().GetResolver().GetResolvingErrors()
			circularReferences = rolodexResolved.GetRootIndex().GetResolver().GetCircularReferences()
		}
	}

	for i := range resolvedModelErrors {
		var m *index.ResolvingError
		if errors.As(resolvedModelErrors[i], &m) {
			resolvingErrors = append(resolvingErrors, m)
		}
	}

	// re-map resolved index (important, the resolved index is not yet mapped)
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

	// add all circular reference errors to the results.
	circularRefRule := &model.Rule{
		Name:         "Circular References",
		Id:           "circular-references",
		Description:  "Circular reference detected",
		Message:      "Circular reference detected",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         "validation",
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "blank",
		},
		HowToFix: CircularReferencesFix,
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

	// add all circular references to the results.
	for _, cr := range circularReferences {
		res := model.RuleFunctionResult{
			RuleId:    "circular-references",
			Rule:      circularRefRule,
			StartNode: cr.Start.Node,
			EndNode:   cr.LoopPoint.Node,
			Message:   fmt.Sprintf("Circular reference detected from %s", cr.Start.Definition),
			Path:      cr.GenerateJourneyPath(),
		}
		ruleResults = append(ruleResults, res)
	}

	for _, er := range indexResolved.GetReferenceIndexErrors() {
		var idxError *index.IndexingError
		errors.As(er, &idxError)
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
	var errs []error

	if execution.RuleSet != nil {
		for _, rule := range execution.RuleSet.Rules {
			ruleSpec := specResolved
			ruleIndex := indexResolved
			info := specInfo
			if !rule.Resolved {
				info = specInfoUnresolved
				ruleSpec = specUnresolved
				ruleIndex = indexUnresolved
			}

			// this list of things is most likely going to grow a bit, so we use a nice clean message design.
			ctx := ruleContext{
				rule:              rule,
				specNode:          ruleSpec,
				builtinFunctions:  builtinFunctions,
				ruleResults:       &ruleResults,
				wg:                &ruleWaitGroup,
				errors:            &errs,
				specInfo:          info,
				index:             ruleIndex,
				document:          docResolved,
				customFunctions:   execution.CustomFunctions,
				silenceLogs:       execution.SilenceLogs,
				skipDocumentCheck: execution.SkipDocumentCheck,
				logger:            docConfig.Logger,
			}
			if execution.PanicFunction != nil {
				ctx.panicFunc = execution.PanicFunction
			}
			go runRule(ctx)
		}

		ruleWaitGroup.Wait()
	}

	ruleResults = *removeDuplicates(&ruleResults, execution, indexResolved)

	return &RuleSetExecutionResult{
		RuleSetExecution: execution,
		Results:          ruleResults,
		Index:            indexResolved,
		SpecInfo:         specInfo,
		Errors:           errs,
		FilesProcessed:   rolodexResolved.RolodexTotalFiles(),
		FileSize:         rolodexResolved.RolodexFileSize(),
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

	if x, ok := ctx.rule.Given.([]string); ok {
		givenPaths = x
	}

	if x, ok := ctx.rule.Given.([]interface{}); ok {
		for _, gpI := range x {
			if gp, ok := gpI.(string); ok {
				givenPaths = append(givenPaths, gp)
			}
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
			Document:   ctx.document,
			Logger:     ctx.logger,
		}

		if !ctx.skipDocumentCheck && ctx.specInfo.SpecFormat == "" && ctx.specInfo.Version == "" {
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

type seenResult struct {
	location string
	message  string
}

func removeDuplicates(results *[]model.RuleFunctionResult, rse *RuleSetExecution, idx *index.SpecIndex) *[]model.RuleFunctionResult {
	seen := make(map[string][]*seenResult)
	var newResults []model.RuleFunctionResult
	for _, result := range *results {
		if result.RuleId == "" && result.Rule != nil && result.Rule.Id != "" {
			result.RuleId = result.Rule.Id
		}
		if r, ok := seen[result.RuleId]; !ok {
			if result.StartNode != nil {
				seen[result.RuleId] = []*seenResult{
					{
						fmt.Sprintf("%d:%d", result.StartNode.Line, result.StartNode.Column),
						result.Message,
					},
				}
				origin := idx.FindNodeOrigin(result.StartNode)
				if origin != nil {
					if filepath.Base(origin.AbsoluteLocation) == "root.yaml" {
						origin.AbsoluteLocation = rse.SpecFileName
					}
					result.Origin = origin
				}
				newResults = append(newResults, result)
			}
		} else {
		stopNowPlease:
			for _, line := range r {
				if line.location == fmt.Sprintf("%d:%d", result.StartNode.Line, result.StartNode.Column) &&
					line.message == result.Message {
					break stopNowPlease
				}
				if result.StartNode != nil {
					seen[result.RuleId] = []*seenResult{
						{
							fmt.Sprintf("%d:%d", result.StartNode.Line, result.StartNode.Column),
							result.Message,
						},
					}
					origin := idx.FindNodeOrigin(result.StartNode)
					if origin != nil {
						if filepath.Base(origin.AbsoluteLocation) == "root.yaml" {
							origin.AbsoluteLocation = rse.SpecFileName
						}
						result.Origin = origin
					}
					newResults = append(newResults, result)
				}
			}
		}
	}

	return &newResults
}
