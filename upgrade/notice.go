// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/daveshanley/vacuum/color"
)

func RenderNotice(w io.Writer, result CheckResult) {
	if w == nil || !result.UpdateAvailable() {
		return
	}
	releaseURL := result.ReleaseURL
	if releaseURL == "" {
		releaseURL = DefaultReleaseNotesURL
	}

	primary := lipgloss.NewStyle().Foreground(color.RGBBlue).Bold(true)
	secondary := lipgloss.NewStyle().Foreground(color.RGBPink).Bold(true)
	tertiary := lipgloss.NewStyle().Foreground(color.RGBGrey)

	message := strings.Join([]string{
		fmt.Sprintf("%s: %s -> %s", primary.Render("UPDATE AVAILABLE"), result.CurrentVersion, result.LatestVersion),
		fmt.Sprintf("Run %s to install the latest release.", secondary.Render("vacuum upgrade")),
		tertiary.Render(fmt.Sprintf("Release notes: %s", releaseURL)),
	}, "\n")

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "┃",
			Right:       "│",
			TopLeft:     "┏",
			TopRight:    "┐",
			BottomLeft:  "┗",
			BottomRight: "┘",
		}).
		BorderForeground(color.RGBBlue).
		Padding(1, 2).
		MarginLeft(1)

	fmt.Fprintln(w)
	fmt.Fprintln(w, boxStyle.Render(message))
	fmt.Fprintln(w)
}
