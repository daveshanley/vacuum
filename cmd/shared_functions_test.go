package cmd

import (
	"fmt"
	"github.com/pb33f/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderTime(t *testing.T) {
	start := time.Now()
	time.Sleep(1 * time.Millisecond)
	fi, _ := os.Stat("shared_functions.go")
	RenderTime(true, time.Since(start), fi.Size())
}

func TestLoadCustomFunctionsRejectsFlagValueAsPath(t *testing.T) {
	for _, annotations := range []bool{false, true} {
		_, err := LoadCustomFunctions("--no-banner", true, annotations)
		require.ErrorContains(t, err, "--functions requires a path")
	}
}

func TestLoadCustomFunctionsRendersInvalidScriptOnce(t *testing.T) {
	for _, annotations := range []bool{false, true} {
		t.Run(fmt.Sprint(annotations), func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "invalid.js"), []byte("function runRule("), 0600))
			stdout, stderr, err := captureProcessStreams(t, func() error {
				_, err := LoadCustomFunctions(dir, true, annotations)
				return err
			})
			require.Error(t, err)
			require.Equal(t, 1, strings.Count(stdout+stderr, `unable to load custom function "`))
		})
	}
}
