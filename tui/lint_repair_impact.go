// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"strings"

	"github.com/daveshanley/vacuum/motor/autofix"
)

func repairImpactLabel(impact autofix.RepairImpact) string {
	switch impact.Status {
	case autofix.RepairImpactBreaking:
		return fmt.Sprintf("Breaking changes detected: %d", impact.BreakingChanges)
	case autofix.RepairImpactNoBreaking:
		return "No breaking changes detected"
	default:
		return "Contract impact unknown"
	}
}

func repairImpactDetails(impact autofix.RepairImpact) string {
	var text strings.Builder
	text.WriteString(repairImpactLabel(impact))
	text.WriteByte('\n')
	if impact.Reason != "" {
		text.WriteString(impact.Reason)
		text.WriteByte('\n')
	}
	for _, change := range impact.Changes {
		if change != nil && change.Breaking {
			fmt.Fprintf(&text, "- %s: %q -> %q\n", change.Property, change.Original, change.New)
		}
	}
	return text.String()
}
