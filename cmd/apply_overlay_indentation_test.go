package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestApplyOverlayIssue1037PreservesIndentation(t *testing.T) {
	const retained = `openapi: 3.0.3
info:
	title: Sample API # keep this comment
	version: 1.0.0
paths:
	/users:
		get:
			tags:
				- users
			responses:
				'200':
					description: |
						A list of users.
						One entry per user.
`
	const products = `	/products:
		get:
			tags:
				- products
			responses:
				'200':
					description: A list of products
`
	const unused = `components:
	schemas:
		Unused:
			type: string
`
	const overlay = "overlay: 1.0.0\ninfo: {title: Remove products, version: '1'}\nactions:\n  - target: $.paths['/products']\n    remove: true\n"
	for _, indent := range []int{2, 4} {
		for _, tc := range []struct {
			name     string
			flags    []string
			overlay  bool
			expected string
		}{
			{"filter", []string{"--include-tag", "users"}, false, retained + unused},
			{"prune", []string{"--prune-unused"}, false, retained + products},
			{"filter_and_prune", []string{"--include-tag", "users", "--prune-unused"}, false, retained},
			{"overlay", nil, true, retained + unused},
			{"overlay_filter_and_prune", []string{"--include-tag", "users", "--prune-unused"}, true, retained},
		} {
			for _, piped := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d_spaces/%s/piped=%t", indent, tc.name, piped), func(t *testing.T) {
					dir := t.TempDir()
					input, output := filepath.Join(dir, "input.yaml"), filepath.Join(dir, "output.yaml")
					spec := strings.ReplaceAll(retained+products+unused, "\t", strings.Repeat(" ", indent))
					expected := strings.ReplaceAll(tc.expected, "\t", strings.Repeat(" ", indent))
					require.NoError(t, os.WriteFile(input, []byte(spec), 0o600))
					args := []string{input}
					if piped {
						args = []string{"--stdin"}
					}
					if tc.overlay {
						overlayPath := filepath.Join(dir, "overlay.yaml")
						require.NoError(t, os.WriteFile(overlayPath, []byte(overlay), 0o600))
						args = append(args, overlayPath)
					}
					if piped {
						args = append(args, "--stdout")
					} else {
						args = append(args, output)
					}
					cmd := GetApplyOverlayCommand()
					cmd.SetArgs(append(args, tc.flags...))
					cmd.SetIn(strings.NewReader(spec))
					actual, stderr, err := captureProcessStreams(t, cmd.Execute)
					require.NoError(t, err)
					require.Empty(t, stderr)
					if !piped {
						data, err := os.ReadFile(output)
						require.NoError(t, err)
						actual = string(data)
					}
					require.Equal(t, expected, actual)
				})
			}
		}
	}
}
