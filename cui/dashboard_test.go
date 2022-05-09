// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cui

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	ui "github.com/gizak/termui/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"testing"
	"time"
)

func TestCreateDashboard(t *testing.T) {
	resultSet, index, info := testBootDashboard()
	dash := CreateDashboard(resultSet, index, info)
	assert.Equal(t, "openapi", dash.info.SpecType)
}

func TestDashboard_GenerateTabbedView(t *testing.T) {

	resultSet, index, info := testBootDashboard()
	dash := CreateDashboard(resultSet, index, info)
	dash.ruleCategories = model.RuleCategoriesOrdered
	dash.GenerateTabbedView()
	assert.Equal(t, "information", dash.selectedCategory.Id)
}

func TestDashboard_Render(t *testing.T) {

	resultSet, index, info := testBootDashboard()
	dash := CreateDashboard(resultSet, index, info)
	dash.ruleCategories = model.RuleCategoriesOrdered

	// define our own events channel, so we can trigger the UI in any sequence we want.
	eventChan := make(chan ui.Event)
	dash.uiEvents = eventChan
	sequence := []string{
		"h",
		"<Escape>",
		"<Tab>",
		"<Tab>",
		"<Enter>",
		"<Down>",
		"<Down>",
		"<Up>",
		"<Up>",
		"<Escape>",
		"<Down>",
		"<Enter>",
		"<Down>",
		"<Down>",
		"<Down>",
		"<Enter>",
		"<Up>",
		"<Up>",
		"<Up>",
		"<Escape>",
		"<Right>",
		"<Down>",
		"<Up>",
		"<Enter>",
		"<Escape>",
		"<Left>",
		"<Tab>",
		"<Tab>",
		"<Tab>",
		"<Tab>",
		"<Tab>",
		"<Tab>",
		"<Tab>",
		"<Tab>",
		"<Tab>",
		"<Tab>",
		"<Tab>",
		"<Tab>",
		"q",
	}

	go func() {
		// simulate a super, super fast keyboard.
		for _, cmd := range sequence {
			time.Sleep(1 * time.Millisecond)
			eventChan <- ui.Event{
				Type:    0,
				ID:      cmd,
				Payload: nil,
			}
		}
	}()

	// TODO: detatch console UI renderer from logic, so we can run logic, without
	// worrying about the renderer being available.
	//---------
	// if there is a render error, it's because the console UI cannot be rendered in the
	// pipeline. This will result in a significant reduction in code being called during the
	// test.
	renderError := dash.Render()
	if renderError == nil {
		assert.Len(t, dash.categoryHealthGauge, 9)
	} else {

		// figure out what to do here once we have decoupled logic from rendering.
		dash.generateViewsAfterEvent()
		dash.setGrid()
	}
}

func testBootDashboard() (*model.RuleResultSet, *model.SpecIndex, *model.SpecInfo) {
	var rootNode yaml.Node
	yamlBytes, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	info, _ := model.ExtractSpecInfo(yamlBytes)

	yaml.Unmarshal(yamlBytes, &rootNode)
	index := model.NewSpecIndex(&rootNode)

	// let's go ahead and lint the spec and pass the results to the dashboard.
	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

	applied := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet: selectedRS,
		Spec:    yamlBytes,
	})
	resultSet := model.NewRuleResultSet(applied.Results)
	return resultSet, index, info
}
