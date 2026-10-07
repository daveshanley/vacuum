package cmd

import (
	"github.com/pb33f/testify/require"
	"os"
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
