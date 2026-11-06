package cmd

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestNewRepairFilePreservesSourceAndExactBytes(t *testing.T) {
	dir := t.TempDir()
	source, output := filepath.Join(dir, "api.yaml"), filepath.Join(dir, "api.fixed.yaml")
	original, candidate := []byte("title: Original\n"), []byte("# reviewed\ntitle: Fixed\n")
	require.NoError(t, os.WriteFile(source, original, 0640))
	require.NoError(t, writeNewRepairFile(source, output, original, candidate))
	actual, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, candidate, actual)
	actual, err = os.ReadFile(source)
	require.NoError(t, err)
	require.Equal(t, original, actual)
	info, err := os.Stat(output)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0640), info.Mode().Perm())
	staged, err := filepath.Glob(filepath.Join(dir, ".vacuum-repair-*"))
	require.NoError(t, err)
	require.Empty(t, staged)
}

func TestNewRepairFileRejectsChangedSourceAndExistingTargets(t *testing.T) {
	for _, kind := range []string{"stale", "source", "existing", "hardlink", "symlink", "source-symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			source, output := filepath.Join(dir, "api.yaml"), filepath.Join(dir, "fixed.yaml")
			original := []byte("original")
			require.NoError(t, os.WriteFile(source, original, 0600))
			switch kind {
			case "stale":
				require.NoError(t, os.WriteFile(source, []byte("external edit"), 0600))
			case "source":
				output = source
			case "existing":
				require.NoError(t, os.WriteFile(output, []byte("keep this"), 0600))
			case "hardlink":
				require.NoError(t, os.Link(source, output))
			case "symlink":
				require.NoError(t, os.Symlink(source, output))
			case "source-symlink":
				alias := filepath.Join(dir, "alias.yaml")
				require.NoError(t, os.Symlink(source, alias))
				source = alias
			}
			before, _ := os.ReadFile(source)
			targetBefore, targetErr := os.ReadFile(output)
			require.Error(t, writeNewRepairFile(source, output, original, []byte("fixed")))
			after, err := os.ReadFile(source)
			require.NoError(t, err)
			require.Equal(t, before, after)
			after, err = os.ReadFile(output)
			if targetErr == nil {
				require.NoError(t, err)
				require.Equal(t, targetBefore, after)
			} else {
				require.True(t, os.IsNotExist(err))
			}
			staged, err := filepath.Glob(filepath.Join(dir, ".vacuum-repair-*"))
			require.NoError(t, err)
			require.Empty(t, staged)
		})
	}
}

func TestRepairFileCreateOnlyPublicationRejectsLateDestination(t *testing.T) {
	dir := t.TempDir()
	staged, output := filepath.Join(dir, "staged"), filepath.Join(dir, "fixed.yaml")
	require.NoError(t, os.WriteFile(staged, []byte("reviewed candidate"), 0600))
	_, err := os.Lstat(output)
	require.True(t, os.IsNotExist(err))
	// A different writer creates the destination after our admission check.
	require.NoError(t, os.WriteFile(output, []byte("other writer"), 0600))
	require.ErrorIs(t, publishRepairFile(staged, output, true), os.ErrExist)
	actual, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, []byte("other writer"), actual)
}

func TestNewRepairFileConcurrentCreatesHaveOneWinner(t *testing.T) {
	dir := t.TempDir()
	source, output := filepath.Join(dir, "api.yaml"), filepath.Join(dir, "fixed.yaml")
	original := []byte("original")
	require.NoError(t, os.WriteFile(source, original, 0600))
	candidates := [][]byte{[]byte("first complete candidate\n"), []byte("second complete candidate\n")}
	start := make(chan struct{})
	errs := make([]error, len(candidates))
	var workers sync.WaitGroup
	for i := range candidates {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			errs[i] = writeNewRepairFile(source, output, original, candidates[i])
		}(i)
	}
	close(start)
	workers.Wait()
	actual, err := os.ReadFile(output)
	require.NoError(t, err)
	winners := 0
	for i, err := range errs {
		if err == nil {
			winners++
			require.Equal(t, candidates[i], actual)
		}
	}
	require.Equal(t, 1, winners)
	actual, err = os.ReadFile(source)
	require.NoError(t, err)
	require.Equal(t, original, actual)
	staged, err := filepath.Glob(filepath.Join(dir, ".vacuum-repair-*"))
	require.NoError(t, err)
	require.Empty(t, staged)
}
