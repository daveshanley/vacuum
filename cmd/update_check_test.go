package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/upgrade"
	"github.com/spf13/cobra"
)

func TestWaitForUpdateNoticeWaitsAndCaches(t *testing.T) {
	restore := configureUpdateCheckTest(t, 50*time.Millisecond)
	defer restore()

	now := time.Unix(1700000000, 0)
	cache := &upgrade.UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return cache, nil
	}

	var out bytes.Buffer
	testCmd := &cobra.Command{Use: "test"}
	testCmd.SetErr(&out)

	StartUpdateCheck(testCmd)
	WaitForUpdateNotice()

	if out.String() != "" {
		t.Fatalf("expected first network result to be cached silently, got %q", out.String())
	}
	result, ok := cache.ReadFresh("v0.26.0", upgrade.DefaultCacheMaxAge)
	if !ok {
		t.Fatalf("expected completed update check to populate cache")
	}
	if result.LatestVersion != "v0.27.0" || result.ReleaseURL != "https://example.com/release" {
		t.Fatalf("cached result = %#v", result)
	}
}

func TestRootCommandWaitsForUpdateNoticeAndCachesSilently(t *testing.T) {
	restore := configureUpdateCheckTest(t, 50*time.Millisecond)
	defer restore()

	now := time.Unix(1700000000, 0)
	cache := &upgrade.UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return cache, nil
	}

	var out bytes.Buffer
	rootCmd := GetRootCommand()
	rootCmd.SetArgs([]string{})
	rootCmd.SetErr(&out)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute returned error: %v", err)
	}
	if strings.Contains(out.String(), "UPDATE AVAILABLE") {
		t.Fatalf("expected root command first update check to be silent, got %q", out.String())
	}
	if _, ok := cache.ReadFresh("v0.26.0", upgrade.DefaultCacheMaxAge); !ok {
		t.Fatalf("expected root command to cache completed update check")
	}
}

func TestFlushUpdateNoticeDoesNotWait(t *testing.T) {
	restore := configureUpdateCheckTest(t, 200*time.Millisecond)
	defer restore()

	var out bytes.Buffer
	testCmd := &cobra.Command{Use: "test"}
	testCmd.SetErr(&out)

	StartUpdateCheck(testCmd)
	start := time.Now()
	FlushUpdateNotice()

	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("FlushUpdateNotice waited %s", elapsed)
	}
	if out.String() != "" {
		t.Fatalf("expected no notice before check completed, got %q", out.String())
	}
}

func TestFlushUpdateNoticeDoesNotWriteCanceledCheck(t *testing.T) {
	restore := configureUpdateCheckTest(t, 200*time.Millisecond)
	defer restore()

	now := time.Unix(1700000000, 0)
	cache := &upgrade.UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return cache, nil
	}

	testCmd := &cobra.Command{Use: "test"}
	StartUpdateCheck(testCmd)
	FlushUpdateNotice()

	if _, err := os.Stat(cache.Path); !os.IsNotExist(err) {
		t.Fatalf("canceled in-flight update check wrote cache file, stat error: %v", err)
	}
}

func TestWaitForUpdateNoticeDoesNotCacheCompletedFailure(t *testing.T) {
	restore := configureUpdateCheckTest(t, 0)
	defer restore()

	now := time.Unix(1700000000, 0)
	cache := &upgrade.UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return cache, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	updateCheckOptions = upgrade.CheckOptions{
		LatestReleaseURL: server.URL,
		Timeout:          time.Second,
	}

	testCmd := &cobra.Command{Use: "test"}
	StartUpdateCheck(testCmd)
	WaitForUpdateNotice()

	if _, err := os.Stat(cache.Path); !os.IsNotExist(err) {
		t.Fatalf("completed failing update check wrote cache file, stat error: %v", err)
	}
}

func TestStartUpdateCheckUsesFreshCache(t *testing.T) {
	restore := configureUpdateCheckTest(t, time.Second)
	defer restore()

	now := time.Unix(1700000000, 0)
	cache := &upgrade.UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	if err := cache.Write(upgrade.CheckResult{
		LatestVersion: "v0.27.0",
		ReleaseURL:    "https://example.com/cached",
	}); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return cache, nil
	}
	updateCheckOptions = upgrade.CheckOptions{
		LatestReleaseURL: "http://127.0.0.1:1",
		Timeout:          time.Millisecond,
	}

	var out bytes.Buffer
	testCmd := &cobra.Command{Use: "test"}
	testCmd.SetErr(&out)

	StartUpdateCheck(testCmd)
	WaitForUpdateNotice()

	if !strings.Contains(out.String(), "UPDATE AVAILABLE") ||
		!strings.Contains(out.String(), "https://example.com/cached") {
		t.Fatalf("expected cached update notice, got %q", out.String())
	}
}

func TestStartUpdateCheckSkipsFreshCacheRefresh(t *testing.T) {
	restore := configureUpdateCheckTest(t, time.Second)
	defer restore()

	now := time.Unix(1700000000, 0)
	cache := &upgrade.UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	if err := cache.Write(upgrade.CheckResult{
		LatestVersion: "v0.27.0",
		ReleaseURL:    "https://example.com/cached",
	}); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return cache, nil
	}

	var out bytes.Buffer
	testCmd := &cobra.Command{Use: "test"}
	testCmd.SetErr(&out)
	StartUpdateCheck(testCmd)

	if activeUpdateCheck != nil {
		t.Fatalf("expected fresh cache to skip starting an update check")
	}
	WaitForUpdateNotice()
}

func TestStartUpdateCheckRefreshesStaleCache(t *testing.T) {
	restore := configureUpdateCheckTest(t, 0)
	defer restore()

	now := time.Unix(1700000000, 0)
	cache := &upgrade.UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	if err := cache.Write(upgrade.CheckResult{
		LatestVersion: "v0.27.0",
		ReleaseURL:    "https://example.com/cached",
	}); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	cache.Now = func() time.Time { return now.Add(upgrade.DefaultCacheMaxAge + time.Second) }
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return cache, nil
	}

	var out bytes.Buffer
	testCmd := &cobra.Command{Use: "test"}
	testCmd.SetErr(&out)
	StartUpdateCheck(testCmd)

	if activeUpdateCheck == nil {
		t.Fatalf("expected stale cache to start a new update check")
	}
	WaitForUpdateNotice()
	if cache.ShouldRefresh(upgrade.DefaultCacheMaxAge) {
		t.Fatalf("completed stale-cache refresh did not update cache mtime")
	}
}

func TestWaitForUpdateNoticeDoesNotMakeStaleCacheFreshAfterFailure(t *testing.T) {
	restore := configureUpdateCheckTest(t, 0)
	defer restore()

	now := time.Unix(1700000000, 0)
	cache := &upgrade.UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	if err := cache.Write(upgrade.CheckResult{
		LatestVersion: "v0.27.0",
		ReleaseURL:    "https://example.com/cached",
	}); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	cache.Now = func() time.Time { return now.Add(upgrade.DefaultCacheMaxAge + time.Second) }
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return cache, nil
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	updateCheckOptions = upgrade.CheckOptions{
		LatestReleaseURL: server.URL,
		Timeout:          time.Second,
	}

	var out bytes.Buffer
	testCmd := &cobra.Command{Use: "test"}
	testCmd.SetErr(&out)
	StartUpdateCheck(testCmd)
	WaitForUpdateNotice()

	if !cache.ShouldRefresh(upgrade.DefaultCacheMaxAge) {
		t.Fatalf("failed refresh made stale cache appear fresh")
	}
}

func TestShouldCheckForUpdatesSkipMatrix(t *testing.T) {
	previousIsTerminal := updateCheckIsTerminal
	defer func() { updateCheckIsTerminal = previousIsTerminal }()

	tests := []struct {
		name       string
		cmd        *cobra.Command
		ci         string
		isTerminal bool
		want       bool
	}{
		{name: "lint terminal", cmd: &cobra.Command{Use: "lint"}, isTerminal: true, want: true},
		{name: "ci", cmd: &cobra.Command{Use: "lint"}, ci: "true", isTerminal: true, want: false},
		{name: "non terminal", cmd: &cobra.Command{Use: "lint"}, isTerminal: false, want: false},
		{name: "hidden command", cmd: &cobra.Command{Use: "__complete"}, isTerminal: true, want: false},
		{name: "version command", cmd: &cobra.Command{Use: "version"}, isTerminal: true, want: false},
		{name: "completion command", cmd: &cobra.Command{Use: "completion"}, isTerminal: true, want: false},
		{name: "help command", cmd: &cobra.Command{Use: "help"}, isTerminal: true, want: false},
		{name: "language server command", cmd: &cobra.Command{Use: "language-server"}, isTerminal: true, want: false},
		{name: "upgrade command", cmd: &cobra.Command{Use: "upgrade"}, isTerminal: true, want: false},
		{name: "no update check flag", cmd: boolFlagCommand("lint", "no-update-check"), isTerminal: true, want: false},
		{name: "stdout flag", cmd: boolFlagCommand("lint", "stdout"), isTerminal: true, want: false},
		{name: "silent flag", cmd: boolFlagCommand("lint", "silent"), isTerminal: true, want: false},
		{name: "pipeline output flag", cmd: boolFlagCommand("lint", "pipeline-output"), isTerminal: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CI", tt.ci)
			updateCheckIsTerminal = func() bool { return tt.isTerminal }
			if got := ShouldCheckForUpdates(tt.cmd); got != tt.want {
				t.Fatalf("ShouldCheckForUpdates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func boolFlagCommand(use, name string) *cobra.Command {
	cmd := &cobra.Command{Use: use}
	cmd.Flags().Bool(name, true, "")
	return cmd
}

func configureUpdateCheckTest(t *testing.T, delay time.Duration) func() {
	t.Helper()
	t.Setenv("CI", "")

	previousVersionInfo := versionInfo
	previousOptions := updateCheckOptions
	previousCheck := activeUpdateCheck
	previousCache := activeUpdateCache
	previousCachedNotice := activeCachedUpdateNotice
	previousWriter := activeUpdateNoticeWriter
	previousCacheFactory := updateCheckCacheFactory
	previousIsTerminal := updateCheckIsTerminal

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		fmt.Fprint(w, `{"tag_name":"v0.27.0","html_url":"https://example.com/release","draft":false,"prerelease":false}`)
	}))

	versionInfo = VersionInfo{Version: "v0.26.0"}
	updateCheckOptions = upgrade.CheckOptions{
		LatestReleaseURL: server.URL,
		Timeout:          time.Second,
	}
	activeUpdateCheck = nil
	activeUpdateCache = nil
	activeCachedUpdateNotice = nil
	activeUpdateNoticeWriter = nil
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return &upgrade.UpdateCache{Path: filepath.Join(t.TempDir(), "update-check.json")}, nil
	}
	updateCheckIsTerminal = func() bool { return true }

	return func() {
		if activeUpdateCheck != nil {
			activeUpdateCheck.Cancel()
		}
		activeUpdateCheck = previousCheck
		activeUpdateCache = previousCache
		activeCachedUpdateNotice = previousCachedNotice
		activeUpdateNoticeWriter = previousWriter
		updateCheckOptions = previousOptions
		updateCheckCacheFactory = previousCacheFactory
		updateCheckIsTerminal = previousIsTerminal
		versionInfo = previousVersionInfo
		server.Close()
	}
}
