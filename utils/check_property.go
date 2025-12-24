// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

func SuppliedOrDefault(supplied, original string) string {
	if supplied != "" {
		return supplied
	}
	return original
}
