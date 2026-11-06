package motor

import (
	"fmt"
	"sort"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
	whatChanged "github.com/pb33f/libopenapi/what-changed"
	changeModel "github.com/pb33f/libopenapi/what-changed/model"
)

// RepairImpactStatus describes the comparison result, not runtime compatibility.
type RepairImpactStatus string

const (
	// RepairImpactBreaking means the comparison found breaking contract changes.
	RepairImpactBreaking RepairImpactStatus = "breaking"
	// RepairImpactNoBreaking means a complete comparison found no breaking changes.
	RepairImpactNoBreaking RepairImpactStatus = "no-breaking-detected"
	// RepairImpactUnknown means a complete comparison was not possible.
	RepairImpactUnknown RepairImpactStatus = "unknown"
)

// RepairImpact describes the complete selected candidate relative to its source.
// Changes contain detached values; they do not retain either document graph.
type RepairImpact struct {
	Status          RepairImpactStatus    `json:"status"`
	Reason          string                `json:"reason,omitempty"`
	BreakingChanges int                   `json:"breakingChanges"`
	Changes         []*changeModel.Change `json:"changes,omitempty"`
}

// assessRepairImpact reuses models built by lint. It does not load new inputs or
// own the supplied results. Its caller must keep both results alive and serialize
// access. The native comparison is synchronous; cancellation is checked before
// and after it, without leaving a worker that retains the document graphs.
func assessRepairImpact(before, after *RuleSetExecutionResult, options *ExecutionOptions) (impact RepairImpact) {
	impact.Status = RepairImpactUnknown
	defer func() {
		if recovered := recover(); recovered != nil {
			impact = RepairImpact{Status: RepairImpactUnknown, Reason: "Contract comparison failed"}
		}
	}()
	if err := repairCanceled(options); err != nil {
		impact.Reason = err.Error()
		return
	}
	left, reason := repairImpactDocument(before)
	if reason != "" {
		impact.Reason = "Original document: " + reason
		return
	}
	right, reason := repairImpactDocument(after)
	if reason != "" {
		impact.Reason = "Proposed document: " + reason
		return
	}
	leftInfo, rightInfo := left.GetSpecInfo(), right.GetSpecInfo()
	if leftInfo == nil || rightInfo == nil || leftInfo.SpecFormat != rightInfo.SpecFormat {
		impact.Reason = "Document dialects cannot be compared"
		return
	}
	var changes *changeModel.DocumentChanges
	switch leftInfo.SpecType {
	case "openapi":
		l, le := left.BuildV3Model()
		r, re := right.BuildV3Model()
		if le != nil || re != nil || l == nil || r == nil {
			impact.Reason = "OpenAPI model comparison is incomplete"
			return
		}
		changes = whatChanged.CompareOpenAPIDocuments(l.Model.GoLow(), r.Model.GoLow())
	case "swagger":
		l, le := left.BuildV2Model()
		r, re := right.BuildV2Model()
		if le != nil || re != nil || l == nil || r == nil {
			impact.Reason = "Swagger model comparison is incomplete"
			return
		}
		changes = whatChanged.CompareSwaggerDocuments(l.Model.GoLow(), r.Model.GoLow())
	default:
		impact.Reason = "Contract comparison is unavailable for this document type"
		return
	}
	if err := repairCanceled(options); err != nil {
		impact.Reason = err.Error()
		return
	}
	impact.Status = RepairImpactNoBreaking
	if changes == nil {
		return
	}
	impact.BreakingChanges = changes.TotalBreakingChanges()
	if impact.BreakingChanges > 0 {
		impact.Status = RepairImpactBreaking
	}
	for _, change := range changes.GetAllChanges() {
		if change == nil {
			continue
		}
		copy := *change
		copy.OriginalObject, copy.NewObject = nil, nil
		if change.Context != nil {
			contextCopy := *change.Context
			cloneInt := func(value *int) *int {
				if value == nil {
					return nil
				}
				n := *value
				return &n
			}
			contextCopy.OriginalLine = cloneInt(contextCopy.OriginalLine)
			contextCopy.OriginalColumn = cloneInt(contextCopy.OriginalColumn)
			contextCopy.NewLine = cloneInt(contextCopy.NewLine)
			contextCopy.NewColumn = cloneInt(contextCopy.NewColumn)
			copy.Context = &contextCopy
		}
		impact.Changes = append(impact.Changes, &copy)
	}
	// The comparison traverses maps. Keep the user-visible result stable.
	sort.SliceStable(impact.Changes, func(i, j int) bool {
		a, b := impact.Changes[i], impact.Changes[j]
		return fmt.Sprint(repairChangeLocation(a), "\x00", a.Path, "\x00", a.Property, "\x00", a.ChangeType, "\x00", a.Original, "\x00", a.New) <
			fmt.Sprint(repairChangeLocation(b), "\x00", b.Path, "\x00", b.Property, "\x00", b.ChangeType, "\x00", b.Original, "\x00", b.New)
	})
	return
}

func repairImpactDocument(result *RuleSetExecutionResult) (libopenapi.Document, string) {
	if result == nil || result.RuleSetExecution == nil {
		return nil, "comparison inputs are unavailable"
	}
	e := result.RuleSetExecution
	if e.AsyncAPI != nil || e.Arazzo != nil || isJSONSchemaFormat(e.SpecFormat) {
		return nil, "contract comparison is unavailable for this document type"
	}
	if len(result.Errors) != 0 || result.releaseCleared.Load() {
		return nil, "validation did not complete"
	}
	if e.SkipResolve || e.SkipDocumentCheck || e.SkipSchemaErrors || e.SkipCircularCheck {
		return nil, "required document checks were skipped"
	}
	if e.DrDocument != nil && len(e.DrDocument.BuildErrors) != 0 {
		return nil, "schema model construction failed"
	}
	// Cached BuildV2/3Model calls do not repeat earlier construction errors.
	// Inspect the retained index errors before using their cached models.
	doc := result.ownedDocument
	if doc == nil || doc.GetRolodex() == nil {
		return nil, "validated document model is unavailable"
	}
	rolo := doc.GetRolodex()
	if len(rolo.GetCaughtErrors()) != 0 {
		return nil, "reference indexing did not complete"
	}
	indices := append([]*index.SpecIndex{rolo.GetRootIndex()}, rolo.GetIndexes()...)
	for _, idx := range indices {
		if idx == nil || len(idx.GetReferenceIndexErrors()) != 0 || len(idx.GetOperationParametersIndexErrors()) != 0 {
			return nil, "document indexing did not complete"
		}
		if resolver := idx.GetResolver(); resolver != nil && len(resolver.GetResolvingErrors()) != 0 {
			return nil, "reference resolution did not complete"
		}
	}
	return doc, ""
}

func repairChangeLocation(change *changeModel.Change) string {
	if change.Context == nil {
		return ""
	}
	value := func(n *int) int {
		if n == nil {
			return 0
		}
		return *n
	}
	c := change.Context
	return fmt.Sprintf("%s:%012d:%012d:%012d:%012d", c.DocumentLocation, value(c.OriginalLine), value(c.OriginalColumn), value(c.NewLine), value(c.NewColumn))
}
