// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/testify/assert"
)

func TestNoRefSiblingsRules_FormatSelection(t *testing.T) {
	noRefSiblings := GetNoRefSiblingsRule()
	oas3NoRefSiblings := GetOAS3NoRefSiblingsRule()

	assert.True(t, ruleFormatsMatch(noRefSiblings.Formats, model.OAS3))
	assert.False(t, ruleFormatsMatch(noRefSiblings.Formats, model.OAS31))
	assert.False(t, ruleFormatsMatch(noRefSiblings.Formats, model.OAS32))

	assert.True(t, ruleFormatsMatch(oas3NoRefSiblings.Formats, model.OAS31))
	assert.False(t, ruleFormatsMatch(oas3NoRefSiblings.Formats, model.OAS32))
}

func TestGetCamelCasePropertiesRule_DefaultsToCamel(t *testing.T) {
	rule := GetCamelCasePropertiesRule()
	action, ok := rule.Then.(model.RuleAction)
	assert.True(t, ok)
	opts, ok := action.FunctionOptions.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "camel", opts["type"])
}

func ruleFormatsMatch(ruleFormats []string, specFormat string) bool {
	for _, ruleFormat := range ruleFormats {
		if model.FormatMatches(ruleFormat, specFormat) {
			return true
		}
	}
	return false
}
