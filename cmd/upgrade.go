// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/daveshanley/vacuum/upgrade"
	"github.com/spf13/cobra"
)

func GetUpgradeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "upgrade",
		Short:         "Upgrade vacuum to the latest published release",
		Long:          "Upgrade vacuum to the latest published release using the detected installation method.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runUpgrade,
	}
	return cmd
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()
	current := GetVersion()
	cmdCtx := cmd.Context()
	if cmdCtx == nil {
		cmdCtx = context.Background()
	}

	ctx, cancel := context.WithTimeout(cmdCtx, 15*time.Second)
	defer cancel()
	opts := updateCheckOptions
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}
	latest, err := upgrade.FetchLatestRelease(ctx, opts)
	if err != nil {
		return fmt.Errorf("unable to check latest vacuum release: %w", err)
	}

	if cmp, ok := upgrade.CompareVersions(latest.TagName, current); ok && cmp <= 0 {
		fmt.Fprintf(out, "vacuum is already up to date (%s).\n", current)
		return nil
	}

	installContext := upgrade.DetectInstallContext()
	action := upgrade.PlanUpgrade(installContext, latest.TagName)
	if !action.CanRun {
		fmt.Fprintf(out, "vacuum %s is available. Current version: %s.\n", latest.TagName, current)
		fmt.Fprintf(out, "Automatic upgrade is not available: %s.\n", action.Reason)
		writeManualUpgradeCommands(out, action, latest.TagName)
		return nil
	}

	fmt.Fprintf(out, "Upgrading vacuum %s -> %s via %s...\n", current, latest.TagName, action.Method)
	runErr := upgrade.RunAction(cmdCtx, action, out, errOut)
	if runErr != nil {
		fmt.Fprintf(errOut, "Automatic upgrade failed: %v\n", runErr)
		fmt.Fprintf(errOut, "You can retry manually with:\n  %s\n", action.CommandString())
		return runErr
	}
	verifyCtx, verifyCancel := context.WithTimeout(cmdCtx, 30*time.Second)
	defer verifyCancel()
	if verifyErr := upgrade.VerifyUpgrade(verifyCtx, installContext, action.Method, latest.TagName); verifyErr != nil {
		fmt.Fprintf(errOut, "Warning: automatic upgrade verification failed: %v\n", verifyErr)
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "vacuum upgrade completed.")
	return nil
}

func writeManualUpgradeCommands(out io.Writer, action upgrade.Action, latestVersion string) {
	if command := action.CommandString(); command != "" {
		fmt.Fprintln(out, "Use this command:")
		fmt.Fprintf(out, "  %s\n", command)
		return
	}
	fmt.Fprintln(out, "Use one of these commands:")
	for _, command := range upgrade.ManualCommands(latestVersion) {
		fmt.Fprintf(out, "  %s\n", command)
	}
}
