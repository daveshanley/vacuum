package cui

import (
	"strings"
	"testing"
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
				ASCIILightGrey + "Parent",
				ASCIIRed + " -> ",
				ASCIILightGrey + "Child",
				ASCIILightGrey + "Parent",
			},
		},
		{
			name:  "Complex circular reference",
			input: "payment_intent -> customer -> bank_account -> account",
			contains: []string{
				ASCIILightGrey + "payment_intent",
				ASCIIRed + " -> ",
				ASCIILightGrey + "customer",
				ASCIILightGrey + "bank_account",
				ASCIILightGrey + "account",
			},
		},
		{
			name:  "JSON path (no arrows)",
			input: "$.components.schemas.Parent",
			contains: []string{
				ASCIIGrey + "$.components.schemas.Parent",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ColorizePath(tt.input)

			// check that all expected substrings are present
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("ColorizePath(%q) = %q, missing expected substring %q",
						tt.input, result, expected)
				}
			}

			// ensure it ends with reset
			if !strings.HasSuffix(result, ASCIIReset) {
				t.Errorf("ColorizePath(%q) = %q, doesn't end with ASCIIReset",
					tt.input, result)
			}
		})
	}
}
