package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapBuiltinFunctions(t *testing.T) {
	funcs := MapBuiltinFunctions()
	assert.Len(t, funcs.GetAllFunctions(), 83)
	assert.Contains(t, funcs.GetAllFunctions(), "pathsSpecificityOrder")
}
