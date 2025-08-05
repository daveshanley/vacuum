package cmd

import (
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildResults(t *testing.T) {
	_, _, err := BuildResults(false, false, "nuggets", nil, nil, "", true, 5, utils.HTTPClientConfig{})
	assert.Error(t, err)
}

func TestBuildResults_SkipCheck(t *testing.T) {
	_, _, err := BuildResultsWithDocCheckSkip(false, false, "nuggets", nil, nil, "", true, true, 5, utils.HTTPClientConfig{})
	assert.Error(t, err)
}
