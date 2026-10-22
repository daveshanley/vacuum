package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestBundleRejectsAbsoluteDependencies(t *testing.T) {
	for _, kind := range []string{"input", "ruleset", "extends", "reference"} {
		t.Run(kind, func(t *testing.T) {
			s, path := testSuite(t)
			dir := filepath.Dir(path)
			switch kind {
			case "input":
				s.Cases[0].Input = filepath.Join(dir, "good.yaml")
			case "ruleset":
				s.Ruleset = filepath.Join(dir, "rules.yaml")
			case "extends":
				base := fixture(t, "base.yaml", testRules, dir)
				fixture(t, "rules.yaml", "extends: "+base+"\nrules: {}\n", dir)
			case "reference":
				ref := fixture(t, "pet.yaml", "type: string\n", dir)
				fixture(t, "good.yaml", good+"components:\n  schemas:\n    Pet:\n      $ref: "+ref+"\n", dir)
			}
			encoded, err := json.Marshal(s)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, encoded, 0600))
			r := RunFile(path, RunOptions{})
			require.Equal(t, 0, r.ExitCode, "%+v", r.Cases)
			require.False(t, r.Manifest.Reproducible)
			require.Error(t, WriteBundle(r, path, filepath.Join(t.TempDir(), "bundle")))
		})
	}
}

func TestBundleReplaysWithoutOriginalDirectory(t *testing.T) {
	s, path := testSuite(t)
	encoded, err := json.Marshal(s)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, encoded, 0600))
	r := RunFile(path, RunOptions{})
	dest := filepath.Join(t.TempDir(), "bundle")
	require.NoError(t, WriteBundle(r, path, dest))
	require.NoError(t, os.RemoveAll(filepath.Dir(path)))
	require.Equal(t, 0, RunFile(filepath.Join(dest, "suite.yaml"), RunOptions{}).ExitCode)
}
