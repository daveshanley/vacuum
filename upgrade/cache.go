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
	DefaultCacheMaxAge    = 24 * time.Hour
	DefaultFailureBackoff = time.Hour
)

type UpdateCache struct {
	Path string
	Now  func() time.Time
}

type UpdateCacheData struct {
	CheckedAt     time.Time `json:"checkedAt"`
	LatestVersion string    `json:"latestVersion"`
	ReleaseURL    string    `json:"releaseURL"`
}

func DefaultUpdateCache() (*UpdateCache, error) {
	return &UpdateCache{
		Path: filepath.Join(defaultConfigHome(), "vacuum", ".vacuum-update-check.json"),
	}, nil
}

func (c *UpdateCache) ReadFresh(currentVersion string, maxAge time.Duration) (CheckResult, bool) {
	cached, ok := c.read()
	if !ok || !c.isFresh(cached, maxAge) || cached.LatestVersion == "" {
		return CheckResult{}, false
	}
	return CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  cached.LatestVersion,
		ReleaseURL:     cached.ReleaseURL,
	}, true
}

func (c *UpdateCache) ReadStatus(currentVersion string, releaseMaxAge, failureBackoff time.Duration) (result CheckResult, hasFreshRelease bool, recentlyChecked bool) {
	cached, ok := c.read()
	if !ok {
		return CheckResult{}, false, false
	}
	if cached.LatestVersion != "" && c.isFresh(cached, releaseMaxAge) {
		return CheckResult{
			CurrentVersion: currentVersion,
			LatestVersion:  cached.LatestVersion,
			ReleaseURL:     cached.ReleaseURL,
		}, true, true
	}
	return CheckResult{}, false, c.isFresh(cached, failureBackoff)
}

func (c *UpdateCache) RecentlyChecked(maxAge time.Duration) bool {
	cached, ok := c.read()
	return ok && c.isFresh(cached, maxAge)
}

func (c *UpdateCache) MarkChecked() error {
	if c == nil || c.Path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.Path), 0o755); err != nil {
		return err
	}
	cached, _ := c.read()
	cached.CheckedAt = c.now()
	return c.write(cached)
}

func (c *UpdateCache) Write(result CheckResult) error {
	if c == nil || c.Path == "" || result.Err != nil || result.LatestVersion == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.Path), 0o755); err != nil {
		return err
	}
	return c.write(UpdateCacheData{
		CheckedAt:     c.now(),
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
	return os.Rename(tmpPath, c.Path)
}

func (c *UpdateCache) read() (UpdateCacheData, bool) {
	if c == nil || c.Path == "" {
		return UpdateCacheData{}, false
	}
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return UpdateCacheData{}, false
	}
	var cached UpdateCacheData
	if err := json.Unmarshal(data, &cached); err != nil {
		return UpdateCacheData{}, false
	}
	if cached.CheckedAt.IsZero() {
		return UpdateCacheData{}, false
	}
	return cached, true
}

func (c *UpdateCache) isFresh(cached UpdateCacheData, maxAge time.Duration) bool {
	if maxAge <= 0 {
		maxAge = DefaultCacheMaxAge
	}
	delta := c.now().Sub(cached.CheckedAt)
	return delta >= 0 && delta <= maxAge
}

func (c *UpdateCache) now() time.Time {
	if c != nil && c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func defaultConfigHome() string {
	if xdgConfigHome, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		return xdgConfigHome
	}
	return filepath.Join(os.Getenv("HOME"), ".config")
}
