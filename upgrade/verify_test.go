// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("VACUUM_VERIFY_HELPER") == "1" {
		fmt.Println(os.Getenv("VACUUM_VERIFY_VERSION"))
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestParseHomebrewCaskVersion(t *testing.T) {
	version, err := parseHomebrewCaskVersion([]byte(`{"casks":[{"token":"vacuum","version":"0.27.0"}]}`))
	if err != nil {
		t.Fatalf("parseHomebrewCaskVersion returned error: %v", err)
	}
	if version != "0.27.0" {
		t.Fatalf("version = %q, want 0.27.0", version)
	}
}

func TestVerifyUpgradeRejectsHomebrewFormula(t *testing.T) {
	err := VerifyUpgrade(context.Background(), InstallContext{HomebrewKind: HomebrewKindFormula}, MethodHomebrew, "v0.27.0")
	if err == nil {
		t.Fatalf("VerifyUpgrade returned nil error for formula install")
	}
	if !strings.Contains(err.Error(), "switch to the supported cask version of vacuum") {
		t.Fatalf("error did not tell user to switch to the cask: %v", err)
	}
	if !strings.Contains(err.Error(), "brew uninstall --formula vacuum && brew install --cask daveshanley/vacuum/vacuum") {
		t.Fatalf("error did not include cask switch command: %v", err)
	}
}

func TestVerifyBinaryVersion(t *testing.T) {
	t.Setenv("VACUUM_VERIFY_HELPER", "1")
	t.Setenv("VACUUM_VERIFY_VERSION", "v0.27.0")

	if err := verifyBinaryVersion(context.Background(), os.Args[0], "v0.27.0"); err != nil {
		t.Fatalf("verifyBinaryVersion returned error: %v", err)
	}
}

func TestVerifyBinaryVersionRejectsStaleVersion(t *testing.T) {
	t.Setenv("VACUUM_VERIFY_HELPER", "1")
	t.Setenv("VACUUM_VERIFY_VERSION", "v0.26.0")

	if err := verifyBinaryVersion(context.Background(), os.Args[0], "v0.27.0"); err == nil {
		t.Fatalf("verifyBinaryVersion returned nil error for stale version")
	}
}
