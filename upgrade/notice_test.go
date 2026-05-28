package upgrade

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderNoticeUsesBoxedUpdateMessage(t *testing.T) {
	var out bytes.Buffer
	RenderNotice(&out, CheckResult{
		CurrentVersion: "v0.26.0",
		LatestVersion:  "v0.27.0",
		ReleaseURL:     "https://example.com/release",
	})

	rendered := out.String()
	for _, want := range []string{
		"┏",
		"┃",
		"UPDATE AVAILABLE",
		"v0.26.0 -> v0.27.0",
		"Run ",
		"vacuum upgrade",
		"Release notes: https://example.com/release",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered notice is missing %q:\n%s", want, rendered)
		}
	}
	if !strings.HasPrefix(rendered, "\n") || !strings.HasSuffix(rendered, "\n\n") {
		t.Fatalf("expected one blank line before and after notice, got %q", rendered)
	}
}
