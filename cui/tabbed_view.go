package cui

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// TabbedView represents a tabbed view holding various data views
type TabbedView struct {
	tv                   *widgets.TabPane
	dashboard            *Dashboard
	descriptionParagraph *widgets.Paragraph
	rulesList            *widgets.List
	descriptionGridItem  *ui.GridItem
	rulesListGridItem    *ui.GridItem
	currentRuleResults   *model.RuleResultsForCategory
}

func (t *TabbedView) setActiveIndex(index int) {
	t.dashboard.selectedTabIndex = index
	t.dashboard.selectedCategory = t.dashboard.ruleCategories[index]
	t.generateDescriptionGridItem()
	t.generateRulesInCategory()
}

func (t *TabbedView) setActiveRuleIndex(index int) {
	t.dashboard.selectedTabIndex = index
	t.dashboard.selectedCategory = t.dashboard.ruleCategories[index]
	t.generateDescriptionGridItem()
}

func (t *TabbedView) scrollDown() {
	t.rulesList.ScrollDown()
	t.dashboard.selectedRuleIndex = t.rulesList.SelectedRow
	if t.currentRuleResults.Rules != nil && t.currentRuleResults.Rules[t.rulesList.SelectedRow] != nil {
		t.dashboard.selectedRule = t.currentRuleResults.Rules[t.rulesList.SelectedRow]
	}
}

func (t *TabbedView) generateDescriptionGridItem() {
	if t.descriptionParagraph == nil {
		t.descriptionParagraph = widgets.NewParagraph()
		t.descriptionParagraph.Border = false
		t.descriptionParagraph.Text = t.dashboard.selectedCategory.Description
		desc := ui.NewRow(0.3, t.descriptionParagraph)
		t.descriptionGridItem = &desc
	} else {
		t.descriptionParagraph.Text = t.dashboard.selectedCategory.Description
	}
}

func (t *TabbedView) generateRulesInCategory() {

	results := t.dashboard.resultSet.GetRuleResultsForCategory(t.dashboard.selectedCategory.Id)
	t.currentRuleResults = results
	var rows []string
	for _, result := range results.Rules {
		rows = append(rows, fmt.Sprintf("%s (%d)", result.Id, results.Health))
	}
	if len(rows) == 0 {
		rows = append(rows, "🎉 Nothing in here, all clear, nice job!")
	}

	if t.rulesList == nil {
		t.rulesList = widgets.NewList()
		t.rulesList.Title = "Category Rules"
		t.rulesList.TextStyle = ui.NewStyle(ui.ColorGreen)
		rl := ui.NewRow(0.7, t.rulesList)
		t.rulesListGridItem = &rl
	}
	t.rulesList.Rows = rows
}
