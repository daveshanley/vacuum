// Package jsruleset prepares local JavaScript rulesets for vacuum's rule engine.
package jsruleset

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// File records the exact bytes used when preparing a module graph.
type File struct {
	Path   string
	SHA256 string
}

// Prepared owns immutable compiled code and normalized rule data. It is safe to
// share across documents; each execution must open its own Scope.
type Prepared struct {
	program       *goja.Program
	normalized    []byte
	files         []File
	entry         string
	names         []string
	functionNames map[string]string
}

// Data returns the normalized declarative ruleset, with stable callable handles.
func (p *Prepared) Data() []byte { return append([]byte(nil), p.normalized...) }

// Files returns the local module and resolution metadata identities.
func (p *Prepared) Files() []File { return append([]File(nil), p.files...) }

// IsModule reports supported executable entry extensions.
func IsModule(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".cjs", ".mjs":
		return true
	}
	return false
}

//go:embed normalize.js
var normalizeSource string

// Prepare compiles a local module graph without installing packages or loading
// remote code. Initialization and export inspection have a five-second limit.
func Prepare(ctx context.Context, entry string) (*Prepared, error) {
	return PrepareWithReader(ctx, entry, os.ReadFile)
}

const initializationTimeout = 5 * time.Second
