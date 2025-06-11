package cui

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"sort"
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
	violationFixGridItem     *ui.GridItem
	violationViewMessage     *widgets.Paragraph
	violationFixMessage      *widgets.Paragraph
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
	//t.generateRuleViolationView()
}

func (t *TabbedView) scrollRulesDown() {
	t.rulesList.ScrollDown()
	t.setActiveRule()
}

func (t *TabbedView) scrollViolationsDown() {
	if len(t.violationList.Rows) > 0 {
		t.violationList.ScrollDown()
		t.setActiveRule()
		t.setActiveViolation()
		t.generateRuleViolationView()
	}
}

func (t *TabbedView) scrollRulesUp() {
	t.rulesList.ScrollUp()
	t.setActiveRule()
}

func (t *TabbedView) scrollViolationsUp() {
	if len(t.violationList.Rows) > 0 {
		t.violationList.ScrollUp()
		t.setActiveRule()
		t.setActiveViolation()
		t.generateRuleViolationView()
	}
}

func (t *TabbedView) setActiveRule() {
	t.dashboard.selectedRuleIndex = t.rulesList.SelectedRow
	if t.currentRuleResults.RuleResults != nil && t.currentRuleResults.RuleResults[t.rulesList.SelectedRow] != nil {
		t.dashboard.selectedRule = t.currentRuleResults.RuleResults[t.rulesList.SelectedRow].Rule
	}
	//t.generateRuleViolations()
}

func (t *TabbedView) setActiveViolation() {
	if t.violationList == nil {
		return
	}
	t.dashboard.selectedViolationIndex = t.violationList.SelectedRow
	if t.violationList.SelectedRow > len(t.currentViolationRules)-1 {
		if len(t.currentViolationRules) <= 0 {
			return
		}
		t.dashboard.selectedViolation = t.currentViolationRules[0]
		return
	}
	if t.currentViolationRules != nil && t.currentViolationRules[t.violationList.SelectedRow] != nil {
		t.dashboard.selectedViolation = t.currentViolationRules[t.violationList.SelectedRow]
	}
}

func (t *TabbedView) generateDescriptionGridItem() {
	if t.descriptionParagraph == nil {
		t.descriptionParagraph = widgets.NewParagraph()
		t.descriptionParagraph.Border = false
		t.descriptionParagraph.Text = t.dashboard.selectedCategory.Description
		desc := ui.NewRow(0.2, t.descriptionParagraph)
		t.descriptionGridItem = &desc
	} else {
		t.descriptionParagraph.Text = t.dashboard.selectedCategory.Description
	}
}

func (t *TabbedView) generateRulesInCategory() {
	if t.dashboard == nil {
		return
	}
	results := t.dashboard.resultSet.GetRuleResultsForCategory(t.dashboard.selectedCategory.Id)
	t.currentRuleResults = results
	var rows []string
	// sort results
	sort.Sort(results)

	for _, result := range results.RuleResults {

		sev := result.Rule.GetSeverityAsIntValue()
		ruleType := ""
		ruleName := result.Rule.Name
		if ruleName == "" {
			ruleName = result.Rule.Id
		}
		switch sev {
		case 0:
			ruleType = "🔺"
			ruleName = fmt.Sprintf("[%s](fg:red,mod:bold)", ruleName)
		case 1:
			ruleType = "🔸"
			ruleName = fmt.Sprintf("[%s](fg:yellow)", ruleName)
		case 2:
			ruleType = "🔹"
		case 3:
			ruleType = "🔹"
		}

		rows = append(rows, fmt.Sprintf("%s %s (%d)", ruleType, ruleName, result.Seen))
	}
	if len(rows) == 0 {
		rows = append(rows, "🎉 Nothing in here, all clear, nice job!")
		t.dashboard.selectedViolationIndex = 0
		t.dashboard.selectedViolation = nil

	}

	if t.rulesList == nil {
		t.rulesList = widgets.NewList()
		t.rulesList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite, ui.ModifierBold)
		rl := ui.NewRow(0.3, t.rulesList)
		t.rulesList.BorderBottom = false
		t.rulesList.BorderRight = false
		t.rulesList.BorderLeft = false
		t.rulesList.PaddingTop = 1
		t.rulesListGridItem = &rl
	}

	t.rulesList.Rows = rows
	t.rulesList.Title = fmt.Sprintf("Category Rules Broken (%d)", len(rows))

}

func (t *TabbedView) generateRuleViolations() {
	if t.currentRuleResults == nil {
		return
	}
	results := t.currentRuleResults
	var rows []string
	var violationRules []*model.RuleFunctionResult
	for _, result := range results.RuleResults {
		for _, violation := range result.Results {
			if t.dashboard.selectedRule == violation.Rule {
				rows = append(rows, strings.ReplaceAll(violation.Path, "]", "}"))
				violationRules = append(violationRules, violation)
			}
		}
	}

	if t.violationList == nil {
		t.violationList = widgets.NewList()
		t.violationList.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite, ui.ModifierBold)
		vl := ui.NewRow(0.5, t.violationList)
		t.violationList.BorderBottom = false
		t.violationList.BorderRight = false
		t.violationList.BorderLeft = false
		t.violationList.PaddingTop = 1
		t.violationListGridItem = &vl
	}

	if t.dashboard.violationViewActive {
		t.violationList.Rows = rows
		t.violationList.Title = fmt.Sprintf("Rule Violations (%d)", len(rows))
	} else {
		t.violationList.Title = "Select rule to see violations"
		t.violationList.Rows = nil
	}
	t.currentViolationRules = violationRules

}

func (t *TabbedView) generateRuleViolationView() {
	if t.dashboard == nil {
		return
	}
	if t.violationViewMessage == nil {
		resultMessage := widgets.NewParagraph()
		if t.dashboard.violationViewActive {
			resultMessage.Text = t.dashboard.selectedViolation.Message
		} else {
			resultMessage.Text = "Select rule to see violations"
		}
		resultMessage.WrapText = true
		resultMessage.BorderTop = false
		resultMessage.BorderBottom = false
		resultMessage.Title = "Violation Details"
		resultMessage.BorderRight = false
		resultMessage.PaddingLeft = 1
		resultMessage.TextStyle = ui.NewStyle(ui.ColorMagenta, ui.ColorClear, ui.ModifierBold)
		resultMessage.TitleStyle = ui.NewStyle(ui.ColorMagenta, ui.ColorClear, ui.ModifierUnderline)
		resultMessage.PaddingTop = 1
		t.violationViewMessage = resultMessage
		gi := ui.NewRow(0.2, resultMessage)
		t.violationViewGridItem = &gi
	} else {
		if t.dashboard.selectedViolation != nil {
			t.violationViewMessage.Title = "Violation Details"
			t.violationViewMessage.Text = t.dashboard.selectedViolation.Message
		} else {
			t.violationViewMessage.Text = ""
			t.violationViewMessage.Title = ""
		}
	}

	// clear everything if violation view is not active
	if !t.dashboard.violationViewActive {
		t.violationViewMessage.Text = ""
		t.violationViewMessage.Title = ""
	}

	if t.violationCodeSnippet == nil {
		specStringData := strings.Split(string(*t.dashboard.info.SpecBytes), "\n")

		snippet := NewSnippet()
		if t.dashboard.violationViewActive {
			snippet.Text = generateConsoleSnippet(t.dashboard.selectedViolation, specStringData,
				8, 8)
		} else {
			snippet.Text = ""
		}

		snippet.WrapText = false
		snippet.BorderTop = false
		snippet.BorderBottom = false
		snippet.BorderRight = false
		snippet.PaddingLeft = 1
		t.violationCodeSnippet = snippet
		gi := ui.NewRow(0.5, snippet)
		t.violationSnippetGridItem = &gi
	} else {
		if t.dashboard.selectedViolation == nil {
			t.violationCodeSnippet.Text = ""
		} else {
			specStringData := strings.Split(string(*t.dashboard.info.SpecBytes), "\n")

			if t.dashboard.violationViewActive {
				t.violationCodeSnippet.Text = generateConsoleSnippet(t.dashboard.selectedViolation, specStringData,
					10, 10)
			} else {
				t.violationCodeSnippet.Text = ""
			}
		}
	}

	// clear everything if violation view is not active
	if !t.dashboard.violationViewActive {
		t.violationCodeSnippet.Text = ""
	}

	if t.violationFixMessage == nil {
		resultMessage := widgets.NewParagraph()
		if t.dashboard.violationViewActive {
			resultMessage.Text = t.dashboard.selectedViolation.Rule.HowToFix
		} else {
			resultMessage.Text = ""
		}
		resultMessage.WrapText = true
		resultMessage.BorderTop = false
		resultMessage.BorderBottom = false
		resultMessage.BorderRight = false
		resultMessage.PaddingLeft = 1
		resultMessage.TextStyle = ui.NewStyle(ui.ColorCyan, ui.ColorClear, ui.ModifierBold)
		resultMessage.Title = "How to fix violation"
		resultMessage.TitleStyle = ui.NewStyle(ui.ColorCyan, ui.ColorClear, ui.ModifierUnderline)
		resultMessage.PaddingTop = 1
		t.violationFixMessage = resultMessage
		gi := ui.NewRow(0.3, resultMessage)
		t.violationFixGridItem = &gi

	} else {
		if t.dashboard.selectedViolation != nil {
			t.violationFixMessage.Text = t.dashboard.selectedViolation.Rule.HowToFix
			t.violationFixMessage.Title = "How to fix violation"
		} else {
			t.violationFixMessage.Text = ""
			t.violationFixMessage.Title = ""
		}
	}

	// clear everything if violation view is not active
	if !t.dashboard.violationViewActive {
		t.violationFixMessage.Text = ""
		t.violationFixMessage.Title = ""
	}

}

func generateConsoleSnippet(r *model.RuleFunctionResult, specData []string, before, after int) string {
	// render out code snippet
	// TODO clean this up, it's a freaking mess.

	buf := new(strings.Builder)

	startLine := r.StartNode.Line - 1
	var endLine int
	if startLine-before < 0 {
		startLine = before - ((startLine - before) * -1)
	} else {
		startLine = startLine - before
	}

	if r.StartNode.Line+after >= len(specData)-1 {
		endLine = len(specData) - 1
	} else {
		endLine = r.StartNode.Line - 1 + after
	}

	firstDelta := (r.StartNode.Line - 1) - startLine
	secondDelta := endLine - r.StartNode.Line
	for i := 0; i < firstDelta; i++ {
		line := strings.ReplaceAll(specData[startLine+i], "[", "{")
		line = strings.ReplaceAll(line, "]", "}")

		buf.WriteString(fmt.Sprintf("%d |  %s\n", startLine+i, line))
	}
	var line string

	// todo, fix this.
	if r.StartNode.Line-1 < 0 || r.StartNode.Line-1 >= len(specData) {
		line = strings.ReplaceAll(specData[r.StartNode.Line], "[", "{")
	} else {
		line = strings.ReplaceAll(specData[r.StartNode.Line-1], "[", "{")
	}

	line = strings.ReplaceAll(line, "[", "}")

	affectedLine := fmt.Sprintf("%s                                                        ", line)
	buf.WriteString(fmt.Sprintf("[%d | %s](fg:white,bg:red)\n", r.StartNode.Line-1, affectedLine))

	for i := 0; i < secondDelta; i++ {
		line = strings.ReplaceAll(specData[r.StartNode.Line+i], "[", "{")
		line = strings.ReplaceAll(line, "]", "}")
		buf.WriteString(fmt.Sprintf("%d %s %s\n", r.StartNode.Line+i, "|", line))
	}

	return buf.String()
}
