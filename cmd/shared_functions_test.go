// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestBuildRuleSetFromUserSuppliedSet_All(t *testing.T) {
	rsFile := "../rulesets/examples/all-ruleset.yaml"
	rsBytes, _ := ioutil.ReadFile(rsFile)
	rsets := rulesets.BuildDefaultRuleSets()
	rs, err := BuildRuleSetFromUserSuppliedSet(rsBytes, rsets)
	assert.NoError(t, err)
	assert.Len(t, rs.Rules, 47)
}

func TestBuildRuleSetFromUserSuppliedSet_None(t *testing.T) {
	rsFile := "../rulesets/examples/norules-ruleset.yaml"
	rsets := rulesets.BuildDefaultRuleSets()
	rsBytes, _ := ioutil.ReadFile(rsFile)
	rs, err := BuildRuleSetFromUserSuppliedSet(rsBytes, rsets)
	assert.NoError(t, err)
	assert.Len(t, rs.Rules, 0)
}

func TestBuildRuleSetFromUserSuppliedSet_BadFile(t *testing.T) {
	rsFile := "../rulesets/examples/don't-exist.yaml"
	rsets := rulesets.BuildDefaultRuleSets()
	rsBytes, _ := ioutil.ReadFile(rsFile)
	rs, err := BuildRuleSetFromUserSuppliedSet(rsBytes, rsets)
	assert.Error(t, err)
	assert.Nil(t, rs)
}

func TestBuildRuleSetFromUserSuppliedSet_BadRuleset(t *testing.T) {
	rsFile := "../rulesets/schemas/ruleset.schema.json" // not a ruleset!
	rsets := rulesets.BuildDefaultRuleSets()
	rsBytes, _ := ioutil.ReadFile(rsFile)
	rs, err := BuildRuleSetFromUserSuppliedSet(rsBytes, rsets)
	assert.Error(t, err)
	assert.Nil(t, rs)
}

func TestRenderTime(t *testing.T) {
	// nothing really to test here, however I don't want coverage to drop.
	fi, _ := os.Stat("shared_functions.go")
	RenderTime(true, time.Microsecond, fi)
	RenderTime(false, time.Millisecond, fi)
}
