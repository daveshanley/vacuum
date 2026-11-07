// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"golang.org/x/sys/unix"
	"os"
	"syscall"
)

func prepareRepairPrompt(fd int) error {
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return err
	}
	state.Lflag |= unix.ISIG // Keep Ctrl-C connected to the command's signal context.
	if err = unix.IoctlSetTermios(fd, unix.TCSETS, state); err != nil {
		return err
	}
	return unix.IoctlSetInt(fd, unix.TCFLSH, 0) // TCIFLUSH is zero on Linux.
}

func repairPromptSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP}
}
