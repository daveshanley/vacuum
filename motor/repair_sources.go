package motor

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/daveshanley/vacuum/repair"
	"github.com/pb33f/libopenapi/index"
)

// repairSources holds only bytes already read during the baseline lint. Missing
// files fail closed: verification cannot fetch a new or changed dependency.
type repairSources struct {
	files map[string][]byte
	base  string
}

func (s *repairSources) Open(name string) (fs.File, error) {
	data, ok := s.files[name]
	if !ok && !strings.Contains(name, "://") {
		data, ok = s.files[filepath.Join(s.base, name)]
	}
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &repairSourceFile{Reader: bytes.NewReader(data), name: name, size: int64(len(data))}, nil
}

type repairSourceFile struct {
	*bytes.Reader
	name string
	size int64
}

func (f *repairSourceFile) Close() error               { return nil }
func (f *repairSourceFile) Stat() (fs.FileInfo, error) { return f, nil }
func (f *repairSourceFile) Name() string               { return f.name }
func (f *repairSourceFile) Size() int64                { return f.size }
func (f *repairSourceFile) Mode() fs.FileMode          { return 0444 }
func (f *repairSourceFile) ModTime() time.Time         { return time.Time{} }
func (f *repairSourceFile) IsDir() bool                { return false }
func (f *repairSourceFile) Sys() any                   { return nil }

func freezeRepairSources(result *RuleSetExecutionResult) (*repairSources, map[string]string, error) {
	e := result.RuleSetExecution
	source := &repairSources{files: make(map[string][]byte)}
	absolute, err := filepath.Abs(e.SpecFileName)
	if err != nil {
		return nil, nil, err
	}
	source.base = filepath.Dir(absolute)
	hashes := make(map[string]string)
	if !externalRepairInputs(e.CanonicalDocument) {
		return source, hashes, nil
	}
	if e.AsyncAPI != nil || e.Arazzo != nil || result.Index == nil || result.Index.GetRolodex() == nil {
		return nil, nil, fmt.Errorf("external inputs for this format require a frozen source collection")
	}
	rolodexes := []*index.Rolodex{result.Index.GetRolodex()}
	if e.IndexResolved != nil {
		rolodexes = append(rolodexes, e.IndexResolved.GetRolodex())
	}
	if e.IndexUnresolved != nil {
		rolodexes = append(rolodexes, e.IndexUnresolved.GetRolodex())
	}
	if e.Document != nil {
		rolodexes = append(rolodexes, e.Document.GetRolodex())
	}
	seen := make(map[*index.Rolodex]bool)
	for _, rolo := range rolodexes {
		if rolo == nil || seen[rolo] {
			continue
		}
		seen[rolo] = true
		for _, idx := range rolo.GetIndexes() {
			name := idx.GetSpecAbsolutePath()
			if name == "" || name == absolute {
				continue
			}
			file, err := rolo.Open(name)
			if err != nil {
				return nil, nil, fmt.Errorf("snapshot repair dependency: %w", err)
			}
			data := []byte(file.GetContent())
			hash := repair.Hash(data)
			if previous, ok := hashes[name]; ok && previous != hash {
				return nil, nil, fmt.Errorf("dependency changed during baseline lint: %s", name)
			}
			source.files[name] = data
			hashes[name] = hash
		}
	}
	if len(hashes) == 0 {
		return nil, nil, fmt.Errorf("external repair dependencies were not captured")
	}
	return source, hashes, nil
}

func (s *repairSources) GetFiles() map[string]index.RolodexFile { return nil }
