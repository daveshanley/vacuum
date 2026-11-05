package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/daveshanley/vacuum/motor"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"testing"

	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/pb33f/testify/require"
)

func TestBuiltinRepairCLIAndPreview(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.json")
	original := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["cat","cat","dog"]}}}}`)
	require.NoError(t, os.WriteFile(spec, original, 0600))
	output := filepath.Join(dir, "fixed.json")
	preview, _, err := exportCommand(t, "lint", spec, "--fix", "--dry-run", "--fix-file", output, "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(preview)))
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, original, saved)
	require.NoFileExists(t, output)
	require.NotContains(t, preview, "duplicated-entry-in-enum")
	_, _, err = exportCommand(t, "lint", spec, "--fix", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err = os.ReadFile(spec)
	require.NoError(t, err)
	require.True(t, json.Valid(saved))
	require.NotEqual(t, original, saved)
	var document map[string]any
	require.NoError(t, json.Unmarshal(saved, &document))
	enum := document["components"].(map[string]any)["schemas"].(map[string]any)["Pet"].(map[string]any)["enum"].([]any)
	require.Equal(t, []any{"cat", "dog"}, enum)
	_, _, err = exportCommand(t, "lint", spec, "--fix", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	again, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, saved, again)
}

func TestBuiltinRepairRespectsIgnoreFile(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.yaml")
	ignore := filepath.Join(dir, "ignore.yaml")
	original := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Pet: {type: string, enum: [cat, cat]}\n")
	require.NoError(t, os.WriteFile(spec, original, 0600))
	require.NoError(t, os.WriteFile(ignore, []byte("duplicated-entry-in-enum:\n  - $.components.schemas.Pet.enum\n"), 0600))
	_, _, err := exportCommand(t, "lint", spec, "--fix", "--ignore-file", ignore, "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, original, saved)
}

func TestPolicyRepairCLI(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.yaml")
	policy := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"), 0600))
	require.NoError(t, os.WriteFile(policy, []byte("version: 1\npolicies:\n  - id: contact\n    rule: info-contact\n    diagnostic: contact-missing\n    scope: {source: api.yaml, pointer: /info}\n    value: {name: Team, email: api@example.com, url: https://example.com}\n"), 0600))
	_, _, err := exportCommand(t, "lint", spec, "--fix", "--fix-policy", policy, "--hard-mode", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Contains(t, string(saved), "api@example.com")
}

func TestBuiltinRepairRejectsDocumentStream(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stream.yaml")
	original := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Pet: {enum: [a, a]}\n---\nx-keep: true\n")
	require.NoError(t, os.WriteFile(path, original, 0600))
	for _, dry := range []bool{false, true} {
		args := []string{"lint", path, "--fix", "--format", "json", "--fail-severity", "none"}
		if dry {
			args = append(args, "--dry-run")
		}
		_, _, err := exportCommand(t, args...)
		require.Error(t, err)
		saved, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, original, saved)
	}
}

func TestBuiltinRepairSchemaCLI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schema.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string","enum":["a","a"]}`), 0600))
	_, _, err := exportCommand(t, "schema", path, "--fix", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(saved, &doc))
	require.Len(t, doc["enum"], 1)
}

func TestRepairPreservesChangesModelFiltering(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.yaml")
	changesPath := filepath.Join(dir, "changes.json")
	original := []byte("openapi: 3.1.0\n\ninfo: {title: Test, version: '1'}\npaths:\n  /pets:\n    get:\n      responses:\n        '200':\n          description: ok\ncomponents:\n  schemas:\n    Pet: {type: string, enum: [a,a]}\n")
	line := 9
	changes := wcModel.DocumentChanges{PropertyChanges: &wcModel.PropertyChanges{Changes: []*wcModel.Change{{ChangeType: wcModel.Modified, Context: &wcModel.ChangeContext{NewLine: &line}}}}}
	encoded, err := json.Marshal(changes)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(changesPath, encoded, 0600))
	require.NoError(t, os.WriteFile(spec, original, 0600))
	args := []string{"lint", spec, "--changes", changesPath, "--format", "json", "--fail-severity", "none"}
	baseline, _, err := exportCommand(t, args...)
	require.NoError(t, err)
	require.Contains(t, baseline, "operation-description", "changed response must retain the finding on its containing operation model")
	preview, _, err := exportCommand(t, append(append([]string(nil), args...), "--fix", "--dry-run")...)
	require.NoError(t, err)
	require.Contains(t, preview, "operation-description")
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, original, saved)
	applied, _, err := exportCommand(t, append(append([]string(nil), args...), "--fix")...)
	require.NoError(t, err)
	require.Contains(t, applied, "operation-description", "verified output must retain the changed parent model after lines move")
	saved, err = os.ReadFile(spec)
	require.NoError(t, err)
	require.NotEqual(t, original, saved)
}

func TestDryRunShowsVerifiedChangesWithoutWrites(t *testing.T) {
	dir := t.TempDir()
	backups := t.TempDir()
	t.Setenv("TMPDIR", backups)
	path := filepath.Join(dir, "api.yaml")
	original := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Pet: {type: string, enum: [cat, cat]}\n")
	require.NoError(t, os.WriteFile(path, original, 0600))
	root := GetRootCommand()
	root.SetArgs([]string{"lint", path, "--fix", "--dry-run", "--no-style", "--fail-severity", "none", "--no-update-check"})
	var err error
	output, _ := captureOSStreams(t, func() { err = root.Execute() })
	require.NoError(t, err)
	require.Contains(t, output, "PREVIEW, no files written")
	require.Contains(t, output, "1 verified fixes")
	require.Contains(t, output, "duplicated-entry-in-enum")
	require.Contains(t, output, "--- original")
	require.NotContains(t, output, "1 fixes applied")
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, original, saved)
	entries, err := os.ReadDir(backups)
	require.NoError(t, err)
	require.Empty(t, entries, "preview must not create backups")
}

func TestDryRunSchemaAndMultipleFilesLeaveSourcesUnchanged(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join(dir, "schema.json")
	originalSchema := []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string","enum":["a","a"]}`)
	require.NoError(t, os.WriteFile(schema, originalSchema, 0600))
	output, _, err := exportCommand(t, "schema", "lint", schema, "--fix", "--dry-run", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(output)))
	saved, err := os.ReadFile(schema)
	require.NoError(t, err)
	require.Equal(t, originalSchema, saved)
	original := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["a","a"]}}}}`)
	paths := []string{filepath.Join(dir, "one.json"), filepath.Join(dir, "two.json")}
	for _, p := range paths {
		require.NoError(t, os.WriteFile(p, original, 0600))
	}
	args := append([]string{"lint"}, paths...)
	args = append(args, "--fix", "--dry-run", "--format", "json", "--fail-severity", "none")
	output, _, err = exportCommand(t, args...)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(output)))
	require.NotContains(t, output, "duplicated-entry-in-enum")
	for _, p := range paths {
		saved, err := os.ReadFile(p)
		require.NoError(t, err)
		require.Equal(t, original, saved)
	}
}

func TestRepairFlagsHideInternalMechanism(t *testing.T) {
	for _, command := range []*cobra.Command{GetLintCommand(), GetSchemaCommand()} {
		require.Nil(t, command.Flags().Lookup("fix-builtins"))
		require.Nil(t, command.Flags().Lookup("fix-plan"))
		require.NotNil(t, command.Flags().Lookup("fix"))
		require.NotNil(t, command.Flags().Lookup("dry-run"))
	}
	path := filepath.Join(t.TempDir(), "api.yaml")
	require.NoError(t, os.WriteFile(path, []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"), 0600))
	_, _, err := exportCommand(t, "lint", path, "--dry-run", "--format", "json")
	require.ErrorContains(t, err, "require --fix")
}

func TestBuiltinRepairIndependentParameterFixesInOnePass(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.yaml")
	original := []byte(`openapi: 3.1.0
info:
  title: Test
  version: '1'
paths:
  /pets/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: false
          schema:
            type: string
        - name: status
          in: query
          schema:
            type: string
            enum: [a, a]
      responses:
        '200':
          description: OK
`)
	require.NoError(t, os.WriteFile(path, original, 0600))
	report, _, err := exportCommand(t, "lint", path, "--fix", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(report)))
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(saved), "required: true")
	require.Contains(t, string(saved), "enum: [a]")
	require.NotContains(t, report, "duplicated-entry-in-enum")
	_, _, err = exportCommand(t, "lint", path, "--fix", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	again, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, saved, again)
}

func TestBuiltinRepairTagIndexShiftRetainsBaselineReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.yaml")
	original := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ntags:\n  - name: pets\n    description: Pets\n  - name: pets\n    description: Pets\n  - name: zoo\n")
	require.NoError(t, os.WriteFile(path, original, 0600))
	report, _, err := exportCommand(t, "lint", path, "--fix", "--hard-mode", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(report)))
	require.Contains(t, report, "tag-description")
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(saved), "name: pets"))
	var findings []struct {
		Code string `json:"code"`
		Path []any  `json:"path"`
	}
	require.NoError(t, json.Unmarshal([]byte(report), &findings))
	found := false
	for _, finding := range findings {
		if finding.Code == "tag-description" {
			require.Equal(t, []any{"tags", float64(1)}, finding.Path)
			found = true
		}
	}
	require.True(t, found)
}

func TestRepairWarningsUseErrorStream(t *testing.T) {
	var warnings bytes.Buffer
	result := &motor.RuleSetExecutionResult{RepairWarnings: []string{"repairs declined: new diagnostic"}}
	flags := &LintFlags{FormattedOutput: true, RepairWarningWriter: &warnings}
	writeRepairWarnings(result, flags)
	require.Equal(t, "Warning: repairs declined: new diagnostic\n", warnings.String())
	warnings.Reset()
	flags.repairWarningsSilenced = true
	writeRepairWarnings(result, flags)
	require.Empty(t, warnings.String())
}

func TestBuiltinRepairDeclineKeepsCLIReportAndSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.json")
	rules := filepath.Join(dir, "rules.yaml")
	functions := filepath.Join(dir, "functions")
	require.NoError(t, os.Mkdir(functions, 0700))
	original := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["a","a","b"]}}}}`)
	require.NoError(t, os.WriteFile(path, original, 0600))
	require.NoError(t, os.WriteFile(rules, []byte(`extends: [[vacuum:oas, off]]
rules:
  duplicated-entry-in-enum: true
  require-three:
    severity: error
    recommended: true
    given: $.components.schemas.Pet.enum
    then:
      function: requireThree
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(functions, "requireThree.js"), []byte(`function getSchema() { return {name: "requireThree"}; }
function runRule(input) { return input.length === 2 ? [{message: "new diagnostic"}] : []; }`), 0600))
	for _, dry := range []bool{false, true} {
		args := []string{"lint", path, "--fix", "--format", "json", "--fail-severity", "none", "--ruleset", rules, "--functions", functions}
		if dry {
			args = append(args, "--dry-run")
		}
		report, warnings, err := exportCommand(t, args...)
		require.NoError(t, err)
		require.True(t, json.Valid([]byte(report)))
		require.Contains(t, report, "duplicated-entry-in-enum")
		require.NotContains(t, report, "new diagnostic")
		require.Contains(t, warnings, "repairs declined")
		saved, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, original, saved)
	}
}
