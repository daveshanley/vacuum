package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildResults(t *testing.T) {
	_, _, err := BuildResults(false, false, "nuggets", nil, nil, "", 5)
	assert.Error(t, err)
}

func TestBuildResults_SkipCheck(t *testing.T) {
	_, _, err := BuildResultsWithDocCheckSkip(false, false, "nuggets", nil, nil, "", true, 5)
	assert.Error(t, err)
}
