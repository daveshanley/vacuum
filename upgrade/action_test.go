// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"strings"
	"testing"
)

func TestShellInstallerActionUsesNativeVerifiedInstaller(t *testing.T) {
	action := shellInstallerAction("/usr/local/bin/vacuum", "v0.27.0")
	if !action.CanRun {
		t.Skipf("shell installer action cannot run in this environment: %s", action.Reason)
	}
	if action.Command != "" || len(action.Args) != 0 {
		t.Fatalf("shell action should not execute a shell command: %#v", action)
	}
	if action.Executable != "/usr/local/bin/vacuum" || action.LatestVersion != "v0.27.0" {
		t.Fatalf("shell action did not preserve install target: %#v", action)
	}
	if !strings.Contains(action.CommandString(), "https://raw.githubusercontent.com/daveshanley/vacuum/v0.27.0/bin/install.sh") {
		t.Fatalf("manual fallback command did not pin installer tag: %q", action.CommandString())
	}
}

func TestManualCommandsPinShellInstallerReleaseTag(t *testing.T) {
	commands := strings.Join(ManualCommands("v0.27.0"), "\n")
	if !strings.Contains(commands, "https://raw.githubusercontent.com/daveshanley/vacuum/v0.27.0/bin/install.sh") {
		t.Fatalf("manual commands did not pin shell installer tag:\n%s", commands)
	}
	if !strings.Contains(commands, "VERSION='0.27.0'") {
		t.Fatalf("manual commands did not pin shell installer version:\n%s", commands)
	}
	if !strings.Contains(commands, "brew upgrade --cask daveshanley/vacuum/vacuum") {
		t.Fatalf("manual commands did not use published Homebrew cask token:\n%s", commands)
	}
}

func TestBrewUpgradeActionUsesStandardUpdate(t *testing.T) {
	action := brewUpgradeAction(HomebrewKindCask)
	if !action.CanRun {
		t.Skipf("brew upgrade action cannot run in this environment: %s", action.Reason)
	}
	command := action.CommandString()
	if !strings.Contains(command, "brew update && brew upgrade --cask daveshanley/vacuum/vacuum") {
		t.Fatalf("command %q does not use standard brew update flow", command)
	}
}

func TestHomebrewUpgradeCommandUsesCaskFromGoReleaser(t *testing.T) {
	command := homebrewUpgradeCommand()
	if command != "brew update && brew upgrade --cask daveshanley/vacuum/vacuum" {
		t.Fatalf("command = %q", command)
	}
}

func TestBrewUpgradeActionRejectsFormulaInstall(t *testing.T) {
	action := brewUpgradeAction(HomebrewKindFormula)
	if action.CanRun {
		t.Fatalf("formula install action should not run: %#v", action)
	}
	if !strings.Contains(action.Reason, "formula installs are not supported") {
		t.Fatalf("reason = %q", action.Reason)
	}
	expected := "brew uninstall --formula vacuum && brew install --cask daveshanley/vacuum/vacuum"
	if action.CommandString() != expected {
		t.Fatalf("CommandString = %q, want %q", action.CommandString(), expected)
	}
}
