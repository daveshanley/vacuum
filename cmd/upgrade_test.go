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

func TestWriteManualUpgradeCommandsUsesActionSpecificCommand(t *testing.T) {
	var out bytes.Buffer
	action := upgrade.Action{
		Method:        upgrade.MethodHomebrew,
		CanRun:        false,
		Reason:        "Homebrew formula installs are not supported by automatic upgrade; switch to the supported cask version of vacuum",
		ManualCommand: "brew uninstall --formula vacuum && brew install --cask daveshanley/vacuum/vacuum",
	}

	writeManualUpgradeCommands(&out, action, "v0.28.1")

	output := out.String()
	if !strings.Contains(output, "Use this command:") {
		t.Fatalf("output did not use singular command prompt: %q", output)
	}
	if !strings.Contains(output, "brew uninstall --formula vacuum && brew install --cask daveshanley/vacuum/vacuum") {
		t.Fatalf("output did not include cask switch command: %q", output)
	}
	if strings.Contains(output, "Use one of these commands:") {
		t.Fatalf("output included generic commands: %q", output)
	}
}
