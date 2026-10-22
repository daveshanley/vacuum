// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package transform

import (
	"strings"

	libutils "github.com/pb33f/libopenapi/utils"
)

func isSwagger(version string) bool { return strings.HasPrefix(version, "2.") }
func supportsWebhooks(version string) bool {
	return strings.HasPrefix(version, "3.1.") || supportsQueryAndAdditionalOperations(version) || version == "3.1"
}
func supportsQueryAndAdditionalOperations(version string) bool {
	return version == "3.2" || strings.HasPrefix(version, "3.2.")
}
func isOperationKey(key, version string) bool {
	return libutils.IsHttpVerb(key) || (key == "query" && supportsQueryAndAdditionalOperations(version))
}
