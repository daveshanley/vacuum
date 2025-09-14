// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cui

import (
	"fmt"
	"strings"
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

	if AreColorsDisabled() {
		fmt.Printf("%s %s\n", prefix, lines[0])
		for i := 1; i < len(lines); i++ {
			if lines[i] != "" {
				fmt.Printf("%s%s\n", indentSpace, lines[i])
			}
		}
		fmt.Println()
	} else {
		fmt.Printf("%s%s %s%s\n", color, prefix, lines[0], ASCIIReset)
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
		renderMessage(errorPrefix, ASCIIRed, err.Error())
	}
}

func RenderErrorString(format string, args ...interface{}) {
	renderMessage(errorPrefix, ASCIIRed, fmt.Sprintf(format, args...))
}

func RenderWarning(format string, args ...interface{}) {
	renderMessage(warningPrefix, ASCIIYellow, fmt.Sprintf(format, args...))
}

func RenderInfo(format string, args ...interface{}) {
	renderMessage(infoPrefix, ASCIIBlue, fmt.Sprintf(format, args...))
}

func RenderSuccess(format string, args ...interface{}) {
	renderMessage(successPrefix, ASCIIGreen, fmt.Sprintf(format, args...))
}
