//go:build js

// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"errors"
	"os"
	"strconv"
	"syscall/js"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
)

// The browser host runs vacuum against an in-page terminal. It passes the
// terminal size as COLUMNS and LINES, and later resizes through vacuumResize.

// TerminalSize reports the size of the in-page terminal.
func TerminalSize() (int, int, error) {
	width, widthErr := strconv.Atoi(os.Getenv("COLUMNS"))
	height, heightErr := strconv.Atoi(os.Getenv("LINES"))
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return 0, 0, errors.New("terminal size not provided by the browser host")
	}
	return width, height, nil
}

// programOptions reads keys from stdin (there is no TTY to open), draws at the
// host terminal's size, and keeps full colour on the xterm.js terminal.
func programOptions() []tea.ProgramOption {
	options := []tea.ProgramOption{
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
		tea.WithColorProfile(colorprofile.TrueColor),
	}
	if width, height, err := TerminalSize(); err == nil {
		options = append(options, tea.WithWindowSize(width, height))
	}
	return options
}

// attachProgram lets the host page deliver terminal resizes to the running program.
func attachProgram(program *tea.Program) func() {
	resize := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 2 {
			program.Send(tea.WindowSizeMsg{Width: args[0].Int(), Height: args[1].Int()})
		}
		return nil
	})
	js.Global().Set("vacuumResize", resize)
	return func() {
		js.Global().Delete("vacuumResize")
		resize.Release()
	}
}
