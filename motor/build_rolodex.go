// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package motor

import (
	"github.com/pb33f/libopenapi/index"
	"io/fs"
	"path/filepath"
)

func BuildRolodexFromIndexConfig(indexConfig *index.SpecIndexConfig, customFS fs.FS) (*index.Rolodex, error) {

	// create a rolodex
	rolodex := index.NewRolodex(indexConfig)

	// we need to create a local filesystem for the rolodex.
	if indexConfig.AllowFileLookup {
		cwd, absErr := filepath.Abs(indexConfig.BasePath)
		if absErr != nil {
			return nil, absErr
		}

		if customFS == nil {
			// create a local filesystem
			fsCfg := &index.LocalFSConfig{
				BaseDirectory: cwd,
				IndexConfig:   indexConfig,
			}
			fileFS, err := index.NewLocalFSWithConfig(fsCfg)
			if err != nil {
				return nil, err
			}

			// add the filesystem to the rolodex
			rolodex.AddLocalFS(cwd, fileFS)
		} else {
			rolodex.AddLocalFS(cwd, customFS)
		}
	}

	if indexConfig.AllowRemoteLookup {

		if customFS == nil {
			// create a remote filesystem
			remoteFS, err := index.NewRemoteFSWithConfig(indexConfig)
			if err != nil {
				return nil, err
			}

			// add the filesystem to the rolodex
			if indexConfig.BaseURL == nil {
				rolodex.AddRemoteFS("root", remoteFS)
			} else {
				rolodex.AddRemoteFS(indexConfig.BaseURL.String(), remoteFS)
			}
		} else {
			// add the custom filesystem to the rolodex
			if indexConfig.BaseURL == nil {
				rolodex.AddRemoteFS("root", customFS)
			} else {
				rolodex.AddRemoteFS(indexConfig.BaseURL.String(), customFS)
			}
		}
	}

	return rolodex, nil
}
