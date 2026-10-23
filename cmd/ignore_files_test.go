package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/require"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func writeIgnoreFixture(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))
	return path
}

func TestLoadIgnoreFiles(t *testing.T) {
	first := writeIgnoreFixture(t, "first.yaml", "rule: [one, two]\nother: [three]\n")
	second := writeIgnoreFixture(t, "second.yaml", "rule: [four]\n")
	merged, err := LoadIgnoreFiles([]string{first, second}, true, true, false, true)
	require.NoError(t, err)
	require.Equal(t, model.IgnoredItems{"rule": {"one", "two", "four"}, "other": {"three"}}, merged)
	for _, bad := range []string{filepath.Join(t.TempDir(), "missing.yaml"), writeIgnoreFixture(t, "bad.yaml", "rule: [")} {
		merged, err = LoadIgnoreFiles([]string{first, bad}, true, true, false, true)
		require.ErrorContains(t, err, bad)
		require.Nil(t, merged)
	}
	require.Equal(t, []string{first, second}, (&LintFlags{IgnoreFile: first, IgnoreFiles: []string{second}}).ignoreFiles())
}

func TestIgnoreFileFlagsRepeatable(t *testing.T) {
	for _, factory := range []func() *cobra.Command{GetLintCommand, GetSpectralReportCommand, GetHTMLReportCommand, GetVacuumReportCommand, GetDashboardCommand, GetLanguageServerCommand, GetDocsCommand, GetSchemaCommand} {
		command := factory()
		t.Run(command.Name(), func(t *testing.T) {
			require.NoError(t, command.ParseFlags([]string{"--ignore-file", "one, two.yaml", "--ignore-file", "three.yaml"}))
			files, err := command.Flags().GetStringArray("ignore-file")
			require.NoError(t, err)
			require.Equal(t, []string{"one, two.yaml", "three.yaml"}, files)
		})
	}
}

func TestIgnoreFileConfigScalarAndList(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  []string
	}{
		{"one, two.yaml", []string{"one, two.yaml"}},
		{"", []string{}},
		{[]any{"one.yaml", "two.yaml"}, []string{"one.yaml", "two.yaml"}},
	} {
		command := GetLintCommand()
		config := viper.New()
		config.Set("ignore-file", tc.value)
		require.NoError(t, bindFlags(command.Flags(), config))
		got, err := command.Flags().GetStringArray("ignore-file")
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
}

func TestSpectralReportMultipleIgnoreFiles(t *testing.T) {
	spec := writeIgnoreFixture(t, "spec.yaml", "openapi: 3.1.0\ninfo:\n  title: Example\n  version: 1.0.0\npaths:\n  /pets: {}\n  /stores: {}\n")
	rules := writeIgnoreFixture(t, "rules.yaml", "rules:\n  no-plural-paths:\n    severity: warn\n    given: $.paths[*]~\n    then:\n      function: pattern\n      functionOptions:\n        notMatch: s$\n")
	first := writeIgnoreFixture(t, "pets.yaml", "no-plural-paths:\n  - \"$.paths['/pets']\"\n")
	second := writeIgnoreFixture(t, "stores.yaml", "no-plural-paths:\n  - \"$.paths['/stores']\"\n")
	for _, tc := range []struct {
		files     []string
		remaining bool
	}{{[]string{first}, true}, {[]string{first, second}, false}, {[]string{second, first}, false}} {
		command := GetSpectralReportCommand()
		registerPersistentFlags(command)
		output := filepath.Join(t.TempDir(), "report.json")
		args := []string{"--no-style", "-r", rules, spec, output}
		for _, file := range tc.files {
			args = append(args, "--ignore-file", file)
		}
		command.SetArgs(args)
		require.NoError(t, command.Execute())
		data, err := os.ReadFile(requireSingleGeneratedFile(t, output))
		require.NoError(t, err)
		if tc.remaining {
			require.Contains(t, string(data), "no-plural-paths")
		} else {
			require.JSONEq(t, "[]", string(data))
		}
	}
}

func TestLanguageServerMultipleIgnoreFiles(t *testing.T) {
	original := runLanguageServer
	t.Cleanup(func() { runLanguageServer = original })
	first := writeIgnoreFixture(t, "one.yaml", "rule: [one]\n")
	second := writeIgnoreFixture(t, "two.yaml", "rule: [two]\n")
	called := false
	runLanguageServer = func(_ string, request *utils.LintFileRequest, _ *motor.ExecutionOptions) error {
		called = true
		require.Equal(t, []string{first, second}, request.IgnoreFiles)
		require.Equal(t, model.IgnoredItems{"rule": {"one", "two"}}, request.IgnoredResults)
		return nil
	}
	command := GetLanguageServerCommand()
	registerPersistentFlags(command)
	command.SetArgs([]string{"--ignore-file", first, "--ignore-file", second})
	require.NoError(t, command.Execute())
	require.True(t, called)
}

func TestDocsDiagnosticsFingerprintMultipleIgnoreFiles(t *testing.T) {
	first := writeIgnoreFixture(t, "one.yaml", "rule: [one]\n")
	second := writeIgnoreFixture(t, "two.yaml", "rule: [two]\n")
	flags := &LintFlags{IgnoreFiles: []string{first, second}}
	before := docsDiagnosticsFingerprint(true, flags, nil)
	require.NoError(t, os.WriteFile(second, []byte("rule: [three]\n"), 0600))
	require.NotEqual(t, before, docsDiagnosticsFingerprint(true, flags, nil))
}

func TestPathArrayConfigKeepsCommaGlobs(t *testing.T) {
	for _, name := range []string{"globbed-files", "include", "exclude"} {
		for _, value := range []any{"specs/{pets,stores}/*.yaml", []any{"specs/{pets,stores}/*.yaml", "other/*.json"}} {
			cmd := GetSchemaCommand()
			tree := viper.New()
			tree.Set(name, value)
			require.NoError(t, bindFlags(cmd.Flags(), tree))
			got, err := cmd.Flags().GetStringArray(name)
			require.NoError(t, err)
			require.Equal(t, "specs/{pets,stores}/*.yaml", got[0])
			if _, ok := value.(string); ok {
				require.Len(t, got, 1)
			} else {
				require.Len(t, got, 2)
			}
		}
	}
}

func TestLanguageServerConfigPathsAreNotCLIOverrides(t *testing.T) {
	original := runLanguageServer
	t.Cleanup(func() { runLanguageServer = original })
	path := writeIgnoreFixture(t, "one.yaml", "rule: [one]\n")
	runLanguageServer = func(_ string, request *utils.LintFileRequest, _ *motor.ExecutionOptions) error {
		require.Nil(t, request.IgnoreFiles)
		require.Equal(t, model.IgnoredItems{"rule": {"one"}}, request.IgnoredResults)
		return nil
	}
	cmd := GetLanguageServerCommand()
	registerPersistentFlags(cmd)
	tree := viper.New()
	tree.Set("ignore-file", path)
	require.NoError(t, bindFlags(cmd.Flags(), tree))
	require.NoError(t, cmd.Execute())
}
