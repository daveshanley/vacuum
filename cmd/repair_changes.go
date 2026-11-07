// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"

	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
)

// repairChangeFilter maps report lines from the original source to the verified
// candidate. JSON pointers retain model ownership when serialization moves lines.
func repairChangeFilter(changes *wcModel.DocumentChanges, result *motor.RuleSetExecutionResult) *utils.ChangeFilter {
	execution := result.RuleSetExecution
	if len(result.ModifiedSpec) == 0 {
		return utils.NewChangeFilter(changes, execution.DrDocument)
	}
	original, err := repair.ParseDocument(execution.Spec)
	if err != nil {
		return utils.NewChangeFilter(nil, execution.DrDocument)
	}
	candidate, err := repair.ParseDocument(result.ModifiedSpec)
	if err != nil {
		return utils.NewChangeFilter(nil, execution.DrDocument)
	}
	oldFilter := utils.NewChangeFilter(changes, nil)
	oldLocations := repair.IndexLocations(original)
	newLocations := repair.IndexLocations(candidate)
	selected := make(map[string]bool)
	coveredLines := make(map[int]bool)
	oldSequences := make(map[string]*yaml.Node)
	newSequences := make(map[string]*yaml.Node)
	newLines := make(map[string]int)
	walkRepairNodes(original, func(n *yaml.Node) {
		if n.Kind == yaml.SequenceNode {
			if path, ok := oldLocations.Find(n); ok {
				oldSequences[path] = n
			}
		}
		if oldFilter.IsLineChanged(n.Line) {
			coveredLines[n.Line] = true
			if path, ok := oldLocations.Find(n); ok {
				selected[path] = true
			}
		}
	})
	walkRepairNodes(candidate, func(n *yaml.Node) {
		if n.Kind == yaml.SequenceNode {
			if path, ok := newLocations.Find(n); ok {
				newSequences[path] = n
			}
		}
		if path, ok := newLocations.Find(n); ok && n.Line > 0 {
			newLines[path] = n.Line
		}
	})
	// Unknown source lines cannot justify dropping a finding.
	if len(selected) == 0 || len(coveredLines) < oldFilter.GetChangedLineCount() {
		return utils.NewChangeFilter(nil, execution.DrDocument)
	}
	lines := make([]int, 0, len(selected))
	checkedSequences := make(map[string]bool)
	for path := range selected {
		// A changed array can shift positional pointers to different objects.
		// Retain findings rather than guess which object the old line selected.
		sequencePath := ""
		hasSequence := false
		for ancestor := path; ; {
			if _, ok := oldSequences[ancestor]; ok {
				sequencePath = ancestor
				hasSequence = true
			}
			if ancestor == "" {
				break
			}
			ancestor = ancestor[:strings.LastIndex(ancestor, "/")]
		}
		if hasSequence && !checkedSequences[sequencePath] {
			if !sameRepairNodes(oldSequences[sequencePath], newSequences[sequencePath]) {
				return utils.NewChangeFilter(nil, execution.DrDocument)
			}
			checkedSequences[sequencePath] = true
		}
		for {
			if line, ok := newLines[path]; ok {
				lines = append(lines, line)
				break
			}
			if path == "" {
				return utils.NewChangeFilter(nil, execution.DrDocument)
			}
			path = path[:strings.LastIndex(path, "/")]
		}
	}
	return utils.NewChangeFilterFromLines(lines, execution.DrDocument)
}

func walkRepairNodes(root *yaml.Node, visit func(*yaml.Node)) {
	if root == nil {
		return
	}
	visit(root)
	for _, child := range root.Content {
		walkRepairNodes(child, visit)
	}
}

func sameRepairNodes(a, b *yaml.Node) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind != b.Kind || a.Tag != b.Tag || a.Value != b.Value || len(a.Content) != len(b.Content) {
		return false
	}
	for i := range a.Content {
		if !sameRepairNodes(a.Content[i], b.Content[i]) {
			return false
		}
	}
	return true
}
