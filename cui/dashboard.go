package cui

import (
	"github.com/daveshanley/vacuum/model"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
)

// Dashboard represents the dashboard controlling container
type Dashboard struct {
	C                           chan bool
	run                         bool
	grid                        *ui.Grid
	helpGrid                    *ui.Grid
	title                       *widgets.Paragraph
	tabs                        TabbedView
	healthGaugeItems            []ui.GridItem
	categoryHealthGauge         []CategoryGauge
	resultSet                   *model.RuleResultSet
	index                       *index.SpecIndex
	info                        *datamodel.SpecInfo
	selectedTabIndex            int
	ruleCategories              []*model.RuleCategory
	selectedCategory            *model.RuleCategory
	selectedCategoryDescription *ui.GridItem
	selectedRule                *model.Rule
	selectedRuleIndex           int
	selectedViolationIndex      int
	selectedViolation           *model.RuleFunctionResult
	violationViewActive         bool
	helpViewActive              bool
	uiEvents                    <-chan ui.Event
}

func CreateDashboard(resultSet *model.RuleResultSet, index *index.SpecIndex, info *datamodel.SpecInfo) *Dashboard {
	db := new(Dashboard)
	db.resultSet = resultSet
	db.index = index
	db.info = info
	return db
}

// GenerateTabbedView generates tabs
func (dash *Dashboard) GenerateTabbedView() {
	var labels []string
	for _, cat := range dash.ruleCategories {
		labels = append(labels, cat.Name)
	}
	tv := widgets.NewTabPane(labels...)
	tv.BorderTop = false
	tv.BorderLeft = false
	tv.BorderRight = false
	tv.BorderBottom = true

	dash.tabs = TabbedView{tv: tv, dashboard: dash}
	dash.selectedTabIndex = 0
	dash.selectedCategory = dash.ruleCategories[0]
	dash.tabs.generateDescriptionGridItem()
	dash.tabs.generateRulesInCategory()
	if len(dash.tabs.currentRuleResults.RuleResults) > 0 {
		dash.selectedRule = dash.tabs.currentRuleResults.RuleResults[0].Rule
	}
	dash.tabs.generateRuleViolations()
	//dash.tabs.setActiveViolation()
	dash.tabs.generateRuleViolationView()

}

// ComposeGauges prepares health gauges for rendering into the main grid.
func (dash *Dashboard) ComposeGauges() {
	var gridItems []ui.GridItem
	for _, gauge := range dash.categoryHealthGauge {
		numCat := float64(len(dash.categoryHealthGauge))
		ratio := 0.8 / (numCat)
		gridItems = append(gridItems, ui.NewRow(ratio, gauge.g))
	}
	dash.healthGaugeItems = gridItems
}

// Render will render the dashboard.
func (dash *Dashboard) Render() error {

	if err := ui.Init(); err != nil {
		return err
	}

	dash.run = true
	defer ui.Close()

	// extract categories and calculate coverage.
	var gauges []CategoryGauge
	var cats []*model.RuleCategory
	for _, cat := range model.RuleCategoriesOrdered {
		score := dash.resultSet.CalculateCategoryHealth(cat.Id)
		gauges = append(gauges, NewCategoryGauge(cat.Name, score, cat))
		cats = append(cats, cat)
	}

	// todo: create a formula for this.
	gauges = append(gauges, NewCategoryGauge("Overall Health", 12, model.RuleCategoriesOrdered[0]))

	dash.categoryHealthGauge = gauges
	dash.ruleCategories = cats

	dash.grid = ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	dash.grid.SetRect(0, 0, termWidth, termHeight)

	dash.helpGrid = ui.NewGrid()
	dash.helpGrid.SetRect(0, 0, termWidth, termHeight)

	dash.GenerateTabbedView()
	dash.ComposeGauges()

	dash.setGrid()
	//dash.tabs.setActiveCategoryIndex(dash.tabs.tv.ActiveTabIndex)

	ui.Render(dash.grid, dash.title)
	dash.eventLoop(cats)
	return nil
}

func (dash *Dashboard) eventLoop(cats []*model.RuleCategory) {
	var uiEvents <-chan ui.Event
	if dash.uiEvents == nil {
		uiEvents = ui.PollEvents()
		dash.uiEvents = uiEvents
	} else {
		uiEvents = dash.uiEvents
	}
	// TODO: clean this damn mess up.
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "h":
				dash.helpViewActive = true
			case "<Tab>":
				dash.violationViewActive = false
				if dash.tabs.tv.ActiveTabIndex == len(cats)-1 { // loop around and around.
					dash.tabs.tv.ActiveTabIndex = 0
				} else {
					dash.tabs.tv.FocusRight()
				}
				dash.tabs.setActiveCategoryIndex(dash.tabs.tv.ActiveTabIndex)
				dash.generateViewsAfterEvent()

			case "<Enter>":
				dash.violationViewActive = true
				dash.generateViewsAfterEvent()

			case "<Escape>":
				dash.violationViewActive = false
				dash.helpViewActive = false
				dash.generateViewsAfterEvent()

			case "x", "<Right>":
				dash.violationViewActive = false
				dash.tabs.tv.FocusRight()
				dash.tabs.setActiveCategoryIndex(dash.tabs.tv.ActiveTabIndex)
				dash.generateViewsAfterEvent()

			case "s", "<Left>":
				dash.violationViewActive = false
				dash.tabs.tv.FocusLeft()
				dash.tabs.setActiveCategoryIndex(dash.tabs.tv.ActiveTabIndex)
				dash.generateViewsAfterEvent()

			case "z", "<Down>":
				if dash.violationViewActive {
					dash.tabs.scrollViolationsDown()
				} else {
					dash.tabs.scrollRulesDown()
				}
			case "a", "<Up>":
				if dash.violationViewActive {
					dash.tabs.scrollViolationsUp()
				} else {
					dash.tabs.scrollRulesUp()
				}
			}
			ui.Clear()
			if dash.helpViewActive {
				ui.Render(dash.helpGrid)
			} else {
				ui.Render(dash.grid, dash.title)
			}
		}
	}
}

func (dash *Dashboard) generateViewsAfterEvent() {
	dash.tabs.generateRuleViolations()
	dash.tabs.setActiveViolation()
	dash.tabs.generateRuleViolationView()
}

func (dash *Dashboard) setGrid() {

	p := widgets.NewParagraph()
	// todo: bring in correct versioning.

	p.Text = "vacuum v0.0.1: " +
		"[Select Category](fg:white,bg:clear,md:bold) = <Tab>,\u2B05\uFE0F\u27A1\uFE0F/S,X | " +
		"[Change Rule](fg:white,bg:clear,md:bold) = \u2B06\u2B07/A,Z | " +
		"[Select / Leave Rule](fg:white,bg:clear,md:bold) = <Enter> / <Esc>"
	p.TextStyle = ui.NewStyle(ui.ColorCyan, ui.ColorClear)
	p.Border = true
	p.BorderStyle = ui.NewStyle(ui.ColorCyan)
	p.PaddingLeft = 0
	p.PaddingTop = 0
	p.PaddingBottom = 0
	p.PaddingRight = 0
	//p.SetRect(0, 0, 800, 3)

	dash.title = p

	if !dash.helpViewActive {
		if dash.tabs.descriptionGridItem != nil {
			dash.grid.Set(
				ui.NewRow(0.07, p),
				ui.NewRow(0.93,
					// TODO: bring statistics back via a shortcut key combo, they take up too much space and don't add
					// enough value out of the box.
					//ui.NewCol(0.2,
					//	dash.healthGaugeItems[0], dash.healthGaugeItems[1], dash.healthGaugeItems[2], dash.healthGaugeItems[3],
					//	dash.healthGaugeItems[4], dash.healthGaugeItems[5], dash.healthGaugeItems[6], dash.healthGaugeItems[7],
					//	//dash.healthGaugeItems[8],
					//	ui.NewRow(0.3, NewStatsChart(dash.index, dash.info).bc),
					//),
					//ui.NewCol(0.01, nil),
					ui.NewCol(1.0,
						ui.NewRow(0.1, dash.tabs.tv),
						ui.NewRow(0.9,
							ui.NewCol(0.6,
								*dash.tabs.descriptionGridItem,
								*dash.tabs.rulesListGridItem,
								*dash.tabs.violationListGridItem,
							),
							ui.NewCol(0.4,
								*dash.tabs.violationViewGridItem,
								*dash.tabs.violationSnippetGridItem,
								*dash.tabs.violationFixGridItem),
						),
					),
				),
			)
		}
	}

	h := widgets.NewParagraph()
	h.Text = "This is the help screen, but it's not ready yet!"
	h.TextStyle = ui.NewStyle(ui.ColorGreen, ui.ColorYellow)
	h.BorderStyle = ui.NewStyle(ui.ColorBlack, ui.ColorBlack)

	// only render, if we can render.
	if dash.tabs.descriptionGridItem != nil {
		dash.helpGrid.Set(
			ui.NewRow(1,
				ui.NewCol(1.0, h),
			))
	}

}

func getColorForPercentage(percent int) ui.Color {
	if percent <= 30 {
		return ui.ColorRed
	}
	if percent > 30 && percent <= 70 {
		return ui.ColorYellow
	}
	if percent > 70 && percent <= 90 {
		return ui.ColorBlue
	}
	if percent > 80 {
		return ui.ColorGreen
	}
	return ui.ColorClear
}
