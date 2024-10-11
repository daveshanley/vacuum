// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"errors"
	"fmt"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/sourcegraph/conc"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/mitchellh/mapstructure"
	doctor "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
)

type ruleContext struct {
	rule               *model.Rule
	specNode           *yaml.Node
	specNodeUnresolved *yaml.Node
	builtinFunctions   functions.Functions
	ruleResults        *[]model.RuleFunctionResult
	errors             *[]error
	index              *index.SpecIndex
	specInfo           *datamodel.SpecInfo
	customFunctions    map[string]model.RuleFunction
	panicFunc          func(p any)
	silenceLogs        bool
	document           libopenapi.Document
	drDocument         *doctor.DrDocument
	skipDocumentCheck  bool
	logger             *slog.Logger
}

// RuleSetExecution is an instruction set for executing a ruleset. It's a convenience structure to allow the signature
// of ApplyRulesToRuleSet to change, without a huge refactor. The ApplyRulesToRuleSet function only returns a single error also.
type RuleSetExecution struct {
	RuleSet           *rulesets.RuleSet             // The RuleSet in which to apply
	SpecFileName      string                        // The path of the specification file, used to correctly label location
	Spec              []byte                        // The raw bytes of the OpenAPI specification.
	SpecInfo          *datamodel.SpecInfo           // Pre-parsed spec-info.
	CustomFunctions   map[string]model.RuleFunction // custom functions loaded from plugin.
	PanicFunction     func(p any)                   // In case of emergency, do this thing here.
	SilenceLogs       bool                          // Prevent any warnings about rules/rule-sets being printed.
	Base              string                        // The base path or URL of the specification, used for resolving relative or remote paths.
	AllowLookup       bool                          // Allow remote lookup of files or links
	Document          libopenapi.Document           // a ready to render model.
	DrDocument        *doctor.DrDocument            // a high level, more powerful model, powered by the doctor.
	SkipDocumentCheck bool                          // Skip the document check, useful for fragments and non openapi specs.
	Logger            *slog.Logger                  // A custom logger.
	Timeout           time.Duration                 // The timeout for each rule to run, prevents run-away rules, default is five seconds.
	BuildGraph        bool                          // Build a graph of the document, powered by the doctor. (default is false)

	// https://pb33f.io/libopenapi/circular-references/#circular-reference-results
	IgnoreCircularArrayRef       bool // Ignore array circular references
	IgnoreCircularPolymorphicRef bool // Ignore polymorphic circular references
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

	now := time.Now()
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
	indexConfig.SpecFilePath = execution.SpecFileName
	indexConfigUnresolved := index.CreateClosedAPIIndexConfig()
	indexConfigUnresolved.SpecFilePath = execution.SpecFileName

	// avoid building the index, we don't need it to run yet.
	indexConfig.AvoidBuildIndex = true
	//indexConfig.AvoidCircularReferenceCheck = true

	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.SpecFilePath = execution.SpecFileName
	//docConfig.SkipCircularReferenceCheck = true

	if execution.IgnoreCircularArrayRef {
		docConfig.IgnoreArrayCircularReferences = true
	}

	if execution.IgnoreCircularPolymorphicRef {
		docConfig.IgnorePolymorphicCircularReferences = true
	}

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

	indexConfig.Logger.Debug("applying rules to rule set")

	if execution.Base != "" {

		// check if this is a URL or not
		if strings.HasPrefix(execution.Base, "http") {

			if !strings.HasSuffix(execution.Base, "/") {
				execution.Base = execution.Base + "/"
			}
			u, _ := url.Parse(execution.Base)
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
		docConfig.AllowFileReferences = true
	}

	if execution.SkipDocumentCheck {
		docConfig.BypassDocumentCheck = true
	}

	docResolved := execution.Document
	var docUnresolved libopenapi.Document

	// If no docResolved is supplied (default) then create a new one.
	// otherwise update the configuration with the supplied document.
	// and build it.

	indexConfig.Logger.Debug("building documents")
	nowDocs := time.Now()

	var specInfo, specInfoUnresolved *datamodel.SpecInfo
	if docResolved == nil {
		var err error
		// create a new document.

		wg := conc.WaitGroup{}

		wg.Go(func() {
			docResolved, err = libopenapi.NewDocumentWithConfiguration(execution.Spec, docConfig)
		})

		wg.Go(func() {
			dc := *docConfig
			dc.SkipCircularReferenceCheck = false
			docUnresolved, _ = libopenapi.NewDocumentWithConfiguration(execution.Spec, &dc)
		})
		wg.Wait()

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

	timeTaken := time.Since(nowDocs).Milliseconds()
	indexConfig.Logger.Debug("building docs completed", "ms", timeTaken)

	// build model
	var resolvedModelErrors []error
	var indexResolved *index.SpecIndex
	var indexUnresolved *index.SpecIndex

	version := docResolved.GetVersion()

	var resolvingErrors []*index.ResolvingError
	var circularReferences []*index.CircularReferenceResult

	var rolodexResolved, rolodexUnresolved *index.Rolodex

	indexConfig.Logger.Debug("building document models")
	nowModel := time.Now()

	var v3DocumentModel *v3.Document
	//var v2DocumentModel *v2.Swagger

	var drDocument *doctor.DrDocument

	if version != "" {
		switch version[0] {
		case '2':
			_, resolvedModelErrors = docResolved.BuildV2Model()
			rolodexResolved = docResolved.GetRolodex()

			docUnresolved.BuildV2Model()
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

			now = time.Now()
			var mod *libopenapi.DocumentModel[v3.Document]

			_, resolvedModelErrors = docResolved.BuildV3Model()

			rolodexResolved = docResolved.GetRolodex()

			then := time.Since(now).Milliseconds()
			indexConfig.Logger.Debug("built resolved model", "ms", then)

			now = time.Now()
			var errs []error
			mod, errs = docUnresolved.BuildV3Model()
			if mod != nil {
				v3DocumentModel = &mod.Model
			} else {
				if execution.Logger != nil {
					execution.Logger.Error("unable to build unresolved model", "errors", errs)
				}
			}
			rolodexUnresolved = docUnresolved.GetRolodex()

			then = time.Since(now).Milliseconds()
			indexConfig.Logger.Debug("built unresolved model", "ms", then)

			indexResolved = rolodexResolved.GetRootIndex()
			indexUnresolved = rolodexUnresolved.GetRootIndex()

			wg := conc.WaitGroup{}
			wg.Go(func() {
				if v3DocumentModel != nil {
					var drDoc *doctor.DrDocument
					if !execution.BuildGraph {
						drDoc = doctor.NewDrDocument(mod)
					} else {
						drDoc = doctor.NewDrDocumentAndGraph(mod)
					}
					execution.DrDocument = drDoc
					drDocument = drDoc
				}
			})
			wg.Go(func() {
				// we only resolve one.
				resolvedTime := time.Now()
				rolodexResolved.Resolve()
				resolvedTaken := time.Since(resolvedTime).Milliseconds()
				indexConfig.Logger.Debug("resolved model", "ms", resolvedTaken)
			})
			wg.Wait()
			specResolved = rolodexResolved.GetRootIndex().GetRootNode()
			specUnresolved = rolodexUnresolved.GetRootIndex().GetRootNode()

			if rolodexResolved != nil && rolodexResolved.GetRootIndex() != nil {
				//resolvingErrors = rolodexResolved.GetRootIndex().GetResolver().GetResolvingErrors()
				circularReferences = rolodexResolved.GetRootIndex().GetResolver().GetCircularReferences()
			}

		}
	} else {

		unresRoloConfig := *indexConfig
		resRoloConfig := *indexConfig

		wg := conc.WaitGroup{}

		wg.Go(func() {
			// create an index for the unresolved spec.
			rolodexResolved, _ = BuildRolodexFromIndexConfig(&resRoloConfig)
			rolodexResolved.SetRootNode(resRoloConfig.SpecInfo.RootNode)

			_ = rolodexResolved.IndexTheRolodex()
			rolodexResolved.Resolve()
		})

		wg.Go(func() {
			unResInfo, _ := datamodel.ExtractSpecInfo(*specInfo.SpecBytes)
			rolodexUnresolved, _ = BuildRolodexFromIndexConfig(&unresRoloConfig)
			rolodexUnresolved.SetRootNode(unResInfo.RootNode)

			_ = rolodexUnresolved.IndexTheRolodex()
		})
		wg.Wait()

		indexResolved = rolodexResolved.GetRootIndex()
		indexUnresolved = rolodexUnresolved.GetRootIndex()

		specResolved = rolodexResolved.GetRootNode()
		specUnresolved = rolodexUnresolved.GetRootNode()

		if rolodexResolved != nil && rolodexResolved.GetRootIndex() != nil {
			resolvingErrors = rolodexResolved.GetRootIndex().GetResolver().GetResolvingErrors()
			circularReferences = rolodexResolved.GetRootIndex().GetResolver().GetCircularReferences()
		}
	}

	then := time.Since(nowModel).Milliseconds()
	indexConfig.Logger.Debug("built model", "ms", then)

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

	// checks if schemas can be programmatically built using libopenapi. If not, they will generally fail
	// any kind of validation or code generation.
	schemaBuildRule := &model.Rule{
		Name:         "Check schemas can be programmatically built",
		Id:           "schema-build-failure",
		Description:  "Schemas must be able to be programmatically built using automation",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         "validation",
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "blank",
		},
		HowToFix: "If a schema cannot be built programmatically, it will generally fail in any kind of tools. " +
			"The schema must be fixed, follow the error message to find the specific problem.",
	}

	// checks if an index was created for the document or not (could be parsed)
	indexBuildRule := &model.Rule{
		Name:         "Check that an index can be created from the document",
		Id:           "build-index",
		Description:  "vacuum must be able to index the document, if it cannot then it cannot be linted",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         "validation",
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "blank",
		},
		HowToFix: "An index is required to use vacuum. If an index cannot be created then the file cannot be read, or the OpenAPI version is not supported." +
			" Check your version of OpenAPI to start and if that looks correct, check the syntax of the document.",
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
			EndNode:   vacuumUtils.BuildEndNode(er.Node),
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
			StartNode: cr.ParentNode,
			EndNode:   vacuumUtils.BuildEndNode(cr.ParentNode),
			Message:   fmt.Sprintf("circular reference detected from %s", cr.Start.Definition),
			Path:      cr.GenerateJourneyPath(),
		}
		ruleResults = append(ruleResults, res)
	}

	if indexResolved != nil {
		for _, er := range indexResolved.GetReferenceIndexErrors() {
			var idxError *index.IndexingError
			errors.As(er, &idxError)
			res := model.RuleFunctionResult{
				RuleId:    "resolving-references",
				Rule:      resolvingRule,
				StartNode: idxError.Node,
				EndNode:   vacuumUtils.BuildEndNode(idxError.KeyNode),
				Message:   idxError.Error(),
				Path:      idxError.Path,
			}
			ruleResults = append(ruleResults, res)
		}
	}

	// run all rules.
	var errs []error

	// add dr document build errors to the results.
	if drDocument != nil {
		for _, er := range drDocument.BuildErrors {
			res := model.RuleFunctionResult{
				RuleId:    "schema-build-failure",
				Rule:      schemaBuildRule,
				StartNode: er.SchemaProxy.GoLow().GetKeyNode(),
				EndNode:   vacuumUtils.BuildEndNode(er.SchemaProxy.GoLow().GetKeyNode()),
				Message:   er.Error.Error(),
				Path:      er.DrSchemaProxy.GenerateJSONPath(),
			}
			ruleResults = append(ruleResults, res)
		}
	}

	// if there is no index, report an error.
	if indexUnresolved == nil {
		res := model.RuleFunctionResult{
			RuleId:    "index-failure",
			Rule:      indexBuildRule,
			StartNode: &yaml.Node{Line: 1, Column: 1},
			EndNode:   &yaml.Node{Line: 1, Column: 2},
			Message:   "unable to parse the document, no index was created, check the syntax or version of the document.",
			Path:      "$",
		}
		ruleResults = append(ruleResults, res)
	}

	if execution.RuleSet != nil && indexUnresolved != nil {

		totalRules := len(execution.RuleSet.Rules)
		done := make(chan bool)
		indexConfig.Logger.Debug("running rules", "total", totalRules)
		now = time.Now()

		if execution.Timeout <= 0 {
			execution.Timeout = time.Second * 5 // default
		}

		for _, rule := range execution.RuleSet.Rules {

			go func(rule *model.Rule, done chan bool) {

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
					rule:               rule,
					specNode:           ruleSpec,
					specNodeUnresolved: specUnresolved,
					builtinFunctions:   builtinFunctions,
					ruleResults:        &ruleResults,
					errors:             &errs,
					specInfo:           info,
					index:              ruleIndex,
					document:           docUnresolved,
					drDocument:         drDocument,
					customFunctions:    execution.CustomFunctions,
					silenceLogs:        execution.SilenceLogs,
					skipDocumentCheck:  execution.SkipDocumentCheck,
					logger:             docConfig.Logger,
				}
				if execution.PanicFunction != nil {
					ctx.panicFunc = execution.PanicFunction
				}

				timeoutCtx, ruleCancel := context.WithTimeout(context.Background(), execution.Timeout)
				defer ruleCancel()
				doneChan := make(chan bool)

				go runRule(ctx, doneChan)

				select {
				case <-timeoutCtx.Done():
					ctx.logger.Error("Rule timed out, skipping", "rule", rule.Id, "timeout", execution.Timeout)
					break
				case <-doneChan:
					break
				}
				done <- true
			}(rule, done)
		}

		completed := 0
		for completed < totalRules {
			<-done
			completed++
		}
		then = time.Since(now).Milliseconds()
		indexConfig.Logger.Debug("rules completed", "totalRules", totalRules, "ms", then)
	}

	filesProcessed := 0
	fileSize := int64(0)

	if indexResolved != nil && rolodexResolved != nil {
		filesProcessed = rolodexResolved.RolodexTotalFiles()
		fileSize = rolodexResolved.RolodexFileSize()
		//ruleResults = *removeDuplicates(&ruleResults, execution, indexResolved)
	}

	then = time.Since(now).Milliseconds()
	indexConfig.Logger.Debug("applied all rules and completed", "ms", then)

	return &RuleSetExecutionResult{
		RuleSetExecution: execution,
		Results:          ruleResults,
		Index:            indexResolved,
		SpecInfo:         specInfo,
		Errors:           errs,
		FilesProcessed:   filesProcessed,
		FileSize:         fileSize,
	}
}

func runRule(ctx ruleContext, doneChan chan bool) {

	if ctx.panicFunc != nil {
		defer func() {
			if r := recover(); r != nil {
				ctx.panicFunc(r)
			}
		}()
	}

	var givenPaths []string
	if x, ok := ctx.rule.Given.(string); ok {
		givenPaths = append(givenPaths, x)
	}

	if x, ok := ctx.rule.Given.([]string); ok {
		givenPaths = x
	}

	if x, ok := ctx.rule.Given.([]interface{}); ok {
		for _, gpI := range x {
			if gp, ko := gpI.(string); ko {
				givenPaths = append(givenPaths, gp)
			}
		}
	}

	findNodes := func(node *yaml.Node, path string, errChan chan error, nodesChan chan []*yaml.Node) {
		nodes, err := utils.FindNodesWithoutDeserializing(node, path)
		if err != nil {
			errChan <- err
		}
		nodesChan <- nodes
	}

	var nodes []*yaml.Node
	var err error

	for _, givenPath := range givenPaths {

		if givenPath != "$" {

			// create a timeout on this, if we can't get a result within 2s, then
			// try again, but with the unresolved spec.
			lookupCtx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer cancel()
			nodesChan := make(chan []*yaml.Node)
			errChan := make(chan error)

			go findNodes(ctx.specNode, givenPath, errChan, nodesChan)
		topBreak:
			select {
			case nodes = <-nodesChan:
				break
			case err = <-errChan:
				ctx.logger.Error("error looking for nodes", "path", givenPath, "rule", ctx.rule.Id, "error", err)
				break
			case <-lookupCtx.Done():
				ctx.logger.Warn("timeout looking for nodes, trying again with unresolved spec.", "path", givenPath)

				// ok, this timed out, let's try again with the unresolved spec.
				lookupCtxFinal, finalCancel := context.WithTimeout(context.Background(), time.Second*2)
				defer finalCancel()

				go findNodes(ctx.specNodeUnresolved, givenPath, errChan, nodesChan)

				select {
				case nodes = <-nodesChan:
					break
				case err = <-errChan:
					break
				case <-lookupCtxFinal.Done():
					err = fmt.Errorf("timed out looking for nodes using path '%s'", givenPath)
					ctx.logger.Error("timeout looking for unresolved nodes, giving up.", "path", givenPath, "rule",
						ctx.rule.Id)
					break topBreak
				}
			}

		} else {
			// if we're looking for the root, don't bother looking, we already have it.
			nodes = []*yaml.Node{ctx.specNode}
		}

		if err != nil {
			*ctx.errors = append(*ctx.errors, err)
			doneChan <- true
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
	doneChan <- true
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
			DrDocument: ctx.drDocument,
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
				*ctx.ruleResults = append(*ctx.ruleResults, model.RuleFunctionResult{
					Message:      e,
					Rule:         ctx.rule,
					StartNode:    &yaml.Node{},
					EndNode:      &yaml.Node{},
					RuleId:       ctx.rule.Id,
					RuleSeverity: ctx.rule.Severity,
					Path:         fmt.Sprint(ctx.rule.Given),
				})
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
				if result.StartNode == nil {
					if idx.GetLogger() != nil {
						idx.GetLogger().Error("[rule-applicator] start node is nil, no line numbers available", "rule", result.RuleId,
							"message", result.Message)
					}
					continue
				}

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
