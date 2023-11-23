// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package motor

import (
	"github.com/pb33f/libopenapi/index"
	"os"
	"path/filepath"
)

func BuildRolodexFromIndexConfig(indexConfig *index.SpecIndexConfig) (*index.Rolodex, error) {

	// create a rolodex
	rolodex := index.NewRolodex(indexConfig)

	// we need to create a local filesystem for the rolodex.
	if indexConfig.AllowFileLookup {
		cwd, absErr := filepath.Abs(indexConfig.BasePath)
		if absErr != nil {
			return nil, absErr
		}

		// create a local filesystem
		fsCfg := &index.LocalFSConfig{
			BaseDirectory: cwd,
			IndexConfig:   indexConfig,
			DirFS:         os.DirFS(cwd),
		}
		fileFS, err := index.NewLocalFSWithConfig(fsCfg)
		if err != nil {
			return nil, err
		}

		// add the filesystem to the rolodex
		rolodex.AddLocalFS(cwd, fileFS)
	}

	if indexConfig.AllowRemoteLookup {

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
	}

	return rolodex, nil
}
