package cui

import (
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
}

func (t *TabbedView) setActiveIndex(index int) {
	t.dashboard.selectedTabIndex = index
	t.dashboard.selectedCategory = t.dashboard.ruleCategories[index]
	t.generateDescriptionGridItem()
}

func (t *TabbedView) generateDescriptionGridItem() {
	if t.descriptionParagraph == nil {
		t.descriptionParagraph = widgets.NewParagraph()
		t.descriptionParagraph.Text = t.dashboard.selectedCategory.Description
		desc := ui.NewRow(0.3, t.descriptionParagraph)
		t.descriptionGridItem = &desc
	} else {
		t.descriptionParagraph.Text = t.dashboard.selectedCategory.Description
	}
}

func (t *TabbedView) generateRulesInCategory() {
	t.rulesList = widgets.NewList()
	t.rulesList.Title = "Category Rules"
	t.rulesList.TextStyle = ui.NewStyle(ui.ColorGreen)

	// TODO: load category rules that were fired.

	t.rulesList.Rows = []string{
		"[97%](fg:red) Rule 1 Failure",
		"[80%](fg:red) Rule 2 Failure",
		"[50%](fg:blue) Rule 2 Failure",
		"[20%](fg:green) Rule 2 Failure",
	}

	rl := ui.NewRow(0.7, t.rulesList)
	t.rulesListGridItem = &rl
}
