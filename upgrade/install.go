// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	EnvManagedByNPM     = "VACUUM_MANAGED_BY_NPM"
	EnvManagedRoot      = "VACUUM_MANAGED_PACKAGE_ROOT"
	MethodNPM           = "npm"
	MethodHomebrew      = "homebrew"
	MethodShell         = "shell"
	MethodManual        = "manual"
	MethodUnknown       = "unknown"
	HomebrewKindCask    = "cask"
	HomebrewKindFormula = "formula"
	BrewCaskToken       = "vacuum"
	BrewCaskFullToken   = "daveshanley/vacuum/vacuum"
	NPMPackageName      = "@quobix/vacuum"
	shellInstallerURL   = "https://raw.githubusercontent.com/daveshanley/vacuum/%s/bin/install.sh"
	defaultBinaryName   = "vacuum"
	windowsBinarySuffix = ".exe"
	maxSourceCheckDepth = 8
)

type InstallContext struct {
	Method       string
	Executable   string
	PackageRoot  string
	HomebrewKind string
	Reason       string
}

func DetectInstallContext() InstallContext {
	exe, _ := os.Executable()
	return DetectInstallContextFrom(exe, os.Getenv)
}

func DetectInstallContextFrom(executable string, getenv func(string) string) InstallContext {
	if getenv == nil {
		getenv = os.Getenv
	}
	if getenv(EnvManagedByNPM) != "" {
		return InstallContext{
			Method:      MethodNPM,
			Executable:  executable,
			PackageRoot: getenv(EnvManagedRoot),
			Reason:      "npm shim set " + EnvManagedByNPM,
		}
	}

	resolved := executable
	if executable != "" {
		if realPath, err := filepath.EvalSymlinks(executable); err == nil {
			resolved = realPath
		}
	}
	if kind := homebrewInstallKind(resolved); kind != "" {
		return InstallContext{
			Method:       MethodHomebrew,
			Executable:   resolved,
			HomebrewKind: kind,
			Reason:       "executable resolved under Homebrew " + kind,
		}
	}

	if isGoManagedBinary(resolved, getenv) || isInsideVacuumSourceCheckout(resolved) {
		return InstallContext{
			Method:     MethodUnknown,
			Executable: resolved,
			Reason:     "executable appears to be a Go-installed or source checkout binary",
		}
	}

	if runtime.GOOS != "windows" && looksLikeVacuumBinary(resolved) {
		return InstallContext{
			Method:     MethodShell,
			Executable: resolved,
			Reason:     "standalone vacuum binary",
		}
	}

	return InstallContext{
		Method:     MethodUnknown,
		Executable: resolved,
		Reason:     "installation method could not be detected",
	}
}

func looksLikeVacuumBinary(path string) bool {
	name := filepath.Base(path)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(name, defaultBinaryName+windowsBinarySuffix)
	}
	return name == defaultBinaryName
}

func homebrewInstallKind(path string) string {
	if path == "" {
		return ""
	}
	dir := filepath.Clean(filepath.Dir(path))
	for {
		if filepath.Base(dir) == defaultBinaryName && filepath.Base(filepath.Dir(dir)) == "Caskroom" {
			return HomebrewKindCask
		}
		if filepath.Base(dir) == defaultBinaryName && filepath.Base(filepath.Dir(dir)) == "Cellar" {
			return HomebrewKindFormula
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func ShellInstallerURLForTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		tag = "main"
	}
	return fmt.Sprintf(shellInstallerURL, tag)
}

func isGoManagedBinary(executable string, getenv func(string) string) bool {
	if executable == "" {
		return false
	}
	dir := filepath.Clean(filepath.Dir(executable))
	if gobin := getenv("GOBIN"); gobin != "" && samePath(dir, gobin) {
		return true
	}

	gopath := getenv("GOPATH")
	if gopath == "" {
		home := getenv("HOME")
		if home == "" {
			return false
		}
		gopath = filepath.Join(home, "go")
	}
	for _, path := range filepath.SplitList(gopath) {
		if path == "" {
			continue
		}
		if samePath(dir, filepath.Join(path, "bin")) {
			return true
		}
	}
	return false
}

func isInsideVacuumSourceCheckout(executable string) bool {
	if executable == "" {
		return false
	}
	dir := filepath.Clean(filepath.Dir(executable))
	for depth := 0; depth < maxSourceCheckDepth; depth++ {
		goMod := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(goMod); err == nil {
			return strings.Contains(string(data), "module github.com/daveshanley/vacuum")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
	return false
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}
