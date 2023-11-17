package cmd

import (
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
