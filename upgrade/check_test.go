// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchLatestRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v1.2.3","html_url":"https://example.com/release","draft":false,"prerelease":false}`)
	}))
	defer server.Close()

	release, err := FetchLatestRelease(context.Background(), CheckOptions{LatestReleaseURL: server.URL})
	if err != nil {
		t.Fatalf("FetchLatestRelease returned error: %v", err)
	}
	if release.TagName != "v1.2.3" {
		t.Fatalf("TagName = %q, want v1.2.3", release.TagName)
	}
}

func TestFetchLatestReleaseRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer server.Close()

	if _, err := FetchLatestRelease(context.Background(), CheckOptions{LatestReleaseURL: server.URL}); err == nil {
		t.Fatalf("FetchLatestRelease returned nil error for HTTP 403")
	}
}

func TestFetchLatestReleaseRejectsDraftAndPrerelease(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "draft",
			body: `{"tag_name":"v1.2.3","html_url":"https://example.com/release","draft":true,"prerelease":false}`,
		},
		{
			name: "prerelease",
			body: `{"tag_name":"v1.2.3","html_url":"https://example.com/release","draft":false,"prerelease":true}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()

			if _, err := FetchLatestRelease(context.Background(), CheckOptions{LatestReleaseURL: server.URL}); err == nil {
				t.Fatalf("FetchLatestRelease returned nil error")
			}
		})
	}
}

func TestCheckTryResultDoesNotWait(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		fmt.Fprint(w, `{"tag_name":"v1.2.3"}`)
	}))
	defer server.Close()

	check := StartCheck(CheckOptions{
		CurrentVersion:   "1.2.2",
		LatestReleaseURL: server.URL,
		Timeout:          time.Second,
	})
	defer check.Cancel()

	if _, ok := check.TryResult(); ok {
		t.Fatalf("TryResult returned ready result before server responded")
	}
}
