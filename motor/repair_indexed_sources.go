// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"bytes"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
)

// Bundling and final validation need separate external indexes. These private
// files let the native rolodex build indexes from frozen bytes without lending
// it baseline-owned nodes or resolving nested references against the root file.
type repairIndexedSources struct {
	*repairSources
	indexed map[string]index.RolodexFile
}

func newRepairIndexedSources(sources *repairSources) (*repairIndexedSources, error) {
	frozen := &repairIndexedSources{repairSources: sources, indexed: make(map[string]index.RolodexFile)}
	for name, data := range sources.files {
		if strings.Contains(name, "://") {
			continue // The native remote loader indexes the frozen HTTP response.
		}
		info, err := datamodel.ExtractSpecInfoWithDocumentCheckSync(data, true)
		if err != nil {
			return nil, err
		}
		frozen.indexed[name] = &repairIndexedFile{
			repairSourceFile: &repairSourceFile{Reader: bytes.NewReader(data), name: name, size: int64(len(data))},
			data:             data, info: info,
		}
	}
	return frozen, nil
}

func (s *repairIndexedSources) GetFiles() map[string]index.RolodexFile { return s.indexed }

func (s *repairIndexedSources) Open(name string) (fs.File, error) {
	full := name
	if !filepath.IsAbs(full) {
		full = filepath.Join(s.base, full)
	}
	if file := s.indexed[full]; file != nil {
		return file.(*repairIndexedFile), nil
	}
	return s.repairSources.Open(name)
}

type repairIndexedFile struct {
	*repairSourceFile
	data  []byte
	info  *datamodel.SpecInfo
	mu    sync.Mutex
	index *index.SpecIndex
}

func (f *repairIndexedFile) GetContent() string { return string(f.data) }
func (f *repairIndexedFile) GetFileExtension() index.FileExtension {
	return index.ExtractFileType(f.name)
}
func (f *repairIndexedFile) GetFullPath() string { return f.name }
func (f *repairIndexedFile) GetErrors() []error  { return nil }
func (f *repairIndexedFile) GetContentAsYAMLNode() (*yaml.Node, error) {
	return f.info.RootNode, nil
}
func (f *repairIndexedFile) GetIndex() *index.SpecIndex {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.index
}
func (f *repairIndexedFile) WaitForIndexing() {}
func (f *repairIndexedFile) Index(config *index.SpecIndexConfig) (*index.SpecIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.index == nil {
		privateConfig := *config
		privateConfig.SpecInfo = f.info
		f.index = index.NewSpecIndexWithConfig(f.info.RootNode, &privateConfig)
	}
	return f.index, nil
}
