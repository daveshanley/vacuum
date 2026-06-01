// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectInstallContextFromNPM(t *testing.T) {
	getenv := func(key string) string {
		switch key {
		case EnvManagedByNPM:
			return "1"
		case EnvManagedRoot:
			return "/usr/local/lib/node_modules/@quobix/vacuum"
		default:
			return ""
		}
	}

	ctx := DetectInstallContextFrom("/tmp/vacuum", getenv)
	if ctx.Method != MethodNPM {
		t.Fatalf("Method = %q, want %q", ctx.Method, MethodNPM)
	}
	if ctx.PackageRoot == "" {
		t.Fatalf("PackageRoot was empty")
	}
}

func TestDetectInstallContextFromHomebrewCaskroom(t *testing.T) {
	ctx := DetectInstallContextFrom("/opt/homebrew/Caskroom/vacuum/0.27.0/vacuum", nil)
	if ctx.Method != MethodHomebrew {
		t.Fatalf("Method = %q, want %q", ctx.Method, MethodHomebrew)
	}
	if ctx.HomebrewKind != HomebrewKindCask {
		t.Fatalf("HomebrewKind = %q, want %q", ctx.HomebrewKind, HomebrewKindCask)
	}
}

func TestDetectInstallContextFromCustomHomebrewCaskroom(t *testing.T) {
	ctx := DetectInstallContextFrom("/custom/prefix/Caskroom/vacuum/0.27.0/vacuum", nil)
	if ctx.Method != MethodHomebrew {
		t.Fatalf("Method = %q, want %q", ctx.Method, MethodHomebrew)
	}
	if ctx.HomebrewKind != HomebrewKindCask {
		t.Fatalf("HomebrewKind = %q, want %q", ctx.HomebrewKind, HomebrewKindCask)
	}
}

func TestDetectInstallContextFromHomebrewCellarFormula(t *testing.T) {
	ctx := DetectInstallContextFrom("/opt/homebrew/Cellar/vacuum/0.28.0/bin/vacuum", nil)
	if ctx.Method != MethodHomebrew {
		t.Fatalf("Method = %q, want %q", ctx.Method, MethodHomebrew)
	}
	if ctx.HomebrewKind != HomebrewKindFormula {
		t.Fatalf("HomebrewKind = %q, want %q", ctx.HomebrewKind, HomebrewKindFormula)
	}
}

func TestDetectInstallContextRejectsSourceCheckoutBinary(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/daveshanley/vacuum\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	ctx := DetectInstallContextFrom(filepath.Join(dir, "vacuum"), func(string) string { return "" })
	if ctx.Method != MethodUnknown {
		t.Fatalf("Method = %q, want %q", ctx.Method, MethodUnknown)
	}
}

func TestDetectInstallContextRejectsGoInstalledBinary(t *testing.T) {
	home := t.TempDir()
	getenv := func(key string) string {
		if key == "HOME" {
			return home
		}
		return ""
	}

	ctx := DetectInstallContextFrom(filepath.Join(home, "go", "bin", "vacuum"), getenv)
	if ctx.Method != MethodUnknown {
		t.Fatalf("Method = %q, want %q", ctx.Method, MethodUnknown)
	}
}
