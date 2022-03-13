package functions

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMapBuiltinFunctions(t *testing.T) {
	funcs := MapBuiltinFunctions()
	assert.Len(t, funcs.GetAllFunctions(), 24)
}
