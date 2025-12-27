// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package utils

// IsBatchMode checks if batch mode is enabled in function options.
// When batch mode is enabled, all matched nodes are passed to the function
// at once instead of invoking the function once per node.
func IsBatchMode(options interface{}) bool {
	if m, ok := options.(map[string]interface{}); ok {
		if batch, exists := m["batch"]; exists {
			if b, ok := batch.(bool); ok {
				return b
			}
		}
	}
	return false
}
