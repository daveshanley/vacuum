// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultCacheMaxAge = 12 * time.Hour
)

type UpdateCache struct {
	Path string
	Now  func() time.Time
}

type UpdateCacheData struct {
	LatestVersion string `json:"latestVersion"`
	ReleaseURL    string `json:"releaseURL"`
}

func DefaultUpdateCache() (*UpdateCache, error) {
	return &UpdateCache{
		Path: filepath.Join(os.TempDir(), "vacuum-update-check.json"),
	}, nil
}

func (c *UpdateCache) ReadFresh(currentVersion string, maxAge time.Duration) (CheckResult, bool) {
	cached, modTime, ok := c.read()
	if !ok || !c.isFresh(modTime, maxAge) || cached.LatestVersion == "" {
		return CheckResult{}, false
	}
	return CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  cached.LatestVersion,
		ReleaseURL:     cached.ReleaseURL,
	}, true
}

func (c *UpdateCache) ReadStatus(currentVersion string, maxAge time.Duration) (result CheckResult, hasCachedRelease bool, shouldRefresh bool) {
	cached, modTime, ok := c.read()
	if !ok {
		return CheckResult{}, false, true
	}
	if cached.LatestVersion == "" {
		return CheckResult{}, false, true
	}
	result = CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  cached.LatestVersion,
		ReleaseURL:     cached.ReleaseURL,
	}
	return result, true, !c.isFresh(modTime, maxAge)
}

func (c *UpdateCache) ShouldRefresh(maxAge time.Duration) bool {
	_, modTime, ok := c.read()
	return !ok || !c.isFresh(modTime, maxAge)
}

func (c *UpdateCache) Write(result CheckResult) error {
	if c == nil || c.Path == "" || result.Err != nil || result.LatestVersion == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.Path), 0o755); err != nil {
		return err
	}
	return c.write(UpdateCacheData{
		LatestVersion: result.LatestVersion,
		ReleaseURL:    result.ReleaseURL,
	})
}

func (c *UpdateCache) write(data UpdateCacheData) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.Path)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(c.Path)+".tmp.")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(encoded); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(0o644); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, c.Path); err != nil {
		return err
	}
	now := c.now()
	return os.Chtimes(c.Path, now, now)
}

func (c *UpdateCache) read() (UpdateCacheData, time.Time, bool) {
	if c == nil || c.Path == "" {
		return UpdateCacheData{}, time.Time{}, false
	}
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return UpdateCacheData{}, time.Time{}, false
	}
	info, err := os.Stat(c.Path)
	if err != nil {
		return UpdateCacheData{}, time.Time{}, false
	}
	var cached UpdateCacheData
	if err := json.Unmarshal(data, &cached); err != nil {
		return UpdateCacheData{}, time.Time{}, false
	}
	return cached, info.ModTime(), true
}

func (c *UpdateCache) isFresh(modTime time.Time, maxAge time.Duration) bool {
	if maxAge <= 0 {
		maxAge = DefaultCacheMaxAge
	}
	delta := c.now().Sub(modTime)
	return delta >= 0 && delta <= maxAge
}

func (c *UpdateCache) now() time.Time {
	if c != nil && c.Now != nil {
		return c.Now()
	}
	return time.Now()
}
