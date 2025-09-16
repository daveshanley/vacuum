// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"strings"

	color2 "github.com/daveshanley/vacuum/color"
)

const (
	errorPrefix   = "✗"
	warningPrefix = "▲"
	infoPrefix    = "●"
	successPrefix = "✓"
	indentSpace   = "  "
)

func renderMessage(prefix, color, message string) {
	lines := strings.Split(message, "\n")

	if color2.AreColorsDisabled() {
		fmt.Printf("%s %s\n", prefix, lines[0])
		for i := 1; i < len(lines); i++ {
			if lines[i] != "" {
				fmt.Printf("%s%s\n", indentSpace, lines[i])
			}
		}
		fmt.Println()
	} else {
		fmt.Printf("%s%s %s%s\n", color, prefix, lines[0], color2.ASCIIReset)
		for i := 1; i < len(lines); i++ {
			if lines[i] != "" {
				fmt.Printf("%s%s\n", indentSpace, lines[i])
			}
		}
		fmt.Println()
	}
}

func RenderError(err error) {
	if err != nil {
		renderMessage(errorPrefix, color2.ASCIIRed, err.Error())
	}
}

func RenderErrorString(format string, args ...interface{}) {
	renderMessage(errorPrefix, color2.ASCIIRed, fmt.Sprintf(format, args...))
}

func RenderWarning(format string, args ...interface{}) {
	renderMessage(warningPrefix, color2.ASCIIYellow, fmt.Sprintf(format, args...))
}

func RenderInfo(format string, args ...interface{}) {
	renderMessage(infoPrefix, color2.ASCIIBlue, fmt.Sprintf(format, args...))
}

func RenderSuccess(format string, args ...interface{}) {
	renderMessage(successPrefix, color2.ASCIIGreen, fmt.Sprintf(format, args...))
}
