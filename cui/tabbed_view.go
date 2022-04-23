package cui

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"strings"
)

// TabbedView represents a tabbed view holding various data views
type TabbedView struct {
	tv                       *widgets.TabPane
	dashboard                *Dashboard
	descriptionParagraph     *widgets.Paragraph
	rulesList                *widgets.List
	violationList            *widgets.List
	descriptionGridItem      *ui.GridItem
	rulesListGridItem        *ui.GridItem
	violationListGridItem    *ui.GridItem
	violationViewGridItem    *ui.GridItem
	violationSnippetGridItem *ui.GridItem
	violationViewMessage     *widgets.Paragraph
	violationCodeSnippet     *Snippet
	currentRuleResults       *model.RuleResultsForCategory
	currentViolationRules    []*model.RuleFunctionResult
}

func (t *TabbedView) setActiveCategoryIndex(index int) {
	t.dashboard.selectedTabIndex = index
	t.dashboard.selectedCategory = t.dashboard.ruleCategories[index]
	t.generateDescriptionGridItem()
	t.generateRulesInCategory()
	t.rulesList.SelectedRow = 0
	t.dashboard.violationViewActive = false
	t.violationList.SelectedRow = 0
	t.setActiveRule()
	t.setActiveViolation()
	t.generateRuleViolationView()
}

func (t *TabbedView) setActiveRuleCategoryIndex(index int) {
	t.dashboard.selectedTabIndex = index
	t.dashboard.selectedCategory = t.dashboard.ruleCategories[index]
	t.generateDescriptionGridItem()
}

func (t *TabbedView) scrollDown() {
	if !t.dashboard.violationViewActive {
		t.rulesList.ScrollDown()
	} else {
		t.violationList.ScrollDown()
	}
	t.setActiveRule()
	t.setActiveViolation()
	t.generateRuleViolationView()
}

func (t *TabbedView) scrollUp() {
	if !t.dashboard.violationViewActive {
		t.rulesList.ScrollUp()
	} else {
		t.violationList.ScrollUp()
	}
	t.setActiveRule()
	t.setActiveViolation()
	t.generateRuleViolationView()
}

func (t *TabbedView) setActiveRule() {
	t.dashboard.selectedRuleIndex = t.rulesList.SelectedRow
	if t.currentRuleResults.Rules != nil && t.currentRuleResults.Rules[t.rulesList.SelectedRow] != nil {
		t.dashboard.selectedRule = t.currentRuleResults.Rules[t.rulesList.SelectedRow].Rule
	}
	t.generateRuleViolations()
}

func (t *TabbedView) setActiveViolation() {
	t.dashboard.selectedViolationIndex = t.violationList.SelectedRow
	if t.currentViolationRules != nil && t.currentViolationRules[t.violationList.SelectedRow] != nil {
		t.dashboard.selectedViolation = t.currentViolationRules[t.violationList.SelectedRow]
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

		sev := result.Rule.GetSeverityAsIntValue()
		ruleType := ""
		ruleName := result.Rule.Name
		switch sev {
		case 0:
			ruleType = "🛑 [err!](fg:white,bg:red)"
			ruleName = fmt.Sprintf("[%s](fg:red)", result.Rule.Name)
		case 1:
			ruleType = "⚠️ [warn](fg:black,bg:yellow)"
		case 2:
			ruleType = "ℹ️ info"
		case 3:
			ruleType = "📍 hint"
		}

		rows = append(rows, fmt.Sprintf("%s: %s (%d)", ruleType, ruleName, result.Seen))
	}
	if len(rows) == 0 {
		rows = append(rows, "🎉 Nothing in here, all clear, nice job!")
	}

	if t.rulesList == nil {
		t.rulesList = widgets.NewList()
		t.rulesList.Title = "Category Rules"
		t.rulesList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite, ui.ModifierBold)
		rl := ui.NewRow(0.3, t.rulesList)
		t.rulesList.BorderBottom = false
		t.rulesList.BorderRight = false
		t.rulesList.BorderLeft = false
		t.rulesList.PaddingTop = 1
		t.rulesListGridItem = &rl
	}
	t.rulesList.Rows = rows
}

func (t *TabbedView) generateRuleViolations() {

	results := t.currentRuleResults
	var rows []string
	var violationRules []*model.RuleFunctionResult
	for _, result := range results.Rules {
		for _, violation := range result.Results {
			if t.dashboard.selectedRule == violation.Rule {
				rows = append(rows, fmt.Sprintf("%d:%d | %s",
					violation.StartNode.Line, violation.StartNode.Column, violation.Path))
				violationRules = append(violationRules, violation)
			}
		}
	}

	if t.violationList == nil {
		t.violationList = widgets.NewList()
		t.violationList.Title = "Violations"
		t.violationList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite, ui.ModifierBold)
		vl := ui.NewRow(0.4, t.violationList)
		t.violationList.BorderBottom = false
		t.violationList.BorderRight = false
		t.violationList.BorderLeft = false
		t.violationList.PaddingTop = 1
		t.violationListGridItem = &vl
	}
	t.violationList.Rows = rows
	t.currentViolationRules = violationRules
}

func (t *TabbedView) generateRuleViolationView() {
	if t.violationViewMessage == nil {
		resultMessage := widgets.NewParagraph()
		resultMessage.Text = t.dashboard.selectedViolation.Message
		resultMessage.WrapText = true
		resultMessage.BorderTop = false
		resultMessage.BorderBottom = false
		resultMessage.BorderRight = false
		resultMessage.PaddingLeft = 1
		t.violationViewMessage = resultMessage
		gi := ui.NewRow(0.3, resultMessage)
		t.violationViewGridItem = &gi
	} else {
		t.violationViewMessage.Text = t.dashboard.selectedViolation.Message
	}

	if t.violationCodeSnippet == nil {
		snippet := NewSnippet()
		snippet.Text = t.dashboard.selectedViolation.Message
		snippet.WrapText = false
		snippet.BorderTop = false
		snippet.BorderBottom = false
		snippet.BorderRight = false
		snippet.PaddingLeft = 1
		t.violationCodeSnippet = snippet
		gi := ui.NewRow(0.3, snippet)
		t.violationSnippetGridItem = &gi
	} else {
		specStringData := strings.Split(string(*t.dashboard.info.SpecBytes), "\n")
		t.violationCodeSnippet.Text = generateConsoleSnippet(t.dashboard.selectedViolation, specStringData,
			5, 5)
	}
}

func generateConsoleSnippet(r *model.RuleFunctionResult, specData []string, before, after int) string {
	// render out code snippet

	buf := new(strings.Builder)

	startLine := r.StartNode.Line
	endLine := r.StartNode.Line
	if r.StartNode.Line-before < 0 {
		startLine = before - ((r.StartNode.Line - before) * -1)
	} else {
		startLine = r.StartNode.Line - before
	}

	if r.StartNode.Line+after >= len(specData)-1 {
		endLine = len(specData) - 1
	} else {
		endLine = r.StartNode.Line + after
	}

	firstDelta := r.StartNode.Line - startLine
	secondDelta := endLine - r.StartNode.Line
	for i := 0; i < firstDelta; i++ {
		buf.WriteString(fmt.Sprintf("%d %s+ %s\n", startLine+i, "|", specData[startLine+i]))
	}
	buf.WriteString(fmt.Sprintf("%d %s %s", r.StartNode.Line, "|", "<--renderline-->\n"))

	for i := 0; i < secondDelta; i++ {
		buf.WriteString(fmt.Sprintf("%d %s- %s\n", r.StartNode.Line+i, "|", specData[endLine-i]))
	}

	return buf.String()
}
