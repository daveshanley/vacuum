package tui

import (
	"fmt"
	"github.com/daveshanley/vacuum/motor"
	"strings"
)

func repairImpactLabel(impact motor.RepairImpact) string {
	switch impact.Status {
	case motor.RepairImpactBreaking:
		return fmt.Sprintf("Breaking changes detected: %d", impact.BreakingChanges)
	case motor.RepairImpactNoBreaking:
		return "No breaking changes detected"
	default:
		return "Contract impact unknown"
	}
}

func repairImpactDetails(impact motor.RepairImpact) string {
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
