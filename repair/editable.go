// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"github.com/pb33f/go-yaml"
	"strings"
)

// EditableFields identifies supported absent metadata fields in an unambiguous
// authored document. Keys are the original finding indices; values are field names.
// Existing empty or conflicting fields are not offered as missing-field repairs.
func EditableFields(data []byte, findings []Finding) map[int]string {
	fields := make(map[int]string)
	doc, err := ParseDocument(data)
	if err != nil || len(doc.Content) != 1 || validateTree(doc.Content[0], make(map[*yaml.Node]bool)) != nil {
		return fields
	}
	nodes := sourceNodes(doc.Content[0])
	version := field(doc.Content[0], "openapi")
	for _, finding := range findings {
		name := policyField(finding.RuleID)
		if name == "" || finding.Unavailable != "" || (finding.Code != "" && finding.Code != name+"-missing") {
			continue
		}
		owner := nodes[finding.Pointer]
		if owner != nil && owner.Kind == yaml.MappingNode && field(owner, name) == nil {
			// OAS 3.0 Reference Objects cannot gain annotation siblings. The
			// referenced owner's description needs its own source-owned finding.
			if version != nil && strings.HasPrefix(version.Value, "3.0.") && field(owner, "$ref") != nil {
				continue
			}
			fields[finding.Index] = name
		}
	}
	return fields
}
