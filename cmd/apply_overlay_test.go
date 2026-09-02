package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	openapitransform "github.com/daveshanley/vacuum/openapi/transform"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v4"
)

func captureProcessStreams(t *testing.T, run func() error) (string, string, error) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err)
	stderrReader, stderrWriter, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout, os.Stderr = stdoutWriter, stderrWriter
	outCh, errCh := make(chan []byte), make(chan []byte)
	go func() {
		data, _ := io.ReadAll(stdoutReader)
		outCh <- data
	}()
	go func() {
		data, _ := io.ReadAll(stderrReader)
		errCh <- data
	}()
	runErr := run()
	require.NoError(t, stdoutWriter.Close())
	require.NoError(t, stderrWriter.Close())
	os.Stdout, os.Stderr = oldOut, oldErr
	stdout, stderr := <-outCh, <-errCh
	require.NoError(t, stdoutReader.Close())
	require.NoError(t, stderrReader.Close())
	return string(stdout), string(stderr), runErr
}

func normalizedYAML(t *testing.T, data []byte) any {
	t.Helper()
	var value any
	require.NoError(t, yaml.Unmarshal(data, &value))
	return value
}

func TestApplyOverlayIssue948CombinedPublication(t *testing.T) {
	dir := filepath.Join("test_data", "issue_948")
	output := filepath.Join(t.TempDir(), "public.yaml")
	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{filepath.Join(dir, "spec.yaml"), filepath.Join(dir, "overlay.yaml"), output, "--include-tag", "public", "--prune-unused", "--no-style"})
	stdout, _, err := captureProcessStreams(t, cmd.Execute)
	require.NoError(t, err)
	assert.Contains(t, stdout, "Filtered 5 operations: 2 kept, 3 removed")
	assert.Contains(t, stdout, "Pruned 2 unused components")
	assert.Contains(t, stdout, "link targets filtered operationId")

	actual, err := os.ReadFile(output)
	require.NoError(t, err)
	expected, err := os.ReadFile(filepath.Join(dir, "expected-public.yaml"))
	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(normalizedYAML(t, expected), normalizedYAML(t, actual)))
	doc := normalizedYAML(t, actual).(map[string]any)
	assert.Equal(t, "3.0.3", doc["openapi"])
}

func TestApplyOverlayStdoutPurityAndFailOnWarnings(t *testing.T) {
	dir := filepath.Join("test_data", "issue_948")
	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{filepath.Join(dir, "spec.yaml"), filepath.Join(dir, "overlay.yaml"), "--stdout", "--include-tag", "public", "--prune-unused", "--fail-on-warnings"})
	stdout, stderr, err := captureProcessStreams(t, cmd.Execute)
	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, WarningsExitCode, exitErr.Code)
	assert.NotEmpty(t, normalizedYAML(t, []byte(stdout)))
	assert.NotContains(t, stdout, "Filtered ")
	assert.Contains(t, stderr, "link targets filtered operationId")
	assert.Contains(t, stderr, "Error: apply-overlay produced 1 warning")
}

func TestApplyOverlayInputOutputCombinations(t *testing.T) {
	dir := filepath.Join("test_data", "issue_948")
	spec, err := os.ReadFile(filepath.Join(dir, "spec.yaml"))
	require.NoError(t, err)
	overlay := filepath.Join(dir, "overlay.yaml")
	for _, tc := range []struct {
		name   string
		stdin  bool
		stdout bool
	}{
		{"file file", false, false},
		{"stdin file", true, false},
		{"file stdout", false, true},
		{"stdin stdout", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "out.yaml")
			args := make([]string, 0, 6)
			if !tc.stdin {
				args = append(args, filepath.Join(dir, "spec.yaml"))
			}
			args = append(args, overlay)
			if !tc.stdout {
				args = append(args, output)
			}
			if tc.stdin {
				args = append(args, "--stdin")
			}
			if tc.stdout {
				args = append(args, "--stdout")
			}
			args = append(args, "--include-tag", "public")
			cmd := GetApplyOverlayCommand()
			cmd.SetArgs(args)
			if tc.stdin {
				cmd.SetIn(bytes.NewReader(spec))
			}
			stdout, _, err := captureProcessStreams(t, cmd.Execute)
			require.NoError(t, err)
			if tc.stdout {
				assert.NotEmpty(t, normalizedYAML(t, []byte(stdout)))
			} else {
				data, readErr := os.ReadFile(output)
				require.NoError(t, readErr)
				assert.NotEmpty(t, normalizedYAML(t, data))
			}
		})
	}
}

func TestApplyOverlayDefaultPathPreservesOverlayBytes(t *testing.T) {
	dir := filepath.Join("test_data", "issue_948")
	spec, err := os.ReadFile(filepath.Join(dir, "spec.yaml"))
	require.NoError(t, err)
	overlay, err := os.ReadFile(filepath.Join(dir, "overlay.yaml"))
	require.NoError(t, err)
	expected, err := libopenapi.ApplyOverlayFromBytesToSpecBytes(spec, overlay)
	require.NoError(t, err)
	defer expected.OverlayDocument.Release()

	output := filepath.Join(t.TempDir(), "default.yaml")
	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{filepath.Join(dir, "spec.yaml"), filepath.Join(dir, "overlay.yaml"), output, "--no-style"})
	_, _, err = captureProcessStreams(t, cmd.Execute)
	require.NoError(t, err)
	actual, err := os.ReadFile(output)
	require.NoError(t, err)
	assert.Equal(t, expected.Bytes, actual)
}

func TestApplyOverlayTransformValidationAndAtomicErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		tags []string
		mode string
		set  bool
		want string
	}{
		{"deduplicate", []string{"public", "public"}, "any", false, ""},
		{"empty", []string{""}, "any", false, "must not be empty"},
		{"whitespace", []string{" \t"}, "any", false, "must not be empty"},
		{"bad mode", []string{"public"}, "some", true, "expected any or all"},
		{"orphan mode", nil, "all", true, "requires at least one"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tags, err := validateOverlayTransformFlags(tc.tags, tc.mode, tc.set)
			if tc.want == "" {
				require.NoError(t, err)
				assert.Len(t, tags, 1)
			} else {
				assert.ErrorContains(t, err, tc.want)
			}
		})
	}

	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.yaml")
	overlay := filepath.Join(dir, "overlay.yaml")
	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.0.0\npaths: {}\ncomponents:\n  schemas:\n    A: {$ref: other.yaml#/A}\n"), 0o600))
	require.NoError(t, os.WriteFile(overlay, []byte("overlay: 1.0.0\ninfo: {title: no-op, version: 1.0.0}\nactions:\n  - target: $.info\n    update: {x-test: true}\n"), 0o600))
	output := filepath.Join(dir, "out.yaml")
	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{spec, overlay, output, "--prune-unused"})
	_, _, err := captureProcessStreams(t, cmd.Execute)
	require.Error(t, err)
	assert.ErrorContains(t, err, "require a bundled OpenAPI document")
	_, statErr := os.Stat(output)
	assert.True(t, errors.Is(statErr, os.ErrNotExist))

	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.0.0\npaths:\n  /x:\n    get:\n      responses:\n        '200': {$ref: '#/components/responses/Missing'}\ncomponents:\n  schemas:\n    A: {type: string}\n"), 0o600))
	cmd = GetApplyOverlayCommand()
	cmd.SetArgs([]string{spec, overlay, output, "--prune-unused"})
	_, _, err = captureProcessStreams(t, cmd.Execute)
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing component")

	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.0.0\ninfo: {title: x, version: 1.0.0}\npaths: {}\ncomponents:\n  schemas:\n    A: {type: string}\n"), 0o600))
	require.NoError(t, os.WriteFile(overlay, []byte("overlay: 1.0.0\ninfo: {title: external, version: 1.0.0}\nactions:\n  - target: $.components.schemas.A\n    update: {$ref: 'other.yaml#/A'}\n"), 0o600))
	cmd = GetApplyOverlayCommand()
	cmd.SetArgs([]string{spec, overlay, output, "--prune-unused"})
	_, _, err = captureProcessStreams(t, cmd.Execute)
	require.Error(t, err)
	assert.ErrorContains(t, err, "external reference found")
}

func TestApplyOverlayZeroMatchWarningWritesOutput(t *testing.T) {
	dir := filepath.Join("test_data", "issue_948")
	output := filepath.Join(t.TempDir(), "empty.yaml")
	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{filepath.Join(dir, "spec.yaml"), filepath.Join(dir, "overlay.yaml"), output, "--include-tag", "nobody", "--prune-unused", "--no-style"})
	stdout, _, err := captureProcessStreams(t, cmd.Execute)
	require.NoError(t, err)
	assert.Contains(t, stdout, "kept zero operations")
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	doc := normalizedYAML(t, data).(map[string]any)
	assert.Empty(t, doc["paths"].(map[string]any))
	_, hasComponents := doc["components"]
	assert.False(t, hasComponents)
}

func TestApplyOverlayZeroMatchWarningUsesFinalPrunedDocument(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.yaml")
	overlay := filepath.Join(dir, "overlay.yaml")
	output := filepath.Join(dir, "out.yaml")
	require.NoError(t, os.WriteFile(spec, []byte(`openapi: 3.1.0
info: {title: final operation count, version: 1.0.0}
paths:
  /internal:
    get:
      tags: [internal]
      responses: {}
components:
  pathItems:
    Orphan:
      get:
        tags: [public]
        responses: {}
`), 0o600))
	require.NoError(t, os.WriteFile(overlay, []byte(`overlay: 1.0.0
info: {title: no-op, version: 1.0.0}
actions:
  - target: $.info
    update: {title: final operation count}
`), 0o600))

	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{spec, overlay, output, "--include-tag", "public", "--prune-unused", "--fail-on-warnings", "--no-style"})
	stdout, _, err := captureProcessStreams(t, cmd.Execute)
	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, WarningsExitCode, exitErr.Code)
	assert.Contains(t, stdout, "kept zero operations")

	data, readErr := os.ReadFile(output)
	require.NoError(t, readErr)
	doc := normalizedYAML(t, data).(map[string]any)
	assert.Empty(t, doc["paths"].(map[string]any))
	_, hasComponents := doc["components"]
	assert.False(t, hasComponents)
}

func TestApplyOverlayStandardWarningFailOnWarningsParity(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.yaml")
	overlay := filepath.Join(dir, "overlay.yaml")
	output := filepath.Join(dir, "out.yaml")
	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.0.0\ninfo: {title: x, version: 1.0.0}\npaths: {}\n"), 0o600))
	require.NoError(t, os.WriteFile(overlay, []byte("overlay: 1.0.0\ninfo: {title: warning, version: 1.0.0}\nactions:\n  - target: $.missing\n    update: {value: true}\n"), 0o600))
	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{spec, overlay, output, "--fail-on-warnings", "--no-style"})
	_, _, err := captureProcessStreams(t, cmd.Execute)
	var exitErr *ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, WarningsExitCode, exitErr.Code)
	_, statErr := os.Stat(output)
	assert.NoError(t, statErr)
}

func TestApplyOverlayMultipleTagsAllMode(t *testing.T) {
	dir := filepath.Join("test_data", "issue_948")
	output := filepath.Join(t.TempDir(), "all.yaml")
	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{filepath.Join(dir, "spec.yaml"), filepath.Join(dir, "overlay.yaml"), output, "--include-tag", "public", "--include-tag", "shared", "--tag-match", "all", "--no-style"})
	_, _, err := captureProcessStreams(t, cmd.Execute)
	require.NoError(t, err)
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	text := string(data)
	assert.Contains(t, text, "getPublic")
	assert.NotContains(t, text, "promotedByOverlay")
}

func TestApplyOverlayPruneOnlyAndNoRemovalTransform(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.yaml")
	overlay := filepath.Join(dir, "overlay.yaml")
	output := filepath.Join(dir, "out.yaml")
	require.NoError(t, os.WriteFile(spec, []byte(`openapi: 3.0.3
info: {title: modes, version: 1.0.0}
paths:
  /x:
    get:
      tags: [public]
      responses:
        '200':
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Used'}
components:
  schemas:
    Used: {type: string}
    Unused: {type: string}
`), 0o600))
	require.NoError(t, os.WriteFile(overlay, []byte("overlay: 1.0.0\ninfo: {title: no-op, version: 1.0.0}\nactions:\n  - target: $.info\n    update: {title: modes}\n"), 0o600))

	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{spec, overlay, output, "--prune-unused", "--no-style"})
	_, _, err := captureProcessStreams(t, cmd.Execute)
	require.NoError(t, err)
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Used:")
	assert.NotContains(t, string(data), "Unused:")

	applied, err := libopenapi.ApplyOverlayFromBytesToSpecBytes(mustReadTestFile(t, spec), mustReadTestFile(t, overlay))
	require.NoError(t, err)
	defer applied.OverlayDocument.Release()
	cmd = GetApplyOverlayCommand()
	cmd.SetArgs([]string{spec, overlay, output, "--include-tag", "public", "--no-style"})
	_, _, err = captureProcessStreams(t, cmd.Execute)
	require.NoError(t, err)
	data, err = os.ReadFile(output)
	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(normalizedYAML(t, applied.Bytes), normalizedYAML(t, data)))
}

func mustReadTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func TestApplyOverlayConfigurationAndEnvironmentCollections(t *testing.T) {
	dir := filepath.Join("test_data", "issue_948")
	for _, tc := range []struct {
		name      string
		configure func(*testing.T) []string
	}{
		{
			name: "configuration list",
			configure: func(t *testing.T) []string {
				config := filepath.Join(t.TempDir(), "vacuum.yaml")
				require.NoError(t, os.WriteFile(config, []byte("apply-overlay:\n  include-tag:\n    - public\n    - shared\n  tag-match: all\n  prune-unused: true\n"), 0o600))
				return []string{"--config", config}
			},
		},
		{
			name: "environment CSV",
			configure: func(t *testing.T) []string {
				t.Setenv("VACUUM_INCLUDE_TAG", "public,shared")
				t.Setenv("VACUUM_TAG_MATCH", "all")
				t.Setenv("VACUUM_PRUNE_UNUSED", "true")
				return nil
			},
		},
		{
			name: "environment overrides configuration",
			configure: func(t *testing.T) []string {
				config := filepath.Join(t.TempDir(), "vacuum.yaml")
				require.NoError(t, os.WriteFile(config, []byte("apply-overlay:\n  include-tag: [internal]\n  tag-match: any\n  prune-unused: false\n"), 0o600))
				t.Setenv("VACUUM_INCLUDE_TAG", "public,shared")
				t.Setenv("VACUUM_TAG_MATCH", "all")
				t.Setenv("VACUUM_PRUNE_UNUSED", "true")
				return []string{"--config", config}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			configFile = ""
			t.Cleanup(func() {
				viper.Reset()
				configFile = ""
			})
			output := filepath.Join(t.TempDir(), "configured.yaml")
			args := []string{"apply-overlay", filepath.Join(dir, "spec.yaml"), filepath.Join(dir, "overlay.yaml"), output, "--no-style", "--no-update-check"}
			args = append(args, tc.configure(t)...)
			root := GetRootCommand()
			root.SetArgs(args)
			_, _, err := captureProcessStreams(t, root.Execute)
			require.NoError(t, err)
			data, err := os.ReadFile(output)
			require.NoError(t, err)
			text := string(data)
			assert.Contains(t, text, "getPublic")
			assert.NotContains(t, text, "promotedByOverlay")
			assert.NotContains(t, text, "Internal:")
		})
	}
}

func TestApplyOverlayErrorRenderer(t *testing.T) {
	stdout, stderr, err := captureProcessStreams(t, func() error {
		renderApplyOverlayError(true, errors.New("boom"))
		return nil
	})
	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "Error: boom")
}

func TestApplyOverlayHelpDocumentsTransforms(t *testing.T) {
	cmd := GetApplyOverlayCommand()
	assert.Contains(t, cmd.Long, "case-sensitive")
	assert.Contains(t, cmd.Long, "bundled, self-contained")
	assert.True(t, strings.Contains(cmd.Example, "--include-tag public") && strings.Contains(cmd.Example, "--prune-unused"))
}

func BenchmarkIssue948ApplyOverlay(b *testing.B) {
	dir := filepath.Join("test_data", "issue_948")
	spec, err := os.ReadFile(filepath.Join(dir, "spec.yaml"))
	if err != nil {
		b.Fatal(err)
	}
	overlay, err := os.ReadFile(filepath.Join(dir, "overlay.yaml"))
	if err != nil {
		b.Fatal(err)
	}
	b.Run("standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result, err := libopenapi.ApplyOverlayFromBytesToSpecBytes(spec, overlay)
			if err != nil {
				b.Fatal(err)
			}
			result.OverlayDocument.Release()
		}
	})
	b.Run("filter-and-prune", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result, err := libopenapi.ApplyOverlayFromBytesToSpecBytes(spec, overlay)
			if err != nil {
				b.Fatal(err)
			}
			root := result.OverlayDocument.GetSpecInfo().RootNode
			if _, err = openapitransform.FilterOperationsByTags(root, result.OverlayDocument.GetVersion(), openapitransform.TagFilterOptions{IncludeTags: []string{"public"}}); err != nil {
				b.Fatal(err)
			}
			if _, err = openapitransform.PruneUnusedComponents(root, result.OverlayDocument.GetVersion()); err != nil {
				b.Fatal(err)
			}
			if _, err = yaml.Marshal(root); err != nil {
				b.Fatal(err)
			}
			result.OverlayDocument.Release()
		}
	})
}
