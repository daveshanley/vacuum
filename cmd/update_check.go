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
var activeCachedUpdateNotice *upgrade.CheckResult
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
	activeCachedUpdateNotice = nil
	activeUpdateNoticeWriter = nil

	cache, _ := updateCheckCacheFactory()
	if cache != nil {
		result, hasCachedRelease, shouldRefresh := cache.ReadStatus(currentVersion, upgrade.DefaultCacheMaxAge)
		if hasCachedRelease {
			activeCachedUpdateNotice = &result
			activeUpdateNoticeWriter = cmd.ErrOrStderr()
		}
		if !shouldRefresh {
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
	check := activeUpdateCheck
	cache := activeUpdateCache
	cachedNotice := activeCachedUpdateNotice
	writer := activeUpdateNoticeWriter
	activeUpdateCheck = nil
	activeUpdateCache = nil
	activeCachedUpdateNotice = nil
	activeUpdateNoticeWriter = nil

	if cachedNotice != nil {
		upgrade.RenderNotice(writer, *cachedNotice)
	}
	if check == nil {
		return
	}

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
	if cache != nil && result.Err == nil {
		_ = cache.Write(result)
	}
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
