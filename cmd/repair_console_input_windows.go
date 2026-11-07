// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"golang.org/x/sys/windows"
	"os"
	"syscall"
)

func prepareRepairPrompt(fd int) error {
	handle := windows.Handle(fd)
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return err
	}
	if err := windows.SetConsoleMode(handle, mode|windows.ENABLE_PROCESSED_INPUT); err != nil {
		return err
	}
	return windows.FlushConsoleInputBuffer(handle)
}

func repairPromptSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
