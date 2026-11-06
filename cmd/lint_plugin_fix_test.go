package cmd

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestFileProcessorDeclinesUnreviewedRegisteredRepair(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.yaml")
	require.NoError(t, os.WriteFile(path, []byte("openapi: 3.0.3\ninfo: {title: Bad, version: '1'}\npaths: {}\n"), 0600))
	rs, err := rulesets.CreateRuleSetFromData([]byte("rules:\n  title:\n    given: $.info.title\n    severity: error\n    autoFixFunction: clearTitle\n    then: {function: falsy}\n"))
	require.NoError(t, err)
	calls := 0
	result := ProcessSingleFileOptimized(path, &FileProcessingConfig{
		Flags:           &LintFlags{FixFlag: true, TimeoutFlag: 5, LookupTimeoutFlag: 500, SilentFlag: true},
		SelectedRuleset: rs,
		AutoFixFunctions: map[string]model.AutoFixFunction{"clearTitle": func(n, doc *yaml.Node, ctx *model.RuleFunctionContext) (*yaml.Node, error) {
			calls++
			n.Value = ""
			return n, nil
		}},
	})
	require.ErrorContains(t, result.Error, "custom fix callbacks need a reviewed proposal")
	require.Equal(t, 0, calls)
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(saved), "Bad")
}

func TestMultiFileRepairRejectsSharedOutput(t *testing.T) {
	command := GetLintCommand()
	require.NoError(t, command.Flags().Set("fix", "true"))
	require.NoError(t, command.Flags().Set("fix-file", "shared.yaml"))
	err := runMultipleFiles(command, []string{"one.yaml", "two.yaml"})
	require.ErrorContains(t, err, "requires a single input")
}

func TestRepairBackupsPreserveIdenticalBasenames(t *testing.T) {
	backups := t.TempDir()
	t.Setenv("TMPDIR", backups)
	root := t.TempDir()
	for _, name := range []string{"one", "two"} {
		dir := filepath.Join(root, name)
		require.NoError(t, os.Mkdir(dir, 0700))
		path := filepath.Join(dir, "spec.yaml")
		require.NoError(t, os.WriteFile(path, []byte(name), 0600))
		require.NoError(t, writeFixedFile(&motor.RuleSetExecutionResult{RuleSetExecution: &motor.RuleSetExecution{Spec: []byte(name)}, ModifiedSpec: []byte("fixed")}, path, ""))
	}
	files, err := filepath.Glob(filepath.Join(backups, "vacuum-spec.yaml-*.bak"))
	require.NoError(t, err)
	require.Len(t, files, 2)
	contents := []string{}
	for _, path := range files {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		contents = append(contents, string(data))
	}
	require.ElementsMatch(t, []string{"one", "two"}, contents)
}

func TestRepairWriteRejectsStaleSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"title":"new"}`), 0600))
	result := &motor.RuleSetExecutionResult{RuleSetExecution: &motor.RuleSetExecution{Spec: []byte(`{"title":"old"}`)}, ModifiedSpec: []byte(`{"title":"fixed"}`)}
	require.ErrorContains(t, writeFixedFile(result, path, ""), "changed since lint")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, `{"title":"new"}`, string(data))
}

func TestRepairWritePreservesModeAndRejectsAliases(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.yaml")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0640))
	alias := filepath.Join(dir, "alias.yaml")
	require.NoError(t, os.Symlink(path, alias))
	result := &motor.RuleSetExecutionResult{RuleSetExecution: &motor.RuleSetExecution{Spec: []byte("old")}, ModifiedSpec: []byte("fixed")}
	require.ErrorContains(t, writeFixedFile(result, alias, ""), "regular file")
	require.ErrorContains(t, writeFixedFile(result, path, alias), "regular file")
	require.NoError(t, writeFixedFile(result, path, ""))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0640), info.Mode().Perm())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "fixed", string(data))
	staged, err := filepath.Glob(filepath.Join(dir, ".vacuum-repair-*"))
	require.NoError(t, err)
	require.Empty(t, staged)
}

func TestFileProcessorDryRunVerifiesRegisteredRepairWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.yaml")
	original := []byte("openapi: 3.0.3\ninfo: {title: Bad, version: '1'}\npaths: {}\n")
	require.NoError(t, os.WriteFile(path, original, 0600))
	rs, err := rulesets.CreateRuleSetFromData([]byte("rules:\n  title:\n    given: $.info.title\n    severity: error\n    autoFixFunction: clearTitle\n    then: {function: falsy}\n"))
	require.NoError(t, err)
	calls := 0
	result := ProcessSingleFileOptimized(path, &FileProcessingConfig{
		Flags:           &LintFlags{FixFlag: true, DryRun: true, TimeoutFlag: 5, LookupTimeoutFlag: 500, SilentFlag: true},
		SelectedRuleset: rs,
		AutoFixFunctions: map[string]model.AutoFixFunction{"clearTitle": func(n, doc *yaml.Node, ctx *model.RuleFunctionContext) (*yaml.Node, error) {
			calls++
			n.Value = ""
			return n, nil
		}},
	})
	require.ErrorContains(t, result.Error, "custom fix callbacks need a reviewed proposal")
	require.Equal(t, 0, calls)
	require.Empty(t, result.FixedResults)
	require.Empty(t, result.RepairedSpec)
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, original, saved)
}
