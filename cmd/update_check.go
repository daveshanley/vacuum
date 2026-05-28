// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/upgrade"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var activeUpdateCheck *upgrade.Check
var activeUpdateCache *upgrade.UpdateCache
var activeUpdateNoticeWriter io.Writer
var updateCheckOptions upgrade.CheckOptions
var updateCheckCacheFactory = upgrade.DefaultUpdateCache
var updateCheckIsTerminal = func() bool {
	return term.IsTerminal(int(os.Stderr.Fd()))
}

func StartUpdateCheck(cmd *cobra.Command) {
	if !ShouldCheckForUpdates(cmd) {
		return
	}
	currentVersion := GetVersion()
	if !upgrade.IsComparableVersion(currentVersion) {
		return
	}
	activeUpdateCache = nil

	cache, _ := updateCheckCacheFactory()
	if cache != nil {
		result, hasFreshRelease, recentlyChecked := cache.ReadStatus(currentVersion, upgrade.DefaultCacheMaxAge, upgrade.DefaultFailureBackoff)
		if hasFreshRelease {
			activeUpdateCheck = upgrade.CompletedCheck(result)
			activeUpdateNoticeWriter = cmd.ErrOrStderr()
			return
		}
		if recentlyChecked {
			return
		}
	}

	opts := updateCheckOptions
	opts.CurrentVersion = currentVersion
	activeUpdateCheck = upgrade.StartCheck(opts)
	activeUpdateCache = cache
	activeUpdateNoticeWriter = cmd.ErrOrStderr()
}

func FlushUpdateNotice() {
	flushUpdateNotice(false)
}

func WaitForUpdateNotice() {
	flushUpdateNotice(true)
}

func flushUpdateNotice(wait bool) {
	if activeUpdateCheck == nil {
		return
	}
	check := activeUpdateCheck
	cache := activeUpdateCache
	writer := activeUpdateNoticeWriter
	activeUpdateCheck = nil
	activeUpdateCache = nil
	activeUpdateNoticeWriter = nil

	var result upgrade.CheckResult
	var ok bool
	if wait {
		result, ok = check.WaitResult()
	} else {
		result, ok = check.TryResult()
	}
	if !ok {
		check.Cancel()
		return
	}
	if cache != nil {
		if result.Err != nil {
			_ = cache.MarkChecked()
		} else {
			_ = cache.Write(result)
		}
	}
	upgrade.RenderNotice(writer, result)
}

func ShouldCheckForUpdates(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if boolFlagValue(cmd, "no-update-check") {
		return false
	}
	if os.Getenv("CI") != "" || !updateCheckIsTerminal() {
		return false
	}
	name := cmd.Name()
	if strings.HasPrefix(name, "__") {
		return false
	}
	switch name {
	case "version", "completion", "help", "language-server", "upgrade":
		return false
	}
	if boolFlagValue(cmd, "stdout") || boolFlagValue(cmd, "silent") || boolFlagValue(cmd, "pipeline-output") {
		return false
	}
	return true
}

func boolFlagValue(cmd *cobra.Command, name string) bool {
	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		flag = cmd.InheritedFlags().Lookup(name)
	}
	if flag == nil {
		flag = cmd.PersistentFlags().Lookup(name)
	}
	if flag == nil {
		return false
	}
	value, err := strconv.ParseBool(flag.Value.String())
	return err == nil && value
}
