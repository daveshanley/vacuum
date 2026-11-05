//go:build !js

// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"
)

// TerminalSize reports the width and height of the terminal attached to stdout.
func TerminalSize() (int, int, error) {
	return term.GetSize(int(os.Stdout.Fd()))
}

// programOptions returns platform options for interactive programs; native
// terminals need none beyond bubbletea's defaults.
func programOptions() []tea.ProgramOption {
	return nil
}

// attachProgram connects a running program to its host; native terminals
// deliver resizes through signals, so there is nothing to attach.
func attachProgram(_ *tea.Program) func() {
	return func() {}
}
