package cmd

import (
	"github.com/pb33f/testify/require"
	"strings"
	"testing"
)

func TestApplyOverlayOperationFiltersAndTagPruning(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: Test, version: '1'}
tags: [{name: public}, {name: internal}, {name: unused}]
paths:
  /pets:
    get: {operationId: listPets, tags: [public], responses: {'200': {description: OK}}}
    post: {operationId: addPet, tags: [internal], responses: {'200': {description: OK}}}
`
	for _, flags := range [][]string{
		{"--exclude-tag", "internal", "--tag-match", "any"},
		{"--include-operation-id", "listPets", "--include-operation-id", "unknown"},
		{"--exclude-operation-id", "addPet"},
		{"--include-tag", "public", "--exclude-tag", "internal", "--include-operation-id", "listPets", "--exclude-operation-id", "addPet"},
	} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			cmd := GetApplyOverlayCommand()
			cmd.SetArgs(append([]string{"--stdin", "--stdout", "--prune-unused"}, flags...))
			cmd.SetIn(strings.NewReader(spec))
			out, stderr, err := captureProcessStreams(t, cmd.Execute)
			require.NoError(t, err)
			require.Empty(t, stderr)
			doc := normalizedYAML(t, []byte(out)).(map[string]any)
			paths := doc["paths"].(map[string]any)["/pets"].(map[string]any)
			require.Len(t, paths, 1)
			require.Contains(t, paths, "get")
			tags := doc["tags"].([]any)
			require.Len(t, tags, 1)
			require.Equal(t, "public", tags[0].(map[string]any)["name"])
		})
	}
}

func TestApplyOverlayRejectsBlankOperationFilters(t *testing.T) {
	for _, flag := range []string{"--exclude-tag", "--include-operation-id", "--exclude-operation-id"} {
		cmd := GetApplyOverlayCommand()
		cmd.SetArgs([]string{"--stdin", "--stdout", flag, " "})
		out, _, err := captureProcessStreams(t, cmd.Execute)
		require.ErrorContains(t, err, "must not be empty")
		require.Empty(t, out)
	}
}
