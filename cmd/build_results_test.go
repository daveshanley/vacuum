package cmd

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBuildResults(t *testing.T) {
	_, _, err := BuildResults(false, false, "nuggets", nil, nil, "", true, 5*time.Second, 5*time.Second, utils.HTTPClientConfig{}, nil, model.IgnoredItems{}, nil)
	assert.Error(t, err)
}

func TestBuildResults_SkipCheck(t *testing.T) {
	_, _, err := BuildResultsWithDocCheckSkip(false, false, "nuggets", nil, nil, "", true, true, 5*time.Second, 5*time.Second, utils.HTTPClientConfig{}, nil, model.IgnoredItems{}, nil)
	assert.Error(t, err)
}
