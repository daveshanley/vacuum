package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildResults(t *testing.T) {
	_, _, err := BuildResults("nuggets", nil)
	assert.Error(t, err)
}
