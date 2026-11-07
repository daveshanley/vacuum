package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/daveshanley/vacuum/motor"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"testing"

	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/pb33f/testify/require"
)

func TestBuiltinRepairCLIAndPreview(t *testing.T) {
	dir := t.TempDir()
	rules := focusedRepairRules(t, dir, "duplicated-entry-in-enum")
	spec := filepath.Join(dir, "api.json")
	original := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["cat","cat","dog"]}}}}`)
	require.NoError(t, os.WriteFile(spec, original, 0600))
	output := filepath.Join(dir, "fixed.json")
	preview, _, err := exportCommand(t, "lint", spec, "--ruleset", rules, "--fix", "--dry-run", "--fix-file", output, "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(preview)))
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, original, saved)
	require.NoFileExists(t, output)
	require.Contains(t, preview, "duplicated-entry-in-enum")
	_, _, err = exportCommand(t, "lint", spec, "--ruleset", rules, "--fix", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err, "noninteractive fixes retain the baseline report")
	saved, err = os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, original, saved)

}

func TestBuiltinRepairRespectsIgnoreFile(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.yaml")
	ignore := filepath.Join(dir, "ignore.yaml")
	original := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Pet: {type: string, enum: [cat, cat]}\n")
	require.NoError(t, os.WriteFile(spec, original, 0600))
	require.NoError(t, os.WriteFile(ignore, []byte("duplicated-entry-in-enum:\n  - $.components.schemas.Pet.enum\n"), 0600))
	_, _, err := exportCommand(t, "lint", spec, "--fix", "--dry-run", "--ignore-file", ignore, "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, original, saved)
}

func TestPolicyRepairCLI(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "api.yaml")
	policy := filepath.Join(dir, "policy.yaml")
	rules := filepath.Join(dir, "rules.yaml")
	require.NoError(t, os.WriteFile(rules, []byte("extends: [[vacuum:oas, off]]\nrules:\n  info-contact: true\n  contact-properties: true\n"), 0600))
	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"), 0600))
	require.NoError(t, os.WriteFile(policy, []byte("version: 1\npolicies:\n  - id: contact\n    rule: info-contact\n    diagnostic: contact-missing\n    scope: {source: api.yaml, pointer: /info}\n    value: {name: Team, email: api@example.com, url: https://example.com}\n"), 0600))
	_, warnings, err := exportCommand(t, "lint", spec, "--fix", "--dry-run", "--fix-policy", policy, "--ruleset", rules, "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.NotContains(t, string(saved), "api@example.com")
	require.Contains(t, warnings, "api@example.com", "the supplied policy value must appear in the verified preview")
	require.NotContains(t, warnings, "repairs declined")
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
		require.NoError(t, err)
		saved, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, original, saved)
	}
}

func TestBuiltinRepairSchemaCLI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schema.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string","enum":["a","a"]}`), 0600))
	_, _, err := exportCommand(t, "schema", path, "--fix", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err, "noninteractive fixes retain the baseline report")
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(saved, &doc))
	require.Len(t, doc["enum"], 2)
}

func TestRepairPreservesChangesModelFiltering(t *testing.T) {
	dir := t.TempDir()
	rules := focusedRepairRules(t, dir, "duplicated-entry-in-enum", "operation-description")
	spec := filepath.Join(dir, "api.yaml")
	changesPath := filepath.Join(dir, "changes.json")
	original := []byte("openapi: 3.1.0\n\ninfo: {title: Test, version: '1'}\npaths:\n  /pets:\n    get:\n      responses:\n        '200':\n          description: ok\ncomponents:\n  schemas:\n    Pet: {type: string, enum: [a,a]}\n")
	line := 9
	changes := wcModel.DocumentChanges{PropertyChanges: &wcModel.PropertyChanges{Changes: []*wcModel.Change{{ChangeType: wcModel.Modified, Context: &wcModel.ChangeContext{NewLine: &line}}}}}
	encoded, err := json.Marshal(changes)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(changesPath, encoded, 0600))
	require.NoError(t, os.WriteFile(spec, original, 0600))
	args := []string{"lint", spec, "--ruleset", rules, "--changes", changesPath, "--format", "json", "--fail-severity", "none"}
	baseline, _, err := exportCommand(t, args...)
	require.NoError(t, err)
	require.Contains(t, baseline, "operation-description", "changed response must retain the finding on its containing operation model")
	preview, _, err := exportCommand(t, append(append([]string(nil), args...), "--fix", "--dry-run")...)
	require.NoError(t, err)
	require.Contains(t, preview, "operation-description", "dry run reports the unchanged source")
	saved, err := os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, original, saved)
	_, _, err = exportCommand(t, append(append([]string(nil), args...), "--fix")...)
	require.NoError(t, err)
	saved, err = os.ReadFile(spec)
	require.NoError(t, err)
	require.Equal(t, original, saved)

}

func TestDryRunShowsVerifiedChangesWithoutWrites(t *testing.T) {
	dir := t.TempDir()
	rules := focusedRepairRules(t, dir, "duplicated-entry-in-enum", "component-description", "info-description")
	backups := t.TempDir()
	t.Setenv("TMPDIR", backups)
	path := filepath.Join(dir, "api.yaml")
	original := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Pet: {type: string, enum: [cat, cat]}\n")
	require.NoError(t, os.WriteFile(path, original, 0600))
	root := GetRootCommand()
	root.SetArgs([]string{"lint", path, "--ruleset", rules, "--fix", "--dry-run", "--no-style", "--fail-severity", "none", "--no-update-check"})
	var err error
	output, review := captureOSStreams(t, func() { err = root.Execute() })
	require.NoError(t, err)
	require.Contains(t, review, "Preview only. No files written.")
	require.Contains(t, review, "3 fixes;")
	require.Contains(t, output, "duplicated-entry-in-enum")
	require.Contains(t, review, "--- original")
	require.NotContains(t, output, "--- original")
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
	rules := focusedRepairRules(t, dir, "duplicated-entry-in-enum")
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
	args = append(args, "--ruleset", rules, "--fix", "--dry-run", "--format", "json", "--fail-severity", "none")
	output, _, err = exportCommand(t, args...)
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(output)))
	require.Contains(t, output, "duplicated-entry-in-enum")
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
	report, review, err := exportCommand(t, "lint", path, "--fix", "--dry-run", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(report)))
	require.Contains(t, review, "required: true")
	require.Contains(t, review, "enum: [a]")
	require.Contains(t, report, "duplicated-entry-in-enum")
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, original, saved)

}

func TestBuiltinRepairTagIndexShiftRetainsBaselineReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.yaml")
	original := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ntags:\n  - name: pets\n    description: Pets\n  - name: pets\n    description: Pets\n  - name: zoo\n")
	require.NoError(t, os.WriteFile(path, original, 0600))
	report, warnings, err := exportCommand(t, "lint", path, "--fix", "--dry-run", "--hard-mode", "--format", "json", "--fail-severity", "none")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(report)))
	require.Contains(t, report, "tag-description")
	require.Contains(t, warnings, "Review fixes:", "valid proposals remain available despite conflicting targets")
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, original, saved)

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
		if dry {
			require.Contains(t, warnings, "repairs declined")
		} else {
			require.Contains(t, warnings, "visible terminal review")
		}
		saved, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, original, saved)
	}
	report, _, err := exportCommand(t, "lint", path, "--fix", "--format", "json", "--fail-severity", "warn", "--ruleset", rules, "--functions", functions)
	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, ExitCodeViolations, exitErr.Code)
	require.True(t, json.Valid([]byte(report)))
	report, warnings, err := exportCommand(t, "lint", path, "--fix", "--fail-severity", "none", "--ruleset", rules, "--functions", functions, "--silent")
	require.NoError(t, err)
	require.Empty(t, report)
	require.Empty(t, warnings)
}

func focusedRepairRules(t *testing.T, directory string, ids ...string) string {
	t.Helper()
	data := "extends: [[vacuum:oas, off]]\nrules:\n"
	for _, id := range ids {
		data += "  " + id + ": true\n"
	}
	path := filepath.Join(directory, "repair-rules.yaml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0600))
	return path
}
