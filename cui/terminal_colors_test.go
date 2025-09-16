package cui

import (
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/color"
)

func TestColorizePath_CircularReferences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "Simple circular reference",
			input: "Parent -> Child -> Parent",
			contains: []string{
				color.ASCIILightGrey + "Parent",
				color.ASCIIRed + " -> ",
				color.ASCIILightGrey + "Child",
				color.ASCIILightGrey + "Parent",
			},
		},
		{
			name:  "Complex circular reference",
			input: "payment_intent -> customer -> bank_account -> account",
			contains: []string{
				color.ASCIILightGrey + "payment_intent",
				color.ASCIIRed + " -> ",
				color.ASCIILightGrey + "customer",
				color.ASCIILightGrey + "bank_account",
				color.ASCIILightGrey + "account",
			},
		},
		{
			name:  "JSON path (no arrows)",
			input: "$.components.schemas.Parent",
			contains: []string{
				color.ASCIIGrey + "$.components.schemas.Parent",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := color.ColorizePath(tt.input)

			// check that all expected substrings are present
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("ColorizePath(%q) = %q, missing expected substring %q",
						tt.input, result, expected)
				}
			}

			// ensure it contains ANSI reset codes (lipgloss handles this automatically)
			// Check for either \033[0m or \x1b[m (both are valid reset codes)
			if !strings.Contains(result, "\033[0m") && !strings.Contains(result, "\x1b[m") {
				t.Errorf("ColorizePath(%q) = %q, doesn't contain ANSI reset codes",
					tt.input, result)
			}
		})
	}
}
