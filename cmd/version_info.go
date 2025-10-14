// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"runtime/debug"
	"time"
)

type VersionInfo struct {
	Version   string
	Commit    string
	Date      string
	GoVersion string
	Modified  bool
}

// GetVersionInfo returns version information using ldflags (if set) or debug.ReadBuildInfo as fallback
// This hybrid approach supports both:
// - Package managers using ldflags: go build -ldflags "-X main.version=v1.0.0 ..."
// - Direct go install: go install github.com/daveshanley/vacuum@latest
func GetVersionInfo() VersionInfo {
	info := VersionInfo{
		Version:   "unknown",
		Commit:    "unknown",
		Date:      time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"),
		GoVersion: "",
		Modified:  false,
	}

	// First, check if ldflags were provided (package managers use this)
	if ldVersion != "" {
		info.Version = ldVersion
	}
	if ldCommit != "" {
		info.Commit = ldCommit
	}
	if ldDate != "" {
		// Parse the date if it's in RFC3339 format, otherwise use as-is
		if parsed, err := time.Parse(time.RFC3339, ldDate); err == nil {
			info.Date = parsed.Format("Mon, 02 Jan 2006 15:04:05 MST")
		} else {
			info.Date = ldDate
		}
	}

	// If ldflags provided version info, we're done
	if ldVersion != "" || ldCommit != "" {
		// Get Go version from buildInfo if available
		if buildInfo, ok := debug.ReadBuildInfo(); ok {
			info.GoVersion = buildInfo.GoVersion
		}
		return info
	}

	// Fall back to debug.ReadBuildInfo for go install compatibility
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}

	// get version from go install (e.g., v0.14.2)
	if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		info.Version = buildInfo.Main.Version
	}

	// extract git information from build settings
	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			if len(setting.Value) >= 7 {
				info.Commit = setting.Value[:7] // short commit hash
			} else {
				info.Commit = setting.Value
			}
		case "vcs.time":
			if parsed, err := time.Parse(time.RFC3339, setting.Value); err == nil {
				info.Date = parsed.Format("Mon, 02 Jan 2006 15:04:05 MST")
			}
		case "vcs.modified":
			info.Modified = setting.Value == "true"
		}
	}

	// append modified indicator if there were local changes
	if info.Modified {
		info.Commit += "+CHANGES"
	}

	info.GoVersion = buildInfo.GoVersion

	return info
}
