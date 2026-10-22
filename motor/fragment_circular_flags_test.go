package motor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
)

func TestFragmentCircularArrayPolicy(t *testing.T) {
	for _, tc := range []struct {
		name         string
		schema       string
		wantCircular bool
	}{
		{"finite array", "type: object\n      required: [children]\n      properties:\n        children:\n          type: array\n          items:\n            $ref: '#/components/schemas/Node'", false},
		{"required infinite object", "type: object\n      required: [child]\n      properties:\n        child:\n          $ref: '#/components/schemas/Node'", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			child := []byte("components:\n  schemas:\n    Node:\n      " + tc.schema + "\n")
			if err := os.WriteFile(filepath.Join(directory, "node.yaml"), child, 0o600); err != nil {
				t.Fatal(err)
			}
			results := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
				Spec:                   []byte("components:\n  schemas:\n    Root:\n      $ref: './node.yaml#/components/schemas/Node'\n"),
				SpecFileName:           filepath.Join(directory, "root.yaml"),
				Base:                   directory,
				AllowLookup:            true,
				SkipDocumentCheck:      true,
				IgnoreCircularArrayRef: true,
				RuleSet:                &rulesets.RuleSet{Rules: map[string]*model.Rule{"oas3-schema": rulesets.GetOAS3SchemaRule()}},
			}, &ExecutionOptions{NestedRefsDocContext: true})
			defer results.Release()
			assert.Empty(t, results.Errors)
			circular := false
			for _, result := range results.Results {
				if result.RuleId == "resolving-references" {
					circular = true
				}
			}
			assert.Equal(t, tc.wantCircular, circular)
		})
	}
}
