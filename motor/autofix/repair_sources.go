// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"fmt"
	"path/filepath"

	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/motor/autofix/runtime"
	"github.com/daveshanley/vacuum/repair"
	"github.com/pb33f/libopenapi/index"
)

func freezeRepairSources(result *motor.RuleSetExecutionResult) (*runtime.Sources, map[string]string, error) {
	e := result.RuleSetExecution
	source := &runtime.Sources{Files: make(map[string][]byte)}
	absolute, err := filepath.Abs(e.SpecFileName)
	if err != nil {
		return nil, nil, err
	}
	source.Base = filepath.Dir(absolute)
	hashes := make(map[string]string)
	if !externalRepairInputs(e.CanonicalDocument) {
		return source, hashes, nil
	}
	if e.Arazzo != nil {
		captured := result.RepairSources
		if captured == nil || len(captured.Files) == 0 {
			return nil, nil, fmt.Errorf("external Arazzo source bytes were not captured")
		}
		if captured.CaptureError != nil {
			return nil, nil, captured.CaptureError
		}
		for name, data := range captured.Files {
			hashes[name] = repair.Hash(data)
		}
		return captured, hashes, nil
	}
	if e.AsyncAPI != nil || result.Index == nil || result.Index.GetRolodex() == nil {
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
			source.Files[name] = data
			hashes[name] = hash
		}
	}
	if len(hashes) == 0 {
		return nil, nil, fmt.Errorf("external repair dependencies were not captured")
	}
	return source, hashes, nil
}
