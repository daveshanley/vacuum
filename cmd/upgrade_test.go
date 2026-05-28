package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/upgrade"
)

func TestRunUpgradeUsesUpdateCheckOptions(t *testing.T) {
	previousVersionInfo := versionInfo
	previousOptions := updateCheckOptions
	defer func() {
		versionInfo = previousVersionInfo
		updateCheckOptions = previousOptions
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v0.27.0","html_url":"https://example.com/release","draft":false,"prerelease":false}`)
	}))
	defer server.Close()

	versionInfo = VersionInfo{Version: "v0.27.0"}
	updateCheckOptions = upgrade.CheckOptions{
		LatestReleaseURL: server.URL,
		Timeout:          time.Second,
	}

	var out bytes.Buffer
	cmd := GetUpgradeCommand()
	cmd.SetOut(&out)

	if err := runUpgrade(cmd, nil); err != nil {
		t.Fatalf("runUpgrade returned error: %v", err)
	}
	if !strings.Contains(out.String(), "vacuum is already up to date (v0.27.0).") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
