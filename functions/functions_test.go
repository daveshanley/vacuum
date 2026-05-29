// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapBuiltinFunctions(t *testing.T) {
	funcs := MapBuiltinFunctions()
	assert.Len(t, funcs.GetAllFunctions(), 87)
	assert.Contains(t, funcs.GetAllFunctions(), "pathsSpecificityOrder")
	assert.Contains(t, funcs.GetAllFunctions(), "requiredFieldsDefined")
	assert.Contains(t, funcs.GetAllFunctions(), "jsonSchemaValid")
	assert.Contains(t, funcs.GetAllFunctions(), "jsonSchemaSanity")
	assert.Contains(t, funcs.GetAllFunctions(), "jsonSchemaRefValid")
}
