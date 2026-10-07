package transform

// MatchStrategy controls how operation tags are compared with included tags.
type MatchStrategy string

const (
	// MatchAny keeps an operation containing at least one included tag.
	MatchAny MatchStrategy = "any"
	// MatchAll keeps an operation containing every included tag.
	MatchAll MatchStrategy = "all"
)

// TagFilterOptions configures inclusive operation filtering.
type TagFilterOptions struct {
	IncludeTags   []string
	MatchStrategy MatchStrategy
}

// Warning describes a non-fatal publication problem found by a transform.
type Warning struct {
	Path    string
	Message string
	owner   *ComponentID
}

// FilterStats summarizes inclusive operation filtering.
type FilterStats struct {
	OperationsSeen       int
	OperationsKept       int
	OperationsRemoved    int
	PathItemsRemoved     int
	WebhooksRemoved      int
	CallbackItemsRemoved int
	Warnings             []Warning
}

// PruneStats summarizes component reachability pruning.
type PruneStats struct {
	ComponentsSeen    int
	ComponentsKept    int
	ComponentsRemoved int
	RemovedBySection  map[string]int
	removed           map[ComponentID]struct{}
}

// RetainWarningsForPrunedDocument removes warnings owned by components that
// were deleted by the supplied prune operation.
func RetainWarningsForPrunedDocument(warnings []Warning, prune PruneStats) []Warning {
	if len(warnings) == 0 || len(prune.removed) == 0 {
		return warnings
	}
	retained := make([]Warning, 0, len(warnings))
	for _, warning := range warnings {
		if warning.owner != nil {
			if _, removed := prune.removed[*warning.owner]; removed {
				continue
			}
		}
		retained = append(retained, warning)
	}
	return retained
}

// ComponentID uniquely identifies a reusable OpenAPI component.
type ComponentID struct {
	Section string
	Name    string
}
