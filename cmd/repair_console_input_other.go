//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !dragonfly && !windows

package cmd

import (
	"fmt"
	"os"
	"syscall"
)

func prepareRepairPrompt(fd int) error {
	return fmt.Errorf("interactive fix review is not supported on this platform")
}

func repairPromptSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
