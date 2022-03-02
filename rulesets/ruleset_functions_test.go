// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRuleSet_ContactProperties(t *testing.T) {

	yml := `info:
  contact:
    name: pizza
    email: monkey`

	rules := make(map[string]*model.Rule)
	rules["contact-properties"] = GetContactPropertiesRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Contact object must be complete': 'url' must be set", results[0].Message)

}
