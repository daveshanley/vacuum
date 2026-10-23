package languageserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/language-server/internal/lsp"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/require"
	"github.com/spf13/viper"
)

func TestParseViperIgnoreFiles(t *testing.T) {
	t.Cleanup(viper.Reset)
	for _, tc := range []struct {
		value any
		want  []string
		bad   bool
	}{
		{"one, two.yaml", []string{"one, two.yaml"}, false},
		{[]any{"one.yaml", "two.yaml"}, []string{"one.yaml", "two.yaml"}, false},
		{[]any{}, []string{}, false},
		{[]any{"one.yaml", 42}, nil, true},
	} {
		viper.Reset()
		viper.Set("ignore-file", tc.value)
		config, err := (&ServerState{}).parseViperConfig()
		if tc.bad {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, tc.want, config.IgnoreFiles)
	}
}

func writeLSPIgnoreFiles(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	first, second := filepath.Join(dir, "one, first.yaml"), filepath.Join(dir, "two.yaml")
	require.NoError(t, os.WriteFile(first, []byte("rule: [one]\n"), 0600))
	require.NoError(t, os.WriteFile(second, []byte("rule: [two]\n"), 0600))
	return dir, first, second
}

func TestLSPIgnoreFilesConfigReloadAndClear(t *testing.T) {
	dir, first, second := writeLSPIgnoreFiles(t)
	state := newRuntimeConfigTestState()
	config := &LSPConfig{IgnoreFiles: []string{filepath.Base(first), filepath.Base(second)}}
	require.NoError(t, state.setFileConfig(config, dir))
	want := model.IgnoredItems{"rule": {"one", "two"}}
	require.Equal(t, want, state.lintRequest.IgnoredResults)
	uri := fileURI(filepath.Join(dir, "api.yaml"))
	runtime, err := state.runtimeConfigForDocument(uri)
	require.NoError(t, err)
	require.Equal(t, want, runtime.ignoredResults)
	require.NoError(t, os.WriteFile(second, []byte("rule: [three]\n"), 0600))
	require.NoError(t, state.setFileConfig(config, dir))
	runtime, err = state.runtimeConfigForDocument(uri)
	require.NoError(t, err)
	require.Equal(t, model.IgnoredItems{"rule": {"one", "three"}}, runtime.ignoredResults)
	require.NoError(t, state.setFileConfig(&LSPConfig{IgnoreFiles: []string{}}, dir))
	require.Empty(t, state.lintRequest.IgnoredResults)
	runtime, err = state.runtimeConfigForDocument(uri)
	require.NoError(t, err)
	require.Empty(t, runtime.ignoredResults)
	require.NoError(t, state.setFileConfig(config, dir))
	require.NoError(t, state.setFileConfig(&LSPConfig{}, dir))
	require.Empty(t, state.lintRequest.IgnoredResults)
}

func TestLSPExplicitIgnoreFilesOverrideDiskConfig(t *testing.T) {
	dir, first, second := writeLSPIgnoreFiles(t)
	state := NewServer("test", &utils.LintFileRequest{IgnoreFiles: []string{first, second}})
	require.NoError(t, state.setFileConfig(&LSPConfig{IgnoreFile: first}, dir))
	want := model.IgnoredItems{"rule": {"one", "two"}}
	require.Equal(t, want, state.lintRequest.IgnoredResults)
	runtime, err := state.runtimeConfigForDocument(fileURI(filepath.Join(dir, "api.yaml")))
	require.NoError(t, err)
	require.Equal(t, want, runtime.ignoredResults)
	// Editor settings still override the command-line defaults.
	require.NoError(t, state.setInitConfig(&LSPConfig{IgnoreFile: second}))
	require.Equal(t, model.IgnoredItems{"rule": {"two"}}, state.lintRequest.IgnoredResults)
}

func TestLSPWorkspaceIgnoreFiles(t *testing.T) {
	dir, first, second := writeLSPIgnoreFiles(t)
	state := newRuntimeConfigTestState()
	state.setWorkspaceFolders(nil, nil)
	state.workspaceConfigurationSupported = true
	state.setCallFunc(lsp.CallFunc(func(_ string, _ any, result any) error {
		*result.(*[]any) = []any{map[string]any{"ignoreFiles": []string{first, second}}}
		return nil
	}))
	uri := fileURI(filepath.Join(dir, "api.yaml"))
	runtime, err := state.runtimeConfigForDocument(uri)
	require.NoError(t, err)
	require.Equal(t, model.IgnoredItems{"rule": {"one", "two"}}, runtime.ignoredResults)
	require.NoError(t, os.WriteFile(second, []byte("rule: ["), 0600))
	state.bumpConfigGeneration()
	_, err = state.runtimeConfigForDocument(uri)
	require.ErrorContains(t, err, second)
}

func TestMergeIgnoreConfigReplacesLegacyAndClonesList(t *testing.T) {
	target := &LSPConfig{IgnoreFile: "old.yaml"}
	source := &LSPConfig{IgnoreFiles: []string{"one.yaml", "two.yaml"}}
	MergeConfig(target, source)
	require.Empty(t, target.IgnoreFile)
	source.IgnoreFiles[0] = "changed"
	require.Equal(t, []string{"one.yaml", "two.yaml"}, target.ignoreFiles())
	MergeConfig(target, &LSPConfig{IgnoreFile: "legacy.yaml"})
	require.Nil(t, target.IgnoreFiles)
	require.Equal(t, []string{"legacy.yaml"}, target.ignoreFiles())
	config, err := ParseLSPConfig(map[string]any{"vacuum": map[string]any{"ignoreFiles": []string{}}})
	require.NoError(t, err)
	require.NotNil(t, config.IgnoreFiles)
}
