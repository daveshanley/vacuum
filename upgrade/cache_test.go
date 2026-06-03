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

func TestUpdateCacheReadStatusReturnsStaleReleaseAndRefreshDecision(t *testing.T) {
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

	result, hasCachedRelease, shouldRefresh := cache.ReadStatus("v0.26.0", DefaultCacheMaxAge)
	if !hasCachedRelease {
		t.Fatalf("ReadStatus did not return cached release data")
	}
	if !shouldRefresh {
		t.Fatalf("ReadStatus did not request refresh for stale cache")
	}
	if result.CurrentVersion != "v0.26.0" ||
		result.LatestVersion != "v0.27.0" ||
		result.ReleaseURL != "https://example.com/release" {
		t.Fatalf("cached result = %#v", result)
	}
}

func TestUpdateCacheShouldRefresh(t *testing.T) {
	now := time.Unix(1700000000, 0)
	cache := &UpdateCache{
		Path: filepath.Join(t.TempDir(), "update-check.json"),
		Now:  func() time.Time { return now },
	}

	if !cache.ShouldRefresh(DefaultCacheMaxAge) {
		t.Fatalf("ShouldRefresh returned false for missing cache")
	}
	if err := cache.Write(CheckResult{LatestVersion: "v0.27.0"}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if cache.ShouldRefresh(DefaultCacheMaxAge) {
		t.Fatalf("ShouldRefresh returned true for fresh cache")
	}

	cache.Now = func() time.Time { return now.Add(DefaultCacheMaxAge + time.Second) }
	if !cache.ShouldRefresh(DefaultCacheMaxAge) {
		t.Fatalf("ShouldRefresh returned false for stale cache")
	}
}

func TestUpdateCacheDoesNotWriteFailedResult(t *testing.T) {
	cache := &UpdateCache{Path: filepath.Join(t.TempDir(), "update-check.json")}

	if err := cache.Write(CheckResult{
		LatestVersion: "v0.27.0",
		Err:           fmt.Errorf("unavailable"),
	}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if _, _, ok := cache.read(); ok {
		t.Fatalf("failed result created cache data")
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

	cached, _, ok := cache.read()
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

func TestDefaultUpdateCacheUsesTempDir(t *testing.T) {
	cache, err := DefaultUpdateCache()
	if err != nil {
		t.Fatalf("DefaultUpdateCache returned error: %v", err)
	}
	want := filepath.Join(os.TempDir(), "vacuum-update-check.json")
	if cache.Path != want {
		t.Fatalf("Path = %q, want %q", cache.Path, want)
	}
}
