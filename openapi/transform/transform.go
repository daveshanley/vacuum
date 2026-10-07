// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package transform

import (
	"github.com/pb33f/go-yaml"
	libutils "github.com/pb33f/libopenapi/utils"
)

// Options selects publication transforms. Filtering runs before pruning so only
// components reachable from retained operations remain.
type Options struct {
	TagFilter   TagFilterOptions
	PruneUnused bool
}

// Transform filters operations and prunes unused components with one YAML
// preparation and bundled-reference validation pass. Standalone filter and prune
// entry points use this same path.
func Transform(root *yaml.Node, specVersion string, options Options) (FilterStats, PruneStats, error) {
	var filter FilterStats
	var prune PruneStats
	if err := ValidateTagFilterOptions(options.TagFilter); err != nil {
		return filter, prune, err
	}
	original, err := requireMappingRoot(root)
	if err != nil {
		return filter, prune, err
	}
	filtering := len(options.TagFilter.IncludeTags) > 0
	if !filtering && !options.PruneUnused {
		return filter, prune, nil
	}
	root, err = prepareRoot(original, filtering)
	if err != nil {
		return filter, prune, err
	}
	// Filtering can succeed before pruning discovers an invalid local reference.
	// Keep combined transforms atomic without repeating preparation or validation.
	if filtering && options.PruneUnused && root == original {
		root = libutils.CloneYAMLNode(root)
	}
	if filtering {
		filter = filterOperationsByTags(root, specVersion, options.TagFilter)
	}
	if options.PruneUnused {
		prune, err = pruneUnusedComponents(root, specVersion)
		if err != nil {
			return filter, prune, err
		}
		filter.Warnings = RetainWarningsForPrunedDocument(filter.Warnings, prune)
	}
	if root != original {
		*original = *root
	}
	return filter, prune, nil
}

// prepareRoot validates before any publication edits. Alias references are
// checked in their use-site context even when pruning keeps the aliases intact.
func prepareRoot(root *yaml.Node, aliases bool) (*yaml.Node, error) {
	root, err := requireMappingRoot(root)
	if err != nil {
		return nil, err
	}
	root, err = expandYAMLReferences(root, aliases)
	if err != nil {
		return nil, err
	}
	if err = walkExternalReferences(root, nil); err != nil {
		return nil, err
	}
	return root, nil
}
