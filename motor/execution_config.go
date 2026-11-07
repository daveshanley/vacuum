package motor

// CopyConfig creates an execution with the same settings and no runtime state.
// Parsed documents, worker counters and locks are never copied. Caller-owned
// function maps and read-only settings remain shared.
func (e *RuleSetExecution) CopyConfig() *RuleSetExecution {
	if e == nil {
		return &RuleSetExecution{}
	}
	next := &RuleSetExecution{
		SpecFileName:                    e.SpecFileName,
		Spec:                            e.Spec,
		CustomFunctions:                 e.CustomFunctions,
		AutoFixFunctions:                e.AutoFixFunctions,
		PanicFunction:                   e.PanicFunction,
		SilenceLogs:                     e.SilenceLogs,
		Base:                            e.Base,
		AllowLookup:                     e.AllowLookup,
		SkipDocumentCheck:               e.SkipDocumentCheck,
		Logger:                          e.Logger,
		Timeout:                         e.Timeout,
		NodeLookupTimeout:               e.NodeLookupTimeout,
		BuildGraph:                      e.BuildGraph,
		RenderChanges:                   e.RenderChanges,
		BuildDeepGraph:                  e.BuildDeepGraph,
		ExtractReferencesSequentially:   e.ExtractReferencesSequentially,
		ExtractReferencesFromExtensions: e.ExtractReferencesFromExtensions,
		RepairIgnores:                   e.RepairIgnores,
		ApplyAutoFixes:                  e.ApplyAutoFixes,
		IgnoreCircularArrayRef:          e.IgnoreCircularArrayRef,
		IgnoreCircularPolymorphicRef:    e.IgnoreCircularPolymorphicRef,
		StorageRoot:                     e.StorageRoot,
		RolodexFS:                       e.RolodexFS,
		HTTPClientConfig:                e.HTTPClientConfig,
		FetchConfig:                     e.FetchConfig,
		TurboMode:                       e.TurboMode,
		SkipResolve:                     e.SkipResolve,
		SkipCircularCheck:               e.SkipCircularCheck,
		SkipSchemaErrors:                e.SkipSchemaErrors,
		SpecFormat:                      e.SpecFormat,
	}
	if e.RuleSet != nil {
		next.RuleSet = e.RuleSet.CopyForExecution()
	}
	return next
}
