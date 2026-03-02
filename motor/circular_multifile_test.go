package motor

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// lazyFile implements index.RolodexFile, index.CanBeIndexed, and fs.File.
// It simulates RevisionFS's behavior: files are loaded from disk but indexed
// lazily, with their own resolver and index created on demand.
type lazyFile struct {
	name     string
	fullPath string
	data     []byte
	parsed   *yaml.Node
	idx      *index.SpecIndex
	offset   int64
	mu       sync.Mutex
}

func (f *lazyFile) GetContent() string                    { return string(f.data) }
func (f *lazyFile) GetFileExtension() index.FileExtension { return index.YAML }
func (f *lazyFile) GetFullPath() string                   { return f.fullPath }
func (f *lazyFile) GetErrors() []error                    { return nil }
func (f *lazyFile) GetIndex() *index.SpecIndex            { return f.idx }
func (f *lazyFile) WaitForIndexing()                      {}
func (f *lazyFile) Name() string                          { return f.name }
func (f *lazyFile) ModTime() time.Time                    { return time.Now() }
func (f *lazyFile) IsDir() bool                           { return false }
func (f *lazyFile) Sys() any                              { return nil }
func (f *lazyFile) Size() int64                           { return int64(len(f.data)) }
func (f *lazyFile) Mode() os.FileMode                     { return 0 }
func (f *lazyFile) Close() error                          { return nil }
func (f *lazyFile) Stat() (fs.FileInfo, error)            { return f, nil }

func (f *lazyFile) Read(b []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.offset >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(b, f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *lazyFile) GetContentAsYAMLNode() (*yaml.Node, error) {
	if f.parsed != nil {
		return f.parsed, nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(f.data, &root); err != nil {
		return nil, err
	}
	f.parsed = &root
	return &root, nil
}

func (f *lazyFile) Index(config *index.SpecIndexConfig) (*index.SpecIndex, error) {
	if f.idx != nil {
		return f.idx, nil
	}
	node, err := f.GetContentAsYAMLNode()
	if err != nil {
		return nil, err
	}
	copiedCfg := *config
	copiedCfg.SpecAbsolutePath = f.fullPath
	copiedCfg.AvoidBuildIndex = true
	copiedCfg.SpecInfo = nil

	idx := index.NewSpecIndexWithConfig(node, &copiedCfg)
	idx.BuildIndex()
	f.idx = idx
	return idx, nil
}

// lazyFS implements index.RolodexFS, fs.FS, and index.Rolodexable.
// It simulates RevisionFS: GetFiles() returns only previously-opened files,
// Open() lazily loads files from disk, indexes them with their own resolver,
// and adds them to the rolodex.
type lazyFS struct {
	basePath string
	files    sync.Map
	rolodex  *index.Rolodex
	logger   *slog.Logger
}

func newLazyFS(basePath string) *lazyFS {
	return &lazyFS{basePath: basePath}
}

// SetRolodex implements index.Rolodexable — called by rolodex.AddLocalFS().
func (l *lazyFS) SetRolodex(r *index.Rolodex) {
	l.rolodex = r
}

// SetLogger implements index.Rolodexable — called by rolodex.AddLocalFS().
func (l *lazyFS) SetLogger(logger *slog.Logger) {
	l.logger = logger
}

// GetFiles returns only files that have already been opened.
// On first call during IndexTheRolodex, this returns an empty map
// (simulating RevisionFS behavior).
func (l *lazyFS) GetFiles() map[string]index.RolodexFile {
	files := make(map[string]index.RolodexFile)
	l.files.Range(func(key, value any) bool {
		files[key.(string)] = value.(*lazyFile)
		return true
	})
	return files
}

func (l *lazyFS) Open(name string) (fs.File, error) {
	// Check cache first.
	if v, ok := l.files.Load(name); ok {
		f := v.(*lazyFile)
		f.mu.Lock()
		f.offset = 0
		f.mu.Unlock()
		return f, nil
	}

	// Resolve the file path relative to basePath.
	cleanName := filepath.Clean(name)
	var filePath string
	if filepath.IsAbs(cleanName) {
		filePath = cleanName
	} else {
		filePath = filepath.Join(l.basePath, cleanName)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	absPath, _ := filepath.Abs(filePath)
	lf := &lazyFile{
		name:     filepath.Base(name),
		fullPath: absPath,
		data:     data,
	}

	// Store BEFORE indexing to avoid self-deadlock on recursive refs.
	l.files.Store(name, lf)

	// Index the file with its own resolver (simulating RevisionFS lines 619-640).
	// This is the critical behavior: each file gets pre-indexed with its own
	// resolver, and the index is added to the rolodex.
	if l.rolodex != nil {
		cfg := l.rolodex.GetConfig()
		if cfg != nil {
			copiedCfg := *cfg
			copiedCfg.SpecAbsolutePath = absPath
			copiedCfg.AvoidBuildIndex = true
			copiedCfg.SpecInfo = nil
			copiedCfg.Rolodex = l.rolodex

			node, parseErr := lf.GetContentAsYAMLNode()
			if parseErr == nil && node != nil {
				idx := index.NewSpecIndexWithConfig(node, &copiedCfg)
				idx.SetRolodex(l.rolodex)

				// Each file gets its own resolver (RevisionFS line 620)
				resolver := index.NewResolver(idx)
				idx.SetResolver(resolver)
				idx.BuildIndex()
				lf.idx = idx

				// Add to rolodex (RevisionFS line 640)
				l.rolodex.AddIndex(idx)
			}
		}
	}

	return lf, nil
}

func collectCircularResults(results *RuleSetExecutionResult) []model.RuleFunctionResult {
	var circularResults []model.RuleFunctionResult
	for i := range results.Results {
		if results.Results[i].RuleId == "circular-references" {
			circularResults = append(circularResults, results.Results[i])
		}
	}
	return circularResults
}

func formatCircularResults(results []model.RuleFunctionResult) string {
	var msgs []string
	for _, r := range results {
		msgs = append(msgs, fmt.Sprintf("%q", r.Message))
	}
	return strings.Join(msgs, ", ")
}

// TestCircularReferences_MultiFile_LazyFS tests that circular references are detected
// in a multi-file OpenAPI spec when loaded through a lazy filesystem (simulating
// bunkhouse's RevisionFS behavior where GetFiles() returns empty initially and
// files are loaded on demand via Open()).
//
// This is a regression test for the bug where the doctor (bunkhouse) failed to
// report circular references that the CLI detected, because:
// 1. RevisionFS.GetFiles() returns empty during IndexTheRolodex
// 2. Per-file resolvers are created lazily during resolution
// 3. Their circular references were never aggregated into the results
func TestCircularReferences_MultiFile_LazyFS(t *testing.T) {

	rootSpec, err := os.ReadFile("test_data/circular_multifile/root.yaml")
	require.NoError(t, err)

	basePath, _ := filepath.Abs("test_data/circular_multifile")
	lfs := newLazyFS(basePath)

	defaultRS := rulesets.BuildDefaultRuleSets()
	rs := defaultRS.GenerateOpenAPIRecommendedRuleSet()

	rse := &RuleSetExecution{
		RuleSet:     rs,
		Spec:        rootSpec,
		RolodexFS:   lfs,
		AllowLookup: true,
	}

	results := ApplyRulesToRuleSet(rse)
	circularResults := collectCircularResults(results)

	assertExpectedCircularRefs(t, circularResults, "multi-file lazy FS")
}

// TestCircularReferences_MultiFile_LocalFS is a control test that verifies circular
// references ARE detected when using the standard local filesystem (CLI behavior).
// This should always pass — it's the baseline for comparison with the lazy FS test.
func TestCircularReferences_MultiFile_LocalFS(t *testing.T) {

	rootSpec, err := os.ReadFile("test_data/circular_multifile/root.yaml")
	require.NoError(t, err)

	defaultRS := rulesets.BuildDefaultRuleSets()
	rs := defaultRS.GenerateOpenAPIRecommendedRuleSet()

	rse := &RuleSetExecution{
		RuleSet:      rs,
		Spec:         rootSpec,
		AllowLookup:  true,
		SpecFileName: "test_data/circular_multifile/root.yaml",
	}

	results := ApplyRulesToRuleSet(rse)
	circularResults := collectCircularResults(results)

	assertExpectedCircularRefs(t, circularResults, "local FS")
}

func assertExpectedCircularRefs(t *testing.T, circularResults []model.RuleFunctionResult, source string) {
	t.Helper()
	assert.Len(t, circularResults, 2,
		"expected 2 circular references (TreeNode and Task) from %s, got %d: %s",
		source, len(circularResults), formatCircularResults(circularResults))

	expected := map[string]bool{
		"circular reference detected from #/TreeNode": true,
		"circular reference detected from #/Task":     true,
	}
	for _, r := range circularResults {
		assert.True(t, expected[r.Message], "unexpected circular reference: %s", r.Message)
		delete(expected, r.Message)
	}
	assert.Len(t, expected, 0, "missing expected circular references: %v", expected)
}
