// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestUpdateCacheReadFresh(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cache := &UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}

	if err := cache.Write(CheckResult{
		LatestVersion: "v0.27.0",
		ReleaseURL:    "https://example.com/release",
	}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	result, ok := cache.ReadFresh("v0.26.0", DefaultCacheMaxAge)
	if !ok {
		t.Fatalf("ReadFresh did not return cached result")
	}
	if result.CurrentVersion != "v0.26.0" ||
		result.LatestVersion != "v0.27.0" ||
		result.ReleaseURL != "https://example.com/release" {
		t.Fatalf("cached result = %#v", result)
	}
}

func TestUpdateCacheRejectsStaleResult(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cache := &UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}

	if err := cache.Write(CheckResult{LatestVersion: "v0.27.0"}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	cache.Now = func() time.Time { return now.Add(DefaultCacheMaxAge + time.Second) }

	if _, ok := cache.ReadFresh("v0.26.0", DefaultCacheMaxAge); ok {
		t.Fatalf("ReadFresh returned a stale cached result")
	}
}

func TestUpdateCacheTracksRecentAttemptWithoutRelease(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cache := &UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}

	if err := cache.MarkChecked(); err != nil {
		t.Fatalf("MarkChecked returned error: %v", err)
	}
	if !cache.RecentlyChecked(DefaultCacheMaxAge) {
		t.Fatalf("RecentlyChecked returned false for fresh attempted check")
	}
	cache.Now = func() time.Time { return now.Add(DefaultFailureBackoff + time.Second) }
	if cache.RecentlyChecked(DefaultFailureBackoff) {
		t.Fatalf("RecentlyChecked returned true outside failure backoff")
	}
	cache.Now = func() time.Time { return now }
	if _, ok := cache.ReadFresh("v0.26.0", DefaultCacheMaxAge); ok {
		t.Fatalf("ReadFresh returned release data for attempt-only cache")
	}
}

func TestUpdateCacheMarkCheckedPreservesReleaseData(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cache := &UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}
	if err := cache.Write(CheckResult{
		LatestVersion: "v0.27.0",
		ReleaseURL:    "https://example.com/release",
	}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	cache.Now = func() time.Time { return now.Add(DefaultCacheMaxAge + time.Second) }
	if err := cache.MarkChecked(); err != nil {
		t.Fatalf("MarkChecked returned error: %v", err)
	}

	result, ok := cache.ReadFresh("v0.26.0", DefaultCacheMaxAge)
	if !ok {
		t.Fatalf("ReadFresh did not return preserved release data")
	}
	if result.LatestVersion != "v0.27.0" || result.ReleaseURL != "https://example.com/release" {
		t.Fatalf("cached result = %#v", result)
	}
}

func TestUpdateCacheWritesAtomically(t *testing.T) {
	cache := &UpdateCache{Path: filepath.Join(t.TempDir(), "update-check.json")}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = cache.Write(CheckResult{
				LatestVersion: fmt.Sprintf("v0.27.%d", i),
				ReleaseURL:    "https://example.com/release",
			})
		}(i)
	}
	wg.Wait()

	cached, ok := cache.read()
	if !ok {
		t.Fatalf("cache file was not readable after concurrent writes")
	}
	if cached.LatestVersion == "" {
		t.Fatalf("cached latest version was empty")
	}

	matches, err := filepath.Glob(cache.Path + ".tmp.*")
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary cache files were left behind: %v", matches)
	}
}

func TestDefaultUpdateCacheUsesXDGConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	cache, err := DefaultUpdateCache()
	if err != nil {
		t.Fatalf("DefaultUpdateCache returned error: %v", err)
	}
	want := filepath.Join(configHome, "vacuum", ".vacuum-update-check.json")
	if cache.Path != want {
		t.Fatalf("Path = %q, want %q", cache.Path, want)
	}
}

func TestDefaultUpdateCacheUsesHomeConfigFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	previousXDG, hadXDG := os.LookupEnv("XDG_CONFIG_HOME")
	if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
		t.Fatalf("unset XDG_CONFIG_HOME: %v", err)
	}
	t.Cleanup(func() {
		if hadXDG {
			_ = os.Setenv("XDG_CONFIG_HOME", previousXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	})

	cache, err := DefaultUpdateCache()
	if err != nil {
		t.Fatalf("DefaultUpdateCache returned error: %v", err)
	}
	want := filepath.Join(home, ".config", "vacuum", ".vacuum-update-check.json")
	if cache.Path != want {
		t.Fatalf("Path = %q, want %q", cache.Path, want)
	}
}
