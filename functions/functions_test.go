package functions

import (
	"github.com/daveshanley/vaccum/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMapBuiltinFunctions(t *testing.T) {
	funcs := MapBuiltinFunctions()
	assert.Len(t, funcs.GetAllFunctions(), 3)
}

func Test_FindHelloFunction(t *testing.T) {
	funcs := MapBuiltinFunctions()
	assert.NotNil(t, funcs.FindFunction("hello"))
}

func Test_RunHelloFunction(t *testing.T) {
	funcs := MapBuiltinFunctions()
	helloFunc := funcs.FindFunction("hello")

	res := helloFunc.RunRule(nil, model.RuleFunctionContext{})
	assert.NotNil(t, res)
	assert.Equal(t, "oh hello", res[0].Message)
}
