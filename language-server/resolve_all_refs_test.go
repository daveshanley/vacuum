package languageserver

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"go.yaml.in/yaml/v4"
)

const resolveAllRefsServerTestSpec = `openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        '404':
          $ref: '#/components/responses/NotFound'
components:
  responses:
    NotFound:
      description: Not Found
      content:
        application/json:
          schema:
            type: object
`

func buildResolveAllRefsRuleSet() *rulesets.RuleSet {
	return &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"response-has-content": {
				Id:           "response-has-content",
				Description:  "Ensure referenced responses expose content",
				Resolved:     false,
				Given:        "$.paths[*][*].responses['404']",
				RuleCategory: model.RuleCategories[model.CategoryValidation],
				Type:         rulesets.Validation,
				Severity:     model.SeverityError,
				Then: model.RuleAction{
					Field:    "content",
					Function: "defined",
				},
			},
		},
	}
}

type testNestedDocumentContextRecorder struct{}

func (r *testNestedDocumentContextRecorder) GetCategory() string {
	return model.CategoryValidation
}

func (r *testNestedDocumentContextRecorder) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	documentContextEnabled := context.Document != nil &&
		context.Document.GetConfiguration() != nil &&
		context.Document.GetConfiguration().ResolveNestedRefsWithDocumentContext
	indexContextEnabled := context.Index != nil &&
		context.Index.GetConfig() != nil &&
		context.Index.GetConfig().ResolveNestedRefsWithDocumentContext

	if documentContextEnabled && indexContextEnabled {
		return nil
	}

	result := model.BuildFunctionResultString("nested document context not enabled")
	result.Rule = context.Rule
	return []model.RuleFunctionResult{result}
}

func (r *testNestedDocumentContextRecorder) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "nestedDocumentContextRecorder",
	}
}

func TestNewServer_DefaultsExecutionOptionsToNil(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	state := NewServer("v1.2.3", &utils.LintFileRequest{Logger: logger})

	assert.NotNil(t, state)
	assert.Nil(t, state.executionOptions)
	assert.Same(t, logger, state.logger)
}

func TestNewServerWithExecutionOptions_InitializesState(t *testing.T) {
	executionOptions := &motor.ExecutionOptions{
		ResolveAllRefs:       true,
		NestedRefsDocContext: true,
	}

	state := NewServerWithExecutionOptions("v1.2.3", &utils.LintFileRequest{
		BaseFlag:                 "https://example.com/root/",
		Remote:                   true,
		SkipCheckFlag:            true,
		TimeoutFlag:              2,
		IgnoreArrayCircleRef:     true,
		IgnorePolymorphCircleRef: true,
		ExtensionRefs:            true,
	}, executionOptions)

	assert.NotNil(t, state)
	assert.NotNil(t, state.server)
	assert.NotNil(t, state.documentStore)
	assert.NotNil(t, state.logger)
	assert.Same(t, executionOptions, state.executionOptions)
	if assert.NotNil(t, state.baseConfig) {
		assert.Equal(t, "https://example.com/root/", state.baseConfig.Base)
		if assert.NotNil(t, state.baseConfig.Remote) {
			assert.True(t, *state.baseConfig.Remote)
		}
		if assert.NotNil(t, state.baseConfig.SkipCheck) {
			assert.True(t, *state.baseConfig.SkipCheck)
		}
		if assert.NotNil(t, state.baseConfig.Timeout) {
			assert.Equal(t, 2, *state.baseConfig.Timeout)
		}
		if assert.NotNil(t, state.baseConfig.IgnoreArrayCircleRef) {
			assert.True(t, *state.baseConfig.IgnoreArrayCircleRef)
		}
		if assert.NotNil(t, state.baseConfig.IgnorePolymorphCircleRef) {
			assert.True(t, *state.baseConfig.IgnorePolymorphCircleRef)
		}
		if assert.NotNil(t, state.baseConfig.ExtensionRefs) {
			assert.True(t, *state.baseConfig.ExtensionRefs)
		}
	}
}

func TestServerState_RunDiagnostic_ResolveAllRefs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name           string
		resolveAllRefs bool
		wantCount      int
	}{
		{
			name:           "disabled",
			resolveAllRefs: false,
			wantCount:      1,
		},
		{
			name:           "enabled",
			resolveAllRefs: true,
			wantCount:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &ServerState{
				lintRequest: &utils.LintFileRequest{
					SelectedRS:        buildResolveAllRefsRuleSet(),
					TimeoutFlag:       1,
					LookupTimeoutFlag: 50,
					Logger:            logger,
				},
				executionOptions: &motor.ExecutionOptions{ResolveAllRefs: tt.resolveAllRefs},
			}
			doc := &Document{
				URI:     "file:///tmp/spec.yaml",
				Content: resolveAllRefsServerTestSpec,
			}

			notifyCh := make(chan protocol.PublishDiagnosticsParams, 1)
			notify := glsp.NotifyFunc(func(method string, params any) {
				if method == protocol.ServerTextDocumentPublishDiagnostics {
					notifyCh <- params.(protocol.PublishDiagnosticsParams)
				}
			})

			state.runDiagnostic(doc, notify)

			select {
			case published := <-notifyCh:
				assert.Equal(t, doc.URI, published.URI)
				assert.Len(t, published.Diagnostics, tt.wantCount)
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for diagnostics notification")
			}
		})
	}
}

func TestServerState_RunDiagnostic_NestedRefsDocContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name                                 string
		resolveNestedRefsWithDocumentContext bool
		wantCount                            int
	}{
		{
			name:                                 "disabled",
			resolveNestedRefsWithDocumentContext: false,
			wantCount:                            1,
		},
		{
			name:                                 "enabled",
			resolveNestedRefsWithDocumentContext: true,
			wantCount:                            0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &ServerState{
				lintRequest: &utils.LintFileRequest{
					SelectedRS: &rulesets.RuleSet{
						Rules: map[string]*model.Rule{
							"nested-document-context": {
								Id:           "nested-document-context",
								Resolved:     false,
								Given:        "$",
								RuleCategory: model.RuleCategories[model.CategoryValidation],
								Type:         rulesets.Validation,
								Severity:     model.SeverityError,
								Then: model.RuleAction{
									Function: "nestedDocumentContextRecorder",
								},
							},
						},
					},
					TimeoutFlag:       1,
					LookupTimeoutFlag: 50,
					Functions: map[string]model.RuleFunction{
						"nestedDocumentContextRecorder": &testNestedDocumentContextRecorder{},
					},
					Logger: logger,
				},
				executionOptions: &motor.ExecutionOptions{
					ResolveAllRefs:       true,
					NestedRefsDocContext: tt.resolveNestedRefsWithDocumentContext,
				},
			}
			doc := &Document{
				URI:     "file:///tmp/spec.yaml",
				Content: resolveAllRefsServerTestSpec,
			}

			notifyCh := make(chan protocol.PublishDiagnosticsParams, 1)
			notify := glsp.NotifyFunc(func(method string, params any) {
				if method == protocol.ServerTextDocumentPublishDiagnostics {
					notifyCh <- params.(protocol.PublishDiagnosticsParams)
				}
			})

			state.runDiagnostic(doc, notify)

			select {
			case published := <-notifyCh:
				assert.Equal(t, doc.URI, published.URI)
				assert.Len(t, published.Diagnostics, tt.wantCount)
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for diagnostics notification")
			}
		})
	}
}
