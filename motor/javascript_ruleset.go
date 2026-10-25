package motor

import (
	"fmt"
	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
)

func bindExecutableRuleset(source *RuleSetExecution, control *executionControl) (*RuleSetExecution, func(), error) {
	scope, err := source.RuleSet.Executable.Open(control.Context(), source.FetchConfig)
	if err != nil {
		return nil, nil, err
	}
	fail := func(err error) (*RuleSetExecution, func(), error) { scope.Close(); return nil, nil, err }
	var data any
	if len(source.Spec) > 0 {
		err = yaml.Unmarshal(source.Spec, &data)
	} else if source.SpecInfo != nil && source.SpecInfo.RootNode != nil {
		err = source.SpecInfo.RootNode.Decode(&data)
	} else {
		err = fmt.Errorf("JavaScript ruleset requires document bytes or SpecInfo")
	}
	if err != nil {
		return fail(err)
	}
	native := map[string]model.RuleFunction{}
	builtins := functions.MapBuiltinFunctions()
	for _, name := range []string{"truthy", "falsy", "defined", "undefined", "pattern", "enumeration", "length", "alphabetical", "casing", "xor", "or", "unreferencedReusableObject", "schema"} {
		native[name] = builtins.FindFunction(name)
	}
	scope.SetNativeFunctions(native)
	matches, err := scope.MatchFormats(control.Context(), data, source.SpecFileName)
	if err != nil {
		return fail(err)
	}
	execution := &RuleSetExecution{
		RuleSet:                         source.RuleSet,
		SpecFileName:                    source.SpecFileName,
		Spec:                            source.Spec,
		SpecInfo:                        source.SpecInfo,
		IndexUnresolved:                 source.IndexUnresolved,
		IndexResolved:                   source.IndexResolved,
		CustomFunctions:                 source.CustomFunctions,
		AutoFixFunctions:                source.AutoFixFunctions,
		PanicFunction:                   source.PanicFunction,
		SilenceLogs:                     source.SilenceLogs,
		Base:                            source.Base,
		AllowLookup:                     source.AllowLookup,
		Document:                        source.Document,
		DrDocument:                      source.DrDocument,
		Arazzo:                          source.Arazzo,
		AsyncAPI:                        source.AsyncAPI,
		SkipDocumentCheck:               source.SkipDocumentCheck,
		Logger:                          source.Logger,
		Timeout:                         source.Timeout,
		NodeLookupTimeout:               source.NodeLookupTimeout,
		BuildGraph:                      source.BuildGraph,
		RenderChanges:                   source.RenderChanges,
		BuildDeepGraph:                  source.BuildDeepGraph,
		ExtractReferencesSequentially:   source.ExtractReferencesSequentially,
		ExtractReferencesFromExtensions: source.ExtractReferencesFromExtensions,
		ApplyAutoFixes:                  source.ApplyAutoFixes,
		CanonicalDocument:               source.CanonicalDocument,
		IgnoreCircularArrayRef:          source.IgnoreCircularArrayRef,
		IgnoreCircularPolymorphicRef:    source.IgnoreCircularPolymorphicRef,
		StorageRoot:                     source.StorageRoot,
		RolodexFS:                       source.RolodexFS,
		HTTPClientConfig:                source.HTTPClientConfig,
		FetchConfig:                     source.FetchConfig,
		TurboMode:                       source.TurboMode,
		SkipResolve:                     source.SkipResolve,
		SkipCircularCheck:               source.SkipCircularCheck,
		SkipSchemaErrors:                source.SkipSchemaErrors,
		SpecFormat:                      source.SpecFormat,
	}
	execution.RuleSet = source.RuleSet.CopyForExecution()
	execution.RuleSet.FormatMatches = matches
	execution.CustomFunctions = make(map[string]model.RuleFunction, len(source.CustomFunctions))
	for name, fn := range source.CustomFunctions {
		execution.CustomFunctions[name] = fn
	}
	for name, fn := range scope.Functions() {
		execution.CustomFunctions[name] = fn
	}
	return execution, scope.Close, nil
}
