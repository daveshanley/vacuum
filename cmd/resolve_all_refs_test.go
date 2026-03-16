package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

const resolveAllRefsTestSpec = `openapi: "3.0.2"
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

func writeResolveAllRefsTestSpec(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	specPath := filepath.Join(tempDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(resolveAllRefsTestSpec), 0o600))
	return specPath
}

func writeResolveAllRefsRuleset(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	rulesetPath := filepath.Join(tempDir, "ruleset.yaml")
	ruleset := `extends: [[vacuum:oas, off]]
documentationUrl: https://example.com/ruleset
rules:
  response-has-content:
    description: Ensure referenced responses expose content
    severity: error
    given: $.paths[*][*].responses['404']
    then:
      field: content
      function: defined
`
	require.NoError(t, os.WriteFile(rulesetPath, []byte(ruleset), 0o600))
	return rulesetPath
}

func findSubcommand(t *testing.T, root *cobra.Command, name string) *cobra.Command {
	t.Helper()

	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}

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

type fakeLanguageServerRunner struct {
	run func() error
}

func (f *fakeLanguageServerRunner) Run() error {
	return f.run()
}

func TestReadLintFlags_ResolveAllRefsFromViper(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	root := GetRootCommand()
	lintCmd := findSubcommand(t, root, "lint")

	viper.Set("lint.resolve-all-refs", true)

	flags := ReadLintFlags(lintCmd)
	assert.True(t, flags.ResolveAllRefs)
}

func TestReadLintFlags_NestedRefsDocContextFromViper(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	root := GetRootCommand()
	lintCmd := findSubcommand(t, root, "lint")

	viper.Set("lint.nested-refs-doc-context", true)

	flags := ReadLintFlags(lintCmd)
	assert.True(t, flags.NestedRefsDocContext)
}

func TestProcessSingleFileOptimized_ResolveAllRefsFlag(t *testing.T) {
	specPath := writeResolveAllRefsTestSpec(t)

	unresolved := ProcessSingleFileOptimized(specPath, &FileProcessingConfig{
		Flags: &LintFlags{
			TimeoutFlag:       1,
			LookupTimeoutFlag: int((10 * time.Millisecond).Milliseconds()),
		},
		SelectedRuleset: buildResolveAllRefsRuleSet(),
	})

	require.NotNil(t, unresolved)
	assert.NoError(t, unresolved.Error)
	assert.Len(t, unresolved.Results, 1)

	resolved := ProcessSingleFileOptimized(specPath, &FileProcessingConfig{
		Flags: &LintFlags{
			ResolveAllRefs:    true,
			TimeoutFlag:       1,
			LookupTimeoutFlag: int((10 * time.Millisecond).Milliseconds()),
		},
		SelectedRuleset: buildResolveAllRefsRuleSet(),
	})

	require.NotNil(t, resolved)
	assert.NoError(t, resolved.Error)
	assert.Len(t, resolved.Results, 0)
}

func TestProcessSingleFileOptimized_NestedRefsDocContextFlag(t *testing.T) {
	specPath := writeResolveAllRefsTestSpec(t)

	withoutContext := ProcessSingleFileOptimized(specPath, &FileProcessingConfig{
		Flags: &LintFlags{
			ResolveAllRefs:    true,
			TimeoutFlag:       1,
			LookupTimeoutFlag: int((50 * time.Millisecond).Milliseconds()),
		},
		SelectedRuleset: &rulesets.RuleSet{
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
		CustomFunctions: map[string]model.RuleFunction{
			"nestedDocumentContextRecorder": &testNestedDocumentContextRecorder{},
		},
	})

	require.NotNil(t, withoutContext)
	assert.NoError(t, withoutContext.Error)
	assert.Len(t, withoutContext.Results, 1)

	withContext := ProcessSingleFileOptimized(specPath, &FileProcessingConfig{
		Flags: &LintFlags{
			ResolveAllRefs:       true,
			NestedRefsDocContext: true,
			TimeoutFlag:          1,
			LookupTimeoutFlag:    int((50 * time.Millisecond).Milliseconds()),
		},
		SelectedRuleset: &rulesets.RuleSet{
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
		CustomFunctions: map[string]model.RuleFunction{
			"nestedDocumentContextRecorder": &testNestedDocumentContextRecorder{},
		},
	})

	require.NotNil(t, withContext)
	assert.NoError(t, withContext.Error)
	assert.Len(t, withContext.Results, 0)
}

func TestDashboardCommand_ResolveRefFlags(t *testing.T) {
	specPath := writeResolveAllRefsTestSpec(t)
	rulesetPath := writeResolveAllRefsRuleset(t)

	cmd := GetDashboardCommand()
	registerPersistentFlags(cmd)
	cmd.Flags().Bool("silent", false, "Show nothing except the result")

	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{
		"--silent",
		"--ruleset", rulesetPath,
		"--resolve-all-refs",
		"--nested-refs-doc-context",
		specPath,
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestDashboardCommand_ResolveRefFlagsWithOriginal(t *testing.T) {
	specPath := writeResolveAllRefsTestSpec(t)
	rulesetPath := writeResolveAllRefsRuleset(t)

	cmd := GetDashboardCommand()
	registerPersistentFlags(cmd)
	cmd.Flags().Bool("silent", false, "Show nothing except the result")

	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{
		"--silent",
		"--ruleset", rulesetPath,
		"--original", specPath,
		"--resolve-all-refs",
		"--nested-refs-doc-context",
		specPath,
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestHTMLReportCommand_ResolveRefFlags(t *testing.T) {
	specPath := writeResolveAllRefsTestSpec(t)
	rulesetPath := writeResolveAllRefsRuleset(t)
	reportFile := filepath.Join(t.TempDir(), "resolve-all-refs.html")

	cmd := GetHTMLReportCommand()
	registerPersistentFlags(cmd)
	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{
		"--ruleset", rulesetPath,
		"--resolve-all-refs",
		"--nested-refs-doc-context",
		"-b",
		specPath,
		reportFile,
	})

	err := cmd.Execute()
	assert.NoError(t, err)
	_, statErr := os.Stat(reportFile)
	assert.NoError(t, statErr)
}

func TestLanguageServerCommand_ResolveRefFlags(t *testing.T) {
	cmd := GetLanguageServerCommand()
	registerPersistentFlags(cmd)

	originalRunLanguageServer := runLanguageServer
	defer func() {
		runLanguageServer = originalRunLanguageServer
	}()

	var capturedVersion string
	var capturedRequest *utils.LintFileRequest
	var capturedExecutionOptions *motor.ExecutionOptions
	runLanguageServer = func(version string, lintRequest *utils.LintFileRequest, executionOptions *motor.ExecutionOptions) error {
		capturedVersion = version
		capturedRequest = lintRequest
		capturedExecutionOptions = executionOptions
		return nil
	}

	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{
		"--base", "https://example.com/root/",
		"--remote=false",
		"--skip-check",
		"--timeout", "2",
		"--lookup-timeout", "123",
		"--resolve-all-refs",
		"--nested-refs-doc-context",
		"--ignore-array-circle-ref",
		"--ignore-polymorph-circle-ref",
		"--ext-refs",
	})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotNil(t, capturedRequest)
	assert.Equal(t, GetVersion(), capturedVersion)
	assert.Equal(t, "https://example.com/root/", capturedRequest.BaseFlag)
	assert.False(t, capturedRequest.Remote)
	assert.True(t, capturedRequest.SkipCheckFlag)
	assert.Equal(t, 2, capturedRequest.TimeoutFlag)
	assert.Equal(t, 123, capturedRequest.LookupTimeoutFlag)
	require.NotNil(t, capturedExecutionOptions)
	assert.True(t, capturedExecutionOptions.ResolveAllRefs)
	assert.True(t, capturedExecutionOptions.NestedRefsDocContext)
	assert.True(t, capturedRequest.IgnoreArrayCircleRef)
	assert.True(t, capturedRequest.IgnorePolymorphCircleRef)
	assert.True(t, capturedRequest.ExtensionRefs)
	assert.NotNil(t, capturedRequest.SelectedRS)
	assert.NotNil(t, capturedRequest.DefaultRuleSets)
	assert.NotNil(t, capturedRequest.Logger)
}

func TestRunLanguageServer_DefaultRunnerUsesExecutionOptions(t *testing.T) {
	originalRunner := newLanguageServerRunner
	defer func() {
		newLanguageServerRunner = originalRunner
	}()

	lintRequest := &utils.LintFileRequest{}
	executionOptions := &motor.ExecutionOptions{
		ResolveAllRefs:       true,
		NestedRefsDocContext: true,
	}

	var called bool
	newLanguageServerRunner = func(version string, lintRequestArg *utils.LintFileRequest, executionOptionsArg *motor.ExecutionOptions) languageServerRunner {
		assert.Equal(t, "v1.2.3", version)
		assert.Same(t, lintRequest, lintRequestArg)
		assert.Same(t, executionOptions, executionOptionsArg)
		return &fakeLanguageServerRunner{
			run: func() error {
				called = true
				return nil
			},
		}
	}

	err := runLanguageServer("v1.2.3", lintRequest, executionOptions)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestNewLanguageServerRunner_DefaultConstructor(t *testing.T) {
	runner := newLanguageServerRunner("v1.2.3", &utils.LintFileRequest{}, &motor.ExecutionOptions{
		ResolveAllRefs: true,
	})

	assert.NotNil(t, runner)
}

func TestLanguageServerCommand_InvalidIgnoreFile(t *testing.T) {
	cmd := GetLanguageServerCommand()
	registerPersistentFlags(cmd)

	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{
		"--ignore-file", filepath.Join(t.TempDir(), "missing-ignore.yaml"),
	})

	err := cmd.Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read ignore file")
}
