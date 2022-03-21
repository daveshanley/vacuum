package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBlank_GetSchema(t *testing.T) {
	def := Blank{}
	assert.Equal(t, "blank", def.GetSchema().Name)
}

func TestBlank_RunRule(t *testing.T) {
	def := Blank{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}
