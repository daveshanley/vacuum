package languageserver

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"go.yaml.in/yaml/v4"
)

func TestApplyAutoFix(t *testing.T) {
	lintRequest := &utils.LintFileRequest{
		SelectedRS: &rulesets.RuleSet{
			Rules: map[string]*model.Rule{
				"test-rule": {
					AutoFixFunction: "mockAutoFix",
				},
			},
		},
		AutoFixFunctions: map[string]model.AutoFixFunction{
			"mockAutoFix": func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
				return node, nil
			},
		},
	}

	server := &ServerState{
		lintRequest:   lintRequest,
		documentStore: newDocumentStore(),
	}

	doc := &Document{
		URI:     "file:///test.yaml",
		Content: "test: value",
	}

	testRange := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 0, Character: 4},
	}

	fixedText, _ := server.applyAutoFix(doc, "test-rule", testRange)

	assert.NotEmpty(t, fixedText, "Expected non-empty result from applyAutoFix")
}

func TestHasAutoFixForRule(t *testing.T) {
	lintRequest := &utils.LintFileRequest{
		SelectedRS: &rulesets.RuleSet{
			Rules: map[string]*model.Rule{
				"with-autofix": {AutoFixFunction: "testFunc"},
				"no-autofix":   {AutoFixFunction: ""},
			},
		},
		AutoFixFunctions: map[string]model.AutoFixFunction{
			"testFunc": func(*yaml.Node, *yaml.Node, *model.RuleFunctionContext) (*yaml.Node, error) {
				return nil, nil
			},
		},
	}

	server := &ServerState{lintRequest: lintRequest}

	assert.True(t, server.hasAutoFixForRule("with-autofix"), "Should have autofix for rule with autofix function")
	assert.False(t, server.hasAutoFixForRule("no-autofix"), "Should not have autofix for rule without autofix function")
}
