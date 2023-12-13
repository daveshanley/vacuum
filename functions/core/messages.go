// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package core

func SuppliedOrDefault(supplied, original string) string {
	if supplied != "" {
		return supplied
	}
	return original
}
