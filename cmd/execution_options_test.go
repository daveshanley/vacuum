package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMotorExecutionOptions(t *testing.T) {
	assert.Nil(t, newMotorExecutionOptions(false, false))

	opts := newMotorExecutionOptions(true, true)
	if assert.NotNil(t, opts) {
		assert.True(t, opts.ResolveAllRefs)
		assert.True(t, opts.NestedRefsDocContext)
	}
}

func TestNewMotorExecutionOptionsFromExecutionFlags(t *testing.T) {
	assert.Nil(t, newMotorExecutionOptionsFromExecutionFlags(nil))

	opts := newMotorExecutionOptionsFromExecutionFlags(&ExecutionFlags{
		ResolveAllRefs:       true,
		NestedRefsDocContext: true,
	})
	if assert.NotNil(t, opts) {
		assert.True(t, opts.ResolveAllRefs)
		assert.True(t, opts.NestedRefsDocContext)
	}
}

func TestNewMotorExecutionOptionsFromLintFlags(t *testing.T) {
	assert.Nil(t, newMotorExecutionOptionsFromLintFlags(nil))
	assert.Nil(t, newMotorExecutionOptionsFromLintFlags(&LintFlags{}))

	opts := newMotorExecutionOptionsFromLintFlags(&LintFlags{NestedRefsDocContext: true})
	if assert.NotNil(t, opts) {
		assert.False(t, opts.ResolveAllRefs)
		assert.True(t, opts.NestedRefsDocContext)
	}
}
