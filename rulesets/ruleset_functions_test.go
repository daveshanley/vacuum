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
	assert.Equal(t, "Contact details are incomplete: 'url' must be set", results[0].Message)

}

func TestRuleSet_InfoContact(t *testing.T) {

	yml := `info:
  title: Terrible API Spec
  description: No operations, no contact, useless.`

	rules := make(map[string]*model.Rule)
	rules["info-contact"] = GetInfoContactRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Info section is missing contact details: 'contact' must be set", results[0].Message)

}

func TestRuleSet_InfoDescription(t *testing.T) {

	yml := `info:
  title: Terrible API Spec
  contact:
    name: rubbish
    email: no@acme.com`

	rules := make(map[string]*model.Rule)
	rules["info-description"] = GetInfoDescriptionRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Info section is missing a description: 'description' must be set", results[0].Message)

}

func TestRuleSet_InfoLicense(t *testing.T) {

	yml := `info:
  title: Terrible API Spec
  description: really crap
  contact:
    name: rubbish
    email: no@acme.com`

	rules := make(map[string]*model.Rule)
	rules["info-license"] = GetInfoLicenseRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Info section should contain a license: 'license' must be set", results[0].Message)

}

func TestRuleSet_InfoLicenseUrl(t *testing.T) {

	yml := `info:
  title: Terrible API Spec
  description: really crap
  contact:
    name: rubbish
    email: no@acme.com
  license:
      name: Cake`

	rules := make(map[string]*model.Rule)
	rules["license-url"] = GetInfoLicenseUrlRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "License should contain an url: 'url' must be set", results[0].Message)

}
