//go:build js && wasm

package jsruleset

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestBrowserPrepareRejectsModulesWithoutReading(t *testing.T) {
	prepared, err := PrepareWithReader(context.Background(), "rules.mjs", func(string) ([]byte, error) {
		t.Fatal("browser preparation must not read the filesystem")
		return nil, nil
	})
	if prepared != nil || err == nil || !strings.Contains(err.Error(), "unavailable in browser builds") {
		t.Fatalf("expected explicit unsupported error, got %v, %v", prepared, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PrepareWithReader(ctx, "rules.mjs", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
