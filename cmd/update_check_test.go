package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/upgrade"
	"github.com/spf13/cobra"
)

func TestWaitForUpdateNoticeWaitsAndRenders(t *testing.T) {
	restore := configureUpdateCheckTest(t, 50*time.Millisecond)
	defer restore()

	var out bytes.Buffer
	testCmd := &cobra.Command{Use: "test"}
	testCmd.SetErr(&out)

	StartUpdateCheck(testCmd)
	WaitForUpdateNotice()

	if !strings.Contains(out.String(), "UPDATE AVAILABLE") ||
		!strings.Contains(out.String(), "v0.26.0 -> v0.27.0") {
		t.Fatalf("expected update notice, got %q", out.String())
	}
}

func TestRootCommandWaitsForUpdateNotice(t *testing.T) {
	restore := configureUpdateCheckTest(t, 50*time.Millisecond)
	defer restore()

	var out bytes.Buffer
	rootCmd := GetRootCommand()
	rootCmd.SetArgs([]string{})
	rootCmd.SetErr(&out)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute returned error: %v", err)
	}
	if !strings.Contains(out.String(), "UPDATE AVAILABLE") ||
		!strings.Contains(out.String(), "v0.26.0 -> v0.27.0") {
		t.Fatalf("expected root command update notice, got %q", out.String())
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

func TestFlushUpdateNoticeDoesNotMarkCanceledCheck(t *testing.T) {
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

	if cache.RecentlyChecked(upgrade.DefaultFailureBackoff) {
		t.Fatalf("canceled in-flight update check marked the failure backoff")
	}
}

func TestWaitForUpdateNoticeMarksCompletedFailure(t *testing.T) {
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

	if !cache.RecentlyChecked(upgrade.DefaultFailureBackoff) {
		t.Fatalf("completed failing update check did not mark the failure backoff")
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

func TestStartUpdateCheckSkipsRecentlyCheckedCacheWithoutRelease(t *testing.T) {
	restore := configureUpdateCheckTest(t, time.Second)
	defer restore()

	now := time.Unix(1700000000, 0)
	cache := &upgrade.UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	if err := cache.MarkChecked(); err != nil {
		t.Fatalf("mark cache checked: %v", err)
	}
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return cache, nil
	}

	testCmd := &cobra.Command{Use: "test"}
	StartUpdateCheck(testCmd)

	if activeUpdateCheck != nil {
		t.Fatalf("expected recently checked cache to skip starting an update check")
	}
}

func TestStartUpdateCheckRetriesAfterFailureBackoff(t *testing.T) {
	restore := configureUpdateCheckTest(t, 0)
	defer restore()

	now := time.Unix(1700000000, 0)
	cache := &upgrade.UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	if err := cache.MarkChecked(); err != nil {
		t.Fatalf("mark cache checked: %v", err)
	}
	cache.Now = func() time.Time { return now.Add(upgrade.DefaultFailureBackoff + time.Second) }
	updateCheckCacheFactory = func() (*upgrade.UpdateCache, error) {
		return cache, nil
	}

	testCmd := &cobra.Command{Use: "test"}
	StartUpdateCheck(testCmd)
	defer FlushUpdateNotice()

	if activeUpdateCheck == nil {
		t.Fatalf("expected expired failure backoff to start a new update check")
	}
}

func TestShouldCheckForUpdatesSkipsCI(t *testing.T) {
	previousIsTerminal := updateCheckIsTerminal
	defer func() { updateCheckIsTerminal = previousIsTerminal }()
	updateCheckIsTerminal = func() bool { return true }
	t.Setenv("CI", "true")

	if ShouldCheckForUpdates(&cobra.Command{Use: "lint"}) {
		t.Fatalf("expected update check to be skipped in CI")
	}
}

func TestShouldCheckForUpdatesSkipsNonTerminal(t *testing.T) {
	previousIsTerminal := updateCheckIsTerminal
	defer func() { updateCheckIsTerminal = previousIsTerminal }()
	updateCheckIsTerminal = func() bool { return false }

	if ShouldCheckForUpdates(&cobra.Command{Use: "lint"}) {
		t.Fatalf("expected update check to be skipped for non-terminal stderr")
	}
}

func configureUpdateCheckTest(t *testing.T, delay time.Duration) func() {
	t.Helper()
	t.Setenv("CI", "")

	previousVersionInfo := versionInfo
	previousOptions := updateCheckOptions
	previousCheck := activeUpdateCheck
	previousCache := activeUpdateCache
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
		activeUpdateNoticeWriter = previousWriter
		updateCheckOptions = previousOptions
		updateCheckCacheFactory = previousCacheFactory
		updateCheckIsTerminal = previousIsTerminal
		versionInfo = previousVersionInfo
		server.Close()
	}
}
