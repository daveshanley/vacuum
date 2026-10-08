// Copyright 2020-2026 Dave Shanley / Quobix
// Copyright 2022-2026 Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package functions

import (
	"testing"

	"github.com/pb33f/testify/assert"
)

func TestMapBuiltinFunctions(t *testing.T) {
	funcs := MapBuiltinFunctions()
	assert.Len(t, funcs.GetAllFunctions(), 103)
	assert.Contains(t, funcs.GetAllFunctions(), "or")
	assert.Equal(t, "or", funcs.FindFunction("or").GetSchema().Name)
	assert.Contains(t, funcs.GetAllFunctions(), "pathsSpecificityOrder")
	assert.Contains(t, funcs.GetAllFunctions(), "requiredFieldsDefined")
	assert.Contains(t, funcs.GetAllFunctions(), "asyncApiDocument")
	assert.Contains(t, funcs.GetAllFunctions(), "arazzoDocument")
	assert.Contains(t, funcs.GetAllFunctions(), "asyncApiChannelServers")
	assert.Contains(t, funcs.GetAllFunctions(), "jsonSchemaValid")
	assert.Contains(t, funcs.GetAllFunctions(), "jsonSchemaSanity")
	assert.Contains(t, funcs.GetAllFunctions(), "jsonSchemaRefValid")
}
