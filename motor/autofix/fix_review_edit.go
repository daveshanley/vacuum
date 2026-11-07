// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

// The provider identifies a single value edit. Compound transforms must retain
// their linked edits and are never reduced to an independently editable field.
func editableProposalOperation(plan repair.Plan, rule, pointer, metadataField string, root *yaml.Node) (repair.Operation, bool) {
	if group := plan.ReviewedProposals[rule+"\x00"+pointer]; len(group) > 0 {
		if len(group) == 1 && group[0].Op == "copy" {
			node := fixNodeAt(root, group[0].From)
			if node == nil {
				return repair.Operation{}, false
			}
			value, err := utils.RenderRepairJSON(&yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{node}}, nil)
			if err != nil {
				return repair.Operation{}, false
			}
			op := group[0]
			op.Value = value
			return op, true
		}
		if len(group) == 1 && (group[0].Op == "add" || group[0].Op == "replace") {
			return group[0], true
		}
		return repair.Operation{}, false
	}
	if metadataField == "" {
		return repair.Operation{}, false
	}
	path := pointer + "/" + repair.EscapeToken(metadataField)
	for _, op := range plan.Operations {
		if op.Path == path && (op.Op == "add" || op.Op == "replace") {
			return op, true
		}
	}
	return repair.Operation{}, false
}
