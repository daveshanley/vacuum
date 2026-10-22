package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteBundle copies verified local dependencies and the report into a new
// directory. It refuses existing destinations, escaping paths, incomplete
// manifests, and files changed since execution. Run manifest.suite inside the
// bundle with the recorded case IDs, sample limit, and vacuum version.
func WriteBundle(report *Report, suitePath, destination string) error {
	if report == nil || !report.Manifest.Reproducible {
		return fmt.Errorf("cannot bundle an incomplete manifest")
	}
	base, err := filepath.Abs(filepath.Dir(suitePath))
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	for _, f := range report.Manifest.Files {
		clean := filepath.Clean(filepath.FromSlash(f.Path))
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("bundle dependency escapes suite directory: %s", f.Path)
		}
		b, err := os.ReadFile(filepath.Join(base, clean))
		if err != nil {
			return err
		}
		h := sha256.Sum256(b)
		if hex.EncodeToString(h[:]) != f.SHA256 {
			return fmt.Errorf("dependency changed since execution: %s", f.Path)
		}
		files[clean] = b
	}
	if _, ok := files[filepath.Clean(report.Manifest.Suite)]; !ok {
		return fmt.Errorf("suite is missing from manifest")
	}
	// Keep metadata separate and reject collisions with captured input paths.
	for p := range files {
		if p == ".vacuum-test" || strings.HasPrefix(p, ".vacuum-test"+string(filepath.Separator)) {
			return fmt.Errorf("reserved bundle path: %s", p)
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	files[filepath.Join(".vacuum-test", "result.json")] = data
	if err = os.Mkdir(destination, 0700); err != nil {
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(destination)
		}
	}()
	for name, b := range files {
		path := filepath.Join(destination, name)
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		if err = os.WriteFile(path, b, 0600); err != nil {
			return err
		}
	}
	complete = true
	return nil
}
