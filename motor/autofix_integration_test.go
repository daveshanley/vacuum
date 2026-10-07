package motor

import (
	"bytes"
	"io"
	"log/slog"
	"testing"

	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
)

func TestAutoFixIntegration(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  description: ""
`

	emptyDescriptionFix := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		if node.Value == "" {
			node.Value = "TODO: Add description"
		}
		return node, nil
	}

	customRule := model.Rule{
		Id:              "empty-description-autofix",
		Message:         "Empty description found",
		Given:           "$.info.description",
		Severity:        model.SeverityWarn,
		AutoFixFunction: "fixEmptyDescription",
		Then: &model.RuleAction{
			Function: "truthy",
		},
	}

	execution := &RuleSetExecution{
		RuleSet:          &rulesets.RuleSet{Rules: map[string]*model.Rule{"empty-description-autofix": &customRule}},
		Spec:             []byte(spec),
		SpecFileName:     "test.yaml",
		ApplyAutoFixes:   true,
		AutoFixFunctions: map[string]model.AutoFixFunction{"fixEmptyDescription": emptyDescriptionFix},
	}

	result := ApplyRulesToRuleSet(execution)

	assert.Equal(t, len(result.FixedResults), 1, "Should have fixed one violation")

	for _, r := range result.FixedResults {
		assert.True(t, r.AutoFixed)
		assert.Equal(t, "empty-description-autofix", r.RuleId)
	}
}

func TestAutoFixDisabled(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  description: ""
`

	customRule := model.Rule{
		Id:       "empty-description",
		Message:  "Empty description found",
		Given:    "$.info.description",
		Severity: model.SeverityWarn,
		Then:     &model.RuleAction{Function: "truthy"},
	}

	execution := &RuleSetExecution{
		RuleSet:        &rulesets.RuleSet{Rules: map[string]*model.Rule{"empty-description": &customRule}},
		Spec:           []byte(spec),
		ApplyAutoFixes: false,
	}

	result := ApplyRulesToRuleSet(execution)
	assert.Greater(t, len(result.Results), 0)
	assert.Equal(t, 0, len(result.FixedResults))
}

func TestAutoFixDoesNotAffectNonFixableViolations(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  description: ""
paths:
  /test:
    get:
      summary: ""
`

	// Fixable rule
	fixableRule := model.Rule{
		Id:              "empty-description-fixable",
		Message:         "Empty description found",
		Given:           "$.info.description",
		Severity:        model.SeverityWarn,
		AutoFixFunction: "fixEmptyDescription",
		Then:            &model.RuleAction{Function: "truthy"},
	}

	// Non-fixable rule (no AutoFixFunction)
	nonFixableRule := model.Rule{
		Id:       "empty-summary-not-fixable",
		Message:  "Empty summary found",
		Given:    "$.paths..summary",
		Severity: model.SeverityError,
		Then:     &model.RuleAction{Function: "truthy"},
	}

	emptyDescriptionFix := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		if node.Value == "" {
			node.Value = "TODO: Add description"
		}
		return node, nil
	}

	execution := &RuleSetExecution{
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{
			"empty-description-fixable": &fixableRule,
			"empty-summary-not-fixable": &nonFixableRule,
		}},
		Spec:             []byte(spec),
		SpecFileName:     "test.yaml",
		ApplyAutoFixes:   true,
		AutoFixFunctions: map[string]model.AutoFixFunction{"fixEmptyDescription": emptyDescriptionFix},
	}

	result := ApplyRulesToRuleSet(execution)

	assert.Equal(t, len(result.FixedResults), 1, "Should have fixed one violation")
	assert.Equal(t, len(result.Results), 1, "Should have one unfixed violation")

	for _, r := range result.FixedResults {
		assert.True(t, r.AutoFixed)
		assert.Equal(t, "empty-description-fixable", r.RuleId)
	}

	for _, r := range result.Results {
		assert.False(t, r.AutoFixed)
		assert.Equal(t, "empty-summary-not-fixable", r.RuleId)
	}
}

type testAutoFixUnmappedRule struct{}

func (r *testAutoFixUnmappedRule) GetCategory() string {
	return model.CategoryValidation
}

func (r *testAutoFixUnmappedRule) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	return []model.RuleFunctionResult{
		{
			Message:   "unmapped node",
			StartNode: &yaml.Node{Kind: yaml.ScalarNode, Value: ""},
			Path:      "$.info.description",
		},
	}
}

func (r *testAutoFixUnmappedRule) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "autofixUnmapped",
	}
}

func TestAutoFixResolvedRuleUpdatesCanonicalDocument(t *testing.T) {
	spec := `
openapi: 3.0.2
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '404':
          $ref: '#/components/responses/NotFound'
components:
  responses:
    NotFound:
      description: ""
`

	fixedValue := "AUTO-FIXED"
	autoFixCalled := false
	fixDescription := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		autoFixCalled = true
		if node.Value == "" {
			node.Value = fixedValue
		}
		return node, nil
	}

	customRule := model.Rule{
		Id:              "fix-notfound-description",
		Message:         "Empty description found",
		Given:           "$.paths[*][*].responses['404'].description",
		Resolved:        true,
		Severity:        model.SeverityWarn,
		AutoFixFunction: "fixDescription",
		Then: &model.RuleAction{
			Function: "truthy",
		},
	}

	execution := &RuleSetExecution{
		RuleSet:          &rulesets.RuleSet{Rules: map[string]*model.Rule{"fix-notfound-description": &customRule}},
		Spec:             []byte(spec),
		ApplyAutoFixes:   true,
		AutoFixFunctions: map[string]model.AutoFixFunction{"fixDescription": fixDescription},
		SilenceLogs:      true,
	}

	result := ApplyRulesToRuleSet(execution)

	assert.True(t, autoFixCalled)
	assert.Equal(t, 1, len(result.FixedResults), "Should have fixed one violation")
	assert.True(t, result.FixedResults[0].AutoFixed)
	assert.Contains(t, string(result.ModifiedSpec), fixedValue)
}

func TestAutoFixResolvedRuleSkipsWhenUnmapped(t *testing.T) {
	spec := `
openapi: 3.0.2
info:
  title: Test API
  version: 1.0.0
`

	autoFixCalled := false
	fixDescription := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		autoFixCalled = true
		return node, nil
	}

	customRule := model.Rule{
		Id:              "unmapped-autofix",
		Message:         "Unmapped node",
		Given:           "$",
		Resolved:        true,
		Severity:        model.SeverityWarn,
		AutoFixFunction: "fixDescription",
		Then: &model.RuleAction{
			Function: "autofixUnmapped",
		},
	}

	execution := &RuleSetExecution{
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"unmapped-autofix": &customRule}},
		Spec:    []byte(spec),
		CustomFunctions: map[string]model.RuleFunction{
			"autofixUnmapped": &testAutoFixUnmappedRule{},
		},
		ApplyAutoFixes:   true,
		AutoFixFunctions: map[string]model.AutoFixFunction{"fixDescription": fixDescription},
		SilenceLogs:      true,
	}

	result := ApplyRulesToRuleSet(execution)

	assert.False(t, autoFixCalled)
	assert.Equal(t, 0, len(result.FixedResults))
	assert.Equal(t, 1, len(result.Results))
	assert.False(t, result.Results[0].AutoFixed)
}

func TestAutoFixResolvedRuleSkipsWithoutUnresolvedIndex(t *testing.T) {
	autoFixCalled := false
	fixDescription := func(node *yaml.Node, document *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
		autoFixCalled = true
		return node, nil
	}

	rule := &model.Rule{
		Id:              "no-index-autofix",
		Given:           "$",
		Then:            model.RuleAction{Function: "autofixUnmapped"},
		Resolved:        true,
		AutoFixFunction: "fixDescription",
	}

	var ruleResults []model.RuleFunctionResult
	var fixedResults []model.RuleFunctionResult
	ctx := ruleContext{
		rule:               rule,
		specNode:           &yaml.Node{Kind: yaml.MappingNode},
		applyAutoFixes:     true,
		skipDocumentCheck:  true,
		customFunctions:    map[string]model.RuleFunction{"autofixUnmapped": &testAutoFixUnmappedRule{}},
		builtinFunctions:   functions.MapBuiltinFunctions(),
		specNodeUnresolved: &yaml.Node{},
		autoFixFunctions:   map[string]model.AutoFixFunction{"fixDescription": fixDescription},
		ruleResults:        &ruleResults,
		fixedResults:       &fixedResults,
		silenceLogs:        true,
		logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		indexUnresolved:    nil,
	}

	runRule(ctx)

	assert.False(t, autoFixCalled)
	assert.Equal(t, 0, len(fixedResults))
	assert.Equal(t, 1, len(ruleResults))
}

func TestAutoFixSkippedDiagnostics(t *testing.T) {
	for _, missingIndex := range []bool{false, true} {
		for _, silent := range []bool{false, true} {
			var logs bytes.Buffer
			root := &yaml.Node{}
			assert.NoError(t, yaml.Unmarshal([]byte(executionOptionsOpenAPISpec), root))
			var unresolvedIndex *index.SpecIndex
			message := "Auto-fix skipped: unresolved index not available"
			if !missingIndex {
				unresolvedIndex = index.NewSpecIndex(root)
				message = "Auto-fix skipped: unable to map resolved node to canonical document"
			}
			var findings, fixed []model.RuleFunctionResult
			ctx := ruleContext{
				rule: &model.Rule{
					Id: "skip", Given: "$", Resolved: true, AutoFixFunction: "fix",
					Then: model.RuleAction{Function: "autofixUnmapped"},
				},
				specNode:           root,
				specNodeUnresolved: root,
				indexUnresolved:    unresolvedIndex,
				applyAutoFixes:     true,
				skipDocumentCheck:  true,
				builtinFunctions:   functions.MapBuiltinFunctions(),
				customFunctions:    map[string]model.RuleFunction{"autofixUnmapped": &testAutoFixUnmappedRule{}},
				autoFixFunctions: map[string]model.AutoFixFunction{"fix": func(node, _ *yaml.Node, _ *model.RuleFunctionContext) (*yaml.Node, error) {
					t.Fatal("unmapped node must not reach auto-fix callback")
					return node, nil
				}},
				ruleResults:  &findings,
				fixedResults: &fixed,
				logger:       slog.New(slog.NewTextHandler(&logs, nil)),
				silenceLogs:  silent,
			}
			runRule(ctx)
			assert.Len(t, findings, 1)
			assert.Empty(t, fixed)
			if silent {
				assert.Empty(t, logs.String())
			} else {
				assert.Contains(t, logs.String(), message)
			}
		}
	}
}
