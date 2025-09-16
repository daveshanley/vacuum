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

// GetVersionInfo returns version information using modern debug.ReadBuildInfo approach
// this works correctly with go install unlike the old ldflags method
func GetVersionInfo() VersionInfo {
	info := VersionInfo{
		Version:   "unknown",
		Commit:    "unknown",
		Date:      time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST"),
		GoVersion: "",
		Modified:  false,
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}

	// get version from go install (e.g., v0.14.2)
	if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(dev)" {
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
