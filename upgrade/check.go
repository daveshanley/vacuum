// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultLatestReleaseURL = "https://api.github.com/repos/daveshanley/vacuum/releases/latest"
	DefaultReleaseNotesURL  = "https://github.com/daveshanley/vacuum/releases/latest"
	defaultCheckTimeout     = 5 * time.Second
	maxReleaseResponseBytes = 1 << 20
)

type LatestRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

type CheckOptions struct {
	CurrentVersion   string
	LatestReleaseURL string
	HTTPClient       *http.Client
	Timeout          time.Duration
}

type CheckResult struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseURL     string
	Err            error
}

type Check struct {
	cancel context.CancelFunc
	done   chan CheckResult
}

// StartCheck begins a latest-release probe in the background.
func StartCheck(opts CheckOptions) *Check {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultCheckTimeout
	}
	opts.Timeout = timeout

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	check := &Check{
		cancel: cancel,
		done:   make(chan CheckResult, 1),
	}

	go func() {
		defer cancel()
		result := CheckResult{CurrentVersion: opts.CurrentVersion}
		latest, err := FetchLatestRelease(ctx, opts)
		if err != nil {
			result.Err = err
			check.done <- result
			return
		}
		result.LatestVersion = latest.TagName
		result.ReleaseURL = latest.HTMLURL
		check.done <- result
	}()

	return check
}

func CompletedCheck(result CheckResult) *Check {
	done := make(chan CheckResult, 1)
	done <- result
	return &Check{done: done}
}

// TryResult returns immediately. If the network request has not finished,
// ok is false and callers should cancel the check and exit.
func (c *Check) TryResult() (result CheckResult, ok bool) {
	if c == nil {
		return CheckResult{}, false
	}
	select {
	case result = <-c.done:
		return result, true
	default:
		return CheckResult{}, false
	}
}

// WaitResult waits until the latest-release probe finishes or times out.
func (c *Check) WaitResult() (CheckResult, bool) {
	if c == nil {
		return CheckResult{}, false
	}
	result := <-c.done
	return result, true
}

func (c *Check) Cancel() {
	if c != nil && c.cancel != nil {
		c.cancel()
	}
}

func (r CheckResult) UpdateAvailable() bool {
	if r.Err != nil || r.LatestVersion == "" {
		return false
	}
	return IsNewer(r.LatestVersion, r.CurrentVersion)
}

func FetchLatestRelease(ctx context.Context, opts CheckOptions) (*LatestRelease, error) {
	url := opts.LatestReleaseURL
	if url == "" {
		url = DefaultLatestReleaseURL
	}
	client := opts.HTTPClient
	if client == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = defaultCheckTimeout
		}
		client = &http.Client{Timeout: timeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "vacuum-upgrade-check")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("latest release request returned status %s", resp.Status)
	}

	var release LatestRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxReleaseResponseBytes)).Decode(&release); err != nil {
		return nil, err
	}
	if release.TagName == "" {
		return nil, fmt.Errorf("latest release response is missing tag_name")
	}
	if release.HTMLURL == "" {
		release.HTMLURL = DefaultReleaseNotesURL
	}
	if release.Draft || release.Prerelease {
		return nil, fmt.Errorf("latest release is not a stable published release")
	}
	return &release, nil
}
