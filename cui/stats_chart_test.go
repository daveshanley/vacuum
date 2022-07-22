// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cui

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"testing"
)

func TestNewStatsChart(t *testing.T) {
	var rootNode yaml.Node
	yamlBytes, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	info, _ := datamodel.ExtractSpecInfo(yamlBytes)
	mErr := yaml.Unmarshal(yamlBytes, &rootNode)
	assert.NoError(t, mErr)
	idx := index.NewSpecIndex(&rootNode)

	chart := NewStatsChart(idx, info)

	assert.Equal(t, "Filesize: [11kb](fg:green)", chart.bc.Rows[0])
}
