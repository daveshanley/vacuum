// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cui

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"testing"
)

func TestNewStatsChart(t *testing.T) {
	var rootNode yaml.Node
	yamlBytes, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	info, _ := model.ExtractSpecInfo(yamlBytes)
	yaml.Unmarshal(yamlBytes, &rootNode)
	index := model.NewSpecIndex(&rootNode)

	chart := NewStatsChart(index, info)

	assert.Equal(t, "Filesize: [10kb](fg:green)", chart.bc.Rows[0])
}
