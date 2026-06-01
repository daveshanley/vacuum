// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Action struct {
	Method  string
	Command string
	Args    []string
	CanRun  bool
	Reason  string

	Executable    string
	LatestVersion string
	ManualCommand string
}

func (a Action) CommandString() string {
	if a.ManualCommand != "" {
		return a.ManualCommand
	}
	if a.Command == "" {
		return ""
	}
	return strings.Join(append([]string{a.Command}, a.Args...), " ")
}

func PlanUpgrade(ctx InstallContext, latestVersion string) Action {
	switch ctx.Method {
	case MethodNPM:
		return actionIfAvailable(MethodNPM, "npm", []string{"install", "-g", NPMPackageName + "@latest"})
	case MethodHomebrew:
		return brewUpgradeAction(ctx.HomebrewKind)
	case MethodShell:
		return shellInstallerAction(ctx.Executable, latestVersion)
	default:
		return Action{
			Method: MethodManual,
			CanRun: false,
			Reason: "vacuum could not detect whether it was installed with npm, Homebrew, or the shell installer",
		}
	}
}

func brewUpgradeAction(kind string) Action {
	if kind == HomebrewKindFormula {
		return Action{
			Method:        MethodHomebrew,
			CanRun:        false,
			Reason:        "Homebrew formula installs are not supported by automatic upgrade; switch to the supported cask version of vacuum",
			ManualCommand: "brew uninstall --formula " + BrewCaskToken + " && brew install --cask " + BrewCaskFullToken,
		}
	}
	if _, err := exec.LookPath("brew"); err != nil {
		return Action{
			Method: MethodHomebrew,
			CanRun: false,
			Reason: "brew is not available on PATH",
		}
	}
	command := homebrewUpgradeCommand()
	return Action{
		Method:        MethodHomebrew,
		Command:       "sh",
		Args:          []string{"-c", command},
		CanRun:        true,
		ManualCommand: command,
	}
}

func homebrewUpgradeCommand() string {
	return "brew update && brew upgrade --cask " + BrewCaskFullToken
}

func actionIfAvailable(method, command string, args []string) Action {
	_, err := exec.LookPath(command)
	if err != nil {
		return Action{
			Method: method,
			CanRun: false,
			Reason: fmt.Sprintf("%s is not available on PATH", command),
		}
	}
	return Action{
		Method:  method,
		Command: command,
		Args:    args,
		CanRun:  true,
	}
}

func shellInstallerAction(executable, latestVersion string) Action {
	if runtime.GOOS == "windows" {
		return Action{
			Method: MethodShell,
			CanRun: false,
			Reason: "the shell installer is only available on Unix-like systems",
		}
	}
	if executable == "" {
		return Action{
			Method: MethodShell,
			CanRun: false,
			Reason: "vacuum could not resolve the active executable path",
		}
	}

	return Action{
		Method:        MethodShell,
		CanRun:        true,
		Executable:    executable,
		LatestVersion: latestVersion,
		ManualCommand: ManualCommandSet(latestVersion).Shell,
	}
}

func RunAction(ctx context.Context, action Action, stdout, stderr io.Writer) error {
	if !action.CanRun {
		return errors.New(action.Reason)
	}
	if action.Method == MethodShell {
		return RunShellArchiveUpgrade(ctx, action, stdout, stderr)
	}
	cmd := exec.CommandContext(ctx, action.Command, action.Args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func ManualCommands(latestVersion string) []string {
	commands := ManualCommandSet(latestVersion)
	return []string{
		commands.Homebrew,
		commands.NPM,
		commands.Shell,
	}
}

type ManualUpgradeCommands struct {
	Homebrew string
	NPM      string
	Shell    string
}

func ManualCommandSet(latestVersion string) ManualUpgradeCommands {
	return ManualUpgradeCommands{
		Homebrew: "brew upgrade --cask " + BrewCaskFullToken,
		NPM:      "npm install -g " + NPMPackageName + "@latest",
		Shell:    "curl -sSL " + ShellInstallerURLForTag(latestVersion) + " | VERSION=" + shellQuote(NormalizeVersion(latestVersion)) + " sh",
	}
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
