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

func TestFileProcessorPersistsRegisteredRepair(t *testing.T) {
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
	require.NoError(t, result.Error)
	require.Equal(t, 1, calls)
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(saved), "Bad")
	require.Contains(t, string(saved), "title: \"\"")
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
		require.NoError(t, writeFixedFile(&motor.RuleSetExecutionResult{ModifiedSpec: []byte("fixed")}, path, ""))
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
