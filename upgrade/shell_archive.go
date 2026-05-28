// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	releaseDownloadBaseURL = "https://github.com/daveshanley/vacuum/releases/download"
	maxChecksumBytes       = 1 << 20
	maxArchiveBytes        = 200 << 20
	maxBinaryBytes         = 200 << 20
	shellDownloadTimeout   = 2 * time.Minute
	staleInstallLockMaxAge = 15 * time.Minute
)

var releaseDownloadBase = releaseDownloadBaseURL

func RunShellArchiveUpgrade(ctx context.Context, action Action, stdout, stderr io.Writer) error {
	if action.Executable == "" {
		return fmt.Errorf("shell upgrade requires an executable path")
	}
	if action.LatestVersion == "" {
		return fmt.Errorf("shell upgrade requires a target version")
	}

	archive, mode, err := downloadVerifiedReleaseBinary(ctx, action.LatestVersion, filepath.Base(action.Executable), stdout)
	if err != nil {
		return err
	}
	return installVerifiedReleaseBinary(ctx, action.Executable, action.LatestVersion, archive, mode, stdout)
}

func downloadVerifiedReleaseBinary(ctx context.Context, tag, binaryName string, stdout io.Writer) ([]byte, os.FileMode, error) {
	assetName, err := releaseAssetName(tag, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, 0, err
	}
	checksumURL := releaseChecksumURL(tag)
	archiveURL := releaseArchiveURL(tag, assetName)

	client := &http.Client{Timeout: shellDownloadTimeout}
	fmt.Fprintf(stdout, "Downloading %s\n", archiveURL)
	archiveBytes, err := downloadBytes(ctx, client, archiveURL, maxArchiveBytes)
	if err != nil {
		return nil, 0, err
	}

	fmt.Fprintf(stdout, "Verifying %s\n", checksumURL)
	checksumBytes, err := downloadBytes(ctx, client, checksumURL, maxChecksumBytes)
	if err != nil {
		return nil, 0, err
	}
	expectedChecksum, err := checksumForAsset(checksumBytes, assetName)
	if err != nil {
		return nil, 0, err
	}
	actualChecksum := sha256.Sum256(archiveBytes)
	expectedChecksumBytes, err := hex.DecodeString(strings.TrimSpace(expectedChecksum))
	if err != nil || len(expectedChecksumBytes) != sha256.Size {
		return nil, 0, fmt.Errorf("checksum for %s is not a valid SHA256 digest", assetName)
	}
	if subtle.ConstantTimeCompare(actualChecksum[:], expectedChecksumBytes) != 1 {
		return nil, 0, fmt.Errorf("checksum mismatch for %s", assetName)
	}

	return extractBinaryFromArchive(archiveBytes, binaryName)
}

func installVerifiedReleaseBinary(ctx context.Context, executable, latestVersion string, binary []byte, mode os.FileMode, stdout io.Writer) error {
	installDir := filepath.Dir(executable)
	binaryName := filepath.Base(executable)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}
	lock, err := acquireInstallLock(installDir, binaryName)
	if err != nil {
		return err
	}
	defer lock.Release()

	tmpFile, err := os.CreateTemp(installDir, "."+binaryName+".tmp.")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(binary); err != nil {
		tmpFile.Close()
		return err
	}
	if mode == 0 {
		mode = 0o755
	}
	if err := tmpFile.Chmod(mode | 0o700); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := verifyBinaryVersion(ctx, tmpPath, latestVersion); err != nil {
		return err
	}

	backupPath := ""
	backupCreated := false
	if _, err := os.Stat(executable); err == nil {
		backupFile, err := os.CreateTemp(installDir, "."+binaryName+".backup.")
		if err != nil {
			return err
		}
		backupPath = backupFile.Name()
		if err := backupFile.Close(); err != nil {
			return err
		}
		if err := copyFile(executable, backupPath); err != nil {
			_ = os.Remove(backupPath)
			return fmt.Errorf("backup existing binary: %w", err)
		}
		backupCreated = true
	} else if !os.IsNotExist(err) {
		return err
	}

	installed := false
	if err := os.Rename(tmpPath, executable); err != nil {
		return err
	}
	installed = true
	defer func() {
		if installed && backupCreated {
			_ = os.Remove(backupPath)
		}
	}()

	if err := verifyBinaryVersion(ctx, executable, latestVersion); err != nil {
		if backupCreated {
			_ = os.Rename(backupPath, executable)
			installed = false
		} else {
			_ = os.Remove(executable)
		}
		return err
	}

	fmt.Fprintf(stdout, "Installed vacuum to %s\n", executable)
	return nil
}

func releaseAssetName(tag, goos, goarch string) (string, error) {
	version := NormalizeVersion(tag)
	if version == "" {
		return "", fmt.Errorf("release version is empty")
	}
	arch, err := releaseArch(goarch)
	if err != nil {
		return "", err
	}
	switch goos {
	case "darwin", "linux":
		return fmt.Sprintf("vacuum_%s_%s_%s.tar.gz", version, goos, arch), nil
	default:
		return "", fmt.Errorf("%s is not supported by the shell upgrade installer", goos)
	}
}

func releaseArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "x86_64", nil
	case "386":
		return "i386", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("%s is not a supported release architecture", goarch)
	}
}

func releaseArchiveURL(tag, assetName string) string {
	return strings.TrimRight(releaseDownloadBase, "/") + "/" + tag + "/" + assetName
}

func releaseChecksumURL(tag string) string {
	return strings.TrimRight(releaseDownloadBase, "/") + "/" + tag + "/checksums.txt"
}

func checksumForAsset(checksums []byte, assetName string) (string, error) {
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[len(fields)-1] == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s was not found", assetName)
}

func downloadBytes(ctx context.Context, client *http.Client, url string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "vacuum-upgrade")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("%s returned status %s", url, resp.Status)
	}
	reader := io.LimitReader(resp.Body, maxBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%s exceeded %d bytes", url, maxBytes)
	}
	return data, nil
}

func extractBinaryFromArchive(archiveBytes []byte, binaryName string) ([]byte, os.FileMode, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archiveBytes))
	if err != nil {
		return nil, 0, err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		if header.Typeflag != tar.TypeReg || !safeArchiveBinaryName(header.Name, binaryName) {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(tarReader, maxBinaryBytes+1))
		if err != nil {
			return nil, 0, err
		}
		if len(data) > maxBinaryBytes {
			return nil, 0, fmt.Errorf("%s exceeded %d bytes", binaryName, maxBinaryBytes)
		}
		return data, os.FileMode(header.Mode).Perm(), nil
	}
	return nil, 0, fmt.Errorf("%s was not found in release archive", binaryName)
}

func safeArchiveBinaryName(headerName, binaryName string) bool {
	normalized := path.Clean(strings.ReplaceAll(headerName, "\\", "/"))
	if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") || strings.HasPrefix(normalized, "/") {
		return false
	}
	return path.Base(normalized) == binaryName
}

type installLock struct {
	file *os.File
	path string
}

func acquireInstallLock(installDir, binaryName string) (*installLock, error) {
	lockPath := filepath.Join(installDir, "."+binaryName+".upgrade.lock")
	lock, err := createInstallLock(lockPath)
	if err == nil {
		return lock, nil
	}
	if !os.IsExist(err) {
		return nil, err
	}
	if removeStaleInstallLock(lockPath, staleInstallLockMaxAge) {
		lock, err = createInstallLock(lockPath)
		if err == nil {
			return lock, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("another vacuum upgrade is already running")
}

func createInstallLock(lockPath string) (*installLock, error) {
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
		file.Close()
		_ = os.Remove(lockPath)
		return nil, err
	}
	return &installLock{file: file, path: lockPath}, nil
}

func removeStaleInstallLock(lockPath string, maxAge time.Duration) bool {
	info, err := os.Stat(lockPath)
	if err != nil {
		return os.IsNotExist(err)
	}
	if maxAge > 0 && time.Since(info.ModTime()) > maxAge {
		return removeInstallLock(lockPath)
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	alive, ok := processAlive(pid)
	if !ok || alive {
		return false
	}
	return removeInstallLock(lockPath)
}

func removeInstallLock(lockPath string) bool {
	err := os.Remove(lockPath)
	return err == nil || os.IsNotExist(err)
}

func (l *installLock) Release() {
	if l == nil {
		return
	}
	if l.file != nil {
		_ = l.file.Close()
	}
	if l.path != "" {
		_ = os.Remove(l.path)
	}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, srcInfo.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		return err
	}
	if err := dstFile.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode().Perm())
}
