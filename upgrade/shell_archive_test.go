// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunShellArchiveUpgradeDownloadsVerifiesAndInstalls(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is Unix-only")
	}

	assetName, err := releaseAssetName("v0.27.0", runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}
	archiveBytes := testReleaseArchive(t, defaultBinaryName, "#!/bin/sh\necho v0.27.0\n")
	checksum := sha256.Sum256(archiveBytes)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			fmt.Fprintf(w, "%x  %s\n", checksum, assetName)
		case strings.HasSuffix(r.URL.Path, "/"+assetName):
			w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	previousReleaseDownloadBase := releaseDownloadBase
	releaseDownloadBase = server.URL
	defer func() { releaseDownloadBase = previousReleaseDownloadBase }()

	installDir := t.TempDir()
	executable := filepath.Join(installDir, defaultBinaryName)
	writeExecutableScript(t, executable, "#!/bin/sh\necho v0.26.0\n")

	var out bytes.Buffer
	action := Action{
		Method:        MethodShell,
		CanRun:        true,
		Executable:    executable,
		LatestVersion: "v0.27.0",
	}
	if err := RunAction(context.Background(), action, &out, io.Discard); err != nil {
		t.Fatalf("RunAction returned error: %v", err)
	}

	output, err := exec.Command(executable, "version").Output()
	if err != nil {
		t.Fatalf("run installed executable: %v", err)
	}
	if strings.TrimSpace(string(output)) != "v0.27.0" {
		t.Fatalf("installed version = %q, want v0.27.0", strings.TrimSpace(string(output)))
	}
	if _, err := os.Stat(executable + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("backup file should be removed after successful install, stat err=%v", err)
	}
	assertNoInstallTemps(t, installDir)
}

func TestRunShellArchiveUpgradeRejectsChecksumMismatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is Unix-only")
	}

	assetName, err := releaseAssetName("v0.27.0", runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}
	archiveBytes := testReleaseArchive(t, defaultBinaryName, "#!/bin/sh\necho v0.27.0\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			fmt.Fprintf(w, "%064x  %s\n", 0, assetName)
		case strings.HasSuffix(r.URL.Path, "/"+assetName):
			w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	previousReleaseDownloadBase := releaseDownloadBase
	releaseDownloadBase = server.URL
	defer func() { releaseDownloadBase = previousReleaseDownloadBase }()

	installDir := t.TempDir()
	executable := filepath.Join(installDir, defaultBinaryName)
	writeExecutableScript(t, executable, "#!/bin/sh\necho v0.26.0\n")

	action := Action{
		Method:        MethodShell,
		CanRun:        true,
		Executable:    executable,
		LatestVersion: "v0.27.0",
	}
	if err := RunAction(context.Background(), action, io.Discard, io.Discard); err == nil {
		t.Fatalf("RunAction returned nil error for checksum mismatch")
	}

	output, err := exec.Command(executable, "version").Output()
	if err != nil {
		t.Fatalf("run original executable: %v", err)
	}
	if strings.TrimSpace(string(output)) != "v0.26.0" {
		t.Fatalf("original version = %q, want v0.26.0", strings.TrimSpace(string(output)))
	}
}

func TestRunShellArchiveUpgradeRejectsMalformedChecksum(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is Unix-only")
	}

	assetName, err := releaseAssetName("v0.27.0", runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}
	archiveBytes := testReleaseArchive(t, defaultBinaryName, "#!/bin/sh\necho v0.27.0\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			fmt.Fprintf(w, "not-a-sha256  %s\n", assetName)
		case strings.HasSuffix(r.URL.Path, "/"+assetName):
			w.Write(archiveBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	previousReleaseDownloadBase := releaseDownloadBase
	releaseDownloadBase = server.URL
	defer func() { releaseDownloadBase = previousReleaseDownloadBase }()

	_, _, err = downloadVerifiedReleaseBinary(context.Background(), "v0.27.0", defaultBinaryName, io.Discard)
	if err == nil {
		t.Fatalf("downloadVerifiedReleaseBinary returned nil error for malformed checksum")
	}
}

func TestInstallVerifiedReleaseBinaryRejectsConcurrentUpgradeLock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is Unix-only")
	}

	installDir := t.TempDir()
	executable := filepath.Join(installDir, defaultBinaryName)
	writeExecutableScript(t, executable, "#!/bin/sh\necho v0.26.0\n")

	lockPath := filepath.Join(installDir, "."+defaultBinaryName+".upgrade.lock")
	if err := os.WriteFile(lockPath, []byte("locked\n"), 0o600); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	err := installVerifiedReleaseBinary(context.Background(), executable, "v0.27.0", []byte("#!/bin/sh\necho v0.27.0\n"), 0o755, io.Discard)
	if err == nil {
		t.Fatalf("installVerifiedReleaseBinary returned nil error while lock existed")
	}

	output, runErr := exec.Command(executable, "version").Output()
	if runErr != nil {
		t.Fatalf("run original executable: %v", runErr)
	}
	if strings.TrimSpace(string(output)) != "v0.26.0" {
		t.Fatalf("original version = %q, want v0.26.0", strings.TrimSpace(string(output)))
	}
}

func TestAcquireInstallLockReclaimsDeadPID(t *testing.T) {
	alive, ok := processAlive(999999)
	if !ok || alive {
		t.Skip("process liveness probe is unavailable or test PID is alive")
	}

	installDir := t.TempDir()
	lockPath := filepath.Join(installDir, "."+defaultBinaryName+".upgrade.lock")
	if err := os.WriteFile(lockPath, []byte("999999\n"), 0o600); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	lock, err := acquireInstallLock(installDir, defaultBinaryName)
	if err != nil {
		t.Fatalf("acquireInstallLock returned error: %v", err)
	}
	lock.Release()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("lock file should be removed after release, stat err=%v", err)
	}
}

func TestAcquireInstallLockRejectsLivePID(t *testing.T) {
	alive, ok := processAlive(os.Getpid())
	if !ok || !alive {
		t.Skip("process liveness probe is unavailable")
	}

	installDir := t.TempDir()
	lockPath := filepath.Join(installDir, "."+defaultBinaryName+".upgrade.lock")
	if err := os.WriteFile(lockPath, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o600); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	if _, err := acquireInstallLock(installDir, defaultBinaryName); err == nil {
		t.Fatalf("acquireInstallLock returned nil error for live process lock")
	}
}

func TestAcquireInstallLockReclaimsExpiredLock(t *testing.T) {
	installDir := t.TempDir()
	lockPath := filepath.Join(installDir, "."+defaultBinaryName+".upgrade.lock")
	if err := os.WriteFile(lockPath, []byte("not-a-pid\n"), 0o600); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	old := time.Now().Add(-(staleInstallLockMaxAge + time.Second))
	if err := os.Chtimes(lockPath, old, old); err != nil {
		t.Fatalf("age lock: %v", err)
	}

	lock, err := acquireInstallLock(installDir, defaultBinaryName)
	if err != nil {
		t.Fatalf("acquireInstallLock returned error: %v", err)
	}
	lock.Release()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("lock file should be removed after release, stat err=%v", err)
	}
}

func TestInstallVerifiedReleaseBinaryRollsBackOnPostInstallVerificationFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is Unix-only")
	}

	installDir := t.TempDir()
	executable := filepath.Join(installDir, defaultBinaryName)
	writeExecutableScript(t, executable, "#!/bin/sh\necho v0.26.0\n")

	badAfterRename := []byte("#!/bin/sh\ncase \"$0\" in *.tmp.*) echo v0.27.0 ;; *) echo v0.25.0 ;; esac\n")
	err := installVerifiedReleaseBinary(context.Background(), executable, "v0.27.0", badAfterRename, 0o755, io.Discard)
	if err == nil {
		t.Fatalf("installVerifiedReleaseBinary returned nil error")
	}

	output, runErr := exec.Command(executable, "version").Output()
	if runErr != nil {
		t.Fatalf("run restored executable: %v", runErr)
	}
	if strings.TrimSpace(string(output)) != "v0.26.0" {
		t.Fatalf("restored version = %q, want v0.26.0", strings.TrimSpace(string(output)))
	}
	assertNoInstallTemps(t, installDir)
}

func TestChecksumForAsset(t *testing.T) {
	checksum, err := checksumForAsset([]byte("abc123  other.tar.gz\nfed456  vacuum_0.27.0_darwin_arm64.tar.gz\n"), "vacuum_0.27.0_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("checksumForAsset returned error: %v", err)
	}
	if checksum != "fed456" {
		t.Fatalf("checksum = %q, want fed456", checksum)
	}
}

func TestChecksumForAssetRejectsMissingAsset(t *testing.T) {
	if _, err := checksumForAsset([]byte("abc123  other.tar.gz\n"), "vacuum_0.27.0_darwin_arm64.tar.gz"); err == nil {
		t.Fatalf("checksumForAsset returned nil error for missing asset")
	}
}

func TestExtractBinaryFromArchiveRejectsUnsafePath(t *testing.T) {
	archiveBytes := testReleaseArchiveWithHeader(t, tar.Header{
		Name: "../../" + defaultBinaryName,
		Mode: 0o755,
	}, []byte("#!/bin/sh\necho v0.27.0\n"))

	if _, _, err := extractBinaryFromArchive(archiveBytes, defaultBinaryName); err == nil {
		t.Fatalf("extractBinaryFromArchive returned nil error for unsafe archive path")
	}
}

func TestExtractBinaryFromArchiveSkipsSymlink(t *testing.T) {
	archiveBytes := testReleaseArchiveWithHeader(t, tar.Header{
		Name:     defaultBinaryName,
		Mode:     0o755,
		Typeflag: tar.TypeSymlink,
		Linkname: "/tmp/not-vacuum",
	}, nil)

	if _, _, err := extractBinaryFromArchive(archiveBytes, defaultBinaryName); err == nil {
		t.Fatalf("extractBinaryFromArchive returned nil error for symlink entry")
	}
}

func TestDownloadBytesRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "1234567890")
	}))
	defer server.Close()

	_, err := downloadBytes(context.Background(), server.Client(), server.URL, 4)
	if err == nil {
		t.Fatalf("downloadBytes returned nil error for oversized response")
	}
}

func testReleaseArchive(t *testing.T, binaryName, script string) []byte {
	t.Helper()
	return testReleaseArchiveWithHeader(t, tar.Header{
		Name: binaryName,
		Mode: 0o755,
	}, []byte(script))
}

func testReleaseArchiveWithHeader(t *testing.T, header tar.Header, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)
	if header.Typeflag == 0 {
		header.Typeflag = tar.TypeReg
	}
	header.Size = int64(len(data))
	if err := tarWriter.WriteHeader(&header); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if len(data) > 0 {
		if _, err := tarWriter.Write(data); err != nil {
			t.Fatalf("write tar data: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

func assertNoInstallTemps(t *testing.T, installDir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(installDir, "."+defaultBinaryName+".backup.*"))
	if err != nil {
		t.Fatalf("glob backup files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("backup temp files were left behind: %v", matches)
	}
	matches, err = filepath.Glob(filepath.Join(installDir, "."+defaultBinaryName+".upgrade.lock"))
	if err != nil {
		t.Fatalf("glob lock files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("lock file was left behind: %v", matches)
	}
}

func writeExecutableScript(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}
