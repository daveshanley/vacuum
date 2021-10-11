package functions

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMapBuiltinFunctions(t *testing.T) {
	funcs := MapBuiltinFunctions()
	assert.Len(t, funcs.GetAllFunctions(), 1)
}

func Test_FindHelloFunction(t *testing.T) {
	funcs := MapBuiltinFunctions()
	assert.NotNil(t, funcs.FindFunction("hello"))
}

func Test_RunHelloFunction(t *testing.T) {
	funcs := MapBuiltinFunctions()
	helloFunc := funcs.FindFunction("hello")
	res := helloFunc.RunRule("pizza", nil, nil)
	assert.NotNil(t, res)
	assert.Equal(t, "oh hello 'pizza'", res.Message)
}
