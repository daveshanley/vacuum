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

func TestBuildResultsWithDocCheckSkipAndExecutionFlags_NoExecutionFlags(t *testing.T) {
	_, _, err := BuildResultsWithDocCheckSkipAndExecutionFlags(false, false, "nuggets", nil, nil, "", true, true, 5*time.Second, 5*time.Second, utils.HTTPClientConfig{}, nil, model.IgnoredItems{}, nil, nil)
	assert.Error(t, err)
}

func TestBuildResultsWithDocCheckSkipAndExecutionFlags_WithExecutionFlags(t *testing.T) {
	resultSet, ruleset, err := BuildResultsWithDocCheckSkipAndExecutionFlags(false, false, "", nil, nil, "", true, true, 5*time.Second, 5*time.Second, utils.HTTPClientConfig{}, nil, model.IgnoredItems{}, nil, &ExecutionFlags{
		ResolveAllRefs: true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resultSet)
	assert.NotNil(t, ruleset)
}

func TestBuildResultsWithDocCheckSkipAndExecutionFlags_InvalidHTTPClientConfig(t *testing.T) {
	_, _, err := BuildResultsWithDocCheckSkipAndExecutionFlags(false, false, "ruleset.yaml", nil, nil, "", false, true, 5*time.Second, 5*time.Second, utils.HTTPClientConfig{
		CertFile: "cert.pem",
	}, nil, model.IgnoredItems{}, nil, &ExecutionFlags{
		ResolveAllRefs: true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create custom HTTP client")
}
