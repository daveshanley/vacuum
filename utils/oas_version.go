// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import "github.com/pb33f/libopenapi/datamodel"

// IsOAS30 reports whether the spec is an OpenAPI 3.0.x document (excluding 3.1+).
func IsOAS30(specInfo *datamodel.SpecInfo) bool {
	if specInfo == nil {
		return false
	}
	return specInfo.VersionNumeric >= 3.0 && specInfo.VersionNumeric < 3.1
}
