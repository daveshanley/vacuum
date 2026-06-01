// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type brewInfo struct {
	Casks []struct {
		Token   string `json:"token"`
		Version string `json:"version"`
	} `json:"casks"`
}

func VerifyUpgrade(ctx context.Context, installContext InstallContext, method, latestVersion string) error {
	switch method {
	case MethodHomebrew:
		if installContext.HomebrewKind == HomebrewKindFormula {
			return fmt.Errorf("Homebrew formula installs are not supported by automatic upgrade; switch to the supported cask version of vacuum with: brew uninstall --formula %s && brew install --cask %s", BrewCaskToken, BrewCaskFullToken)
		}
		version, err := HomebrewCaskVersion(ctx)
		if err != nil {
			return fmt.Errorf("verify Homebrew cask version: %w", err)
		}
		if cmp, ok := CompareVersions(version, latestVersion); !ok || cmp < 0 {
			return fmt.Errorf("Homebrew cask %s is at %s, but GitHub latest is %s; the tap may not be updated yet",
				BrewCaskFullToken, version, latestVersion)
		}
		return nil
	case MethodNPM:
		path, err := exec.LookPath(defaultBinaryName)
		if err != nil {
			return fmt.Errorf("find upgraded vacuum on PATH: %w", err)
		}
		return verifyBinaryVersion(ctx, path, latestVersion)
	case MethodShell:
		if installContext.Executable == "" {
			return fmt.Errorf("shell install context did not include an executable path")
		}
		return verifyBinaryVersion(ctx, installContext.Executable, latestVersion)
	default:
		return nil
	}
}

func HomebrewCaskVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "brew", "info", "--cask", "--json=v2", BrewCaskToken)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return parseHomebrewCaskVersion(output)
}

func parseHomebrewCaskVersion(output []byte) (string, error) {
	var info brewInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return "", err
	}
	for _, cask := range info.Casks {
		if cask.Token == BrewCaskToken && cask.Version != "" {
			return cask.Version, nil
		}
	}
	return "", fmt.Errorf("cask %s was not found in brew info output", BrewCaskToken)
}

func verifyBinaryVersion(ctx context.Context, executable, latestVersion string) error {
	cmd := exec.CommandContext(ctx, executable, "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("run %s version: %w", executable, err)
	}
	version := strings.TrimSpace(string(output))
	if cmp, ok := CompareVersions(version, latestVersion); !ok || cmp < 0 {
		return fmt.Errorf("%s reports %s, but GitHub latest is %s", executable, version, latestVersion)
	}
	return nil
}
