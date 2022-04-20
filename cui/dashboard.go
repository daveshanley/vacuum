package cui

import (
	"github.com/daveshanley/vacuum/model"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"log"
	"time"
)

// Dashboard represents the dashboard controlling container
type Dashboard struct {
	C         chan bool
	run       bool
	resultSet *model.RuleResultSet
}

func CreateDashboard(resultSet *model.RuleResultSet) *Dashboard {
	db := new(Dashboard)
	db.resultSet = resultSet
	return db
}

// CategoryGauge represents a percent bar visualizing how well spec did in a particular category
type CategoryGauge struct {
	g   *widgets.Gauge
	cat *model.RuleCategory
}

// CategoryChart represents a stacked barchart with all data per category available
type CategoryChart struct {
	c    *widgets.StackedBarChart
	cats []*model.RuleCategory
}

// TabbedView represents a tabbed view holding various data views
type TabbedView struct {
	tv        *widgets.TabPane
	renderTab func()
}

// GenerateTabbedView will return a view with controllable tabs.
func (d Dashboard) GenerateTabbedView(cats []*model.RuleCategory) TabbedView {

	var labels []string
	for _, cat := range cats {
		labels = append(labels, cat.Name)
	}

	tv := widgets.NewTabPane(labels...)
	tv.Border = false
	return TabbedView{tv: tv}
}

// GenerateGauge returns a CategoryGauge already populated with the title, color and percent.
func (d Dashboard) GenerateGauge(title string, percent int, cat *model.RuleCategory) CategoryGauge {
	g := widgets.NewGauge()
	g.Title = title
	g.Percent = percent
	//g.BorderRight = true
	//g.BorderLeft = true
	//g.BorderTop = false
	//g.BorderBottom = false
	g.BarColor = getColorForPercentage(percent)
	return CategoryGauge{g: g, cat: cat}
}

// ComposeGauges returns an array of ui.GridItem containers, ready to render.
func (d Dashboard) ComposeGauges(gauges []CategoryGauge) []ui.GridItem {
	var gridItems []ui.GridItem
	for _, gauge := range gauges {
		numCat := float64(len(gauges))
		ratio := 1.0 / numCat
		gridItems = append(gridItems, ui.NewRow(ratio, gauge.g))
	}
	return gridItems
}

// GenerateStackedBarChart will create a nice rendering of all categories and result counts.
func (d Dashboard) GenerateStackedBarChart(cats []*model.RuleCategory) CategoryChart {
	c := widgets.NewStackedBarChart()
	c.Title = "Linting Results: X-Axis=Category, Y-Axis=Issue Count (Errors, Warnings, Info)"

	var labels []string

	c.Data = make([][]float64, len(cats))

	for x, cat := range cats {
		labels = append(labels, cat.Name)
		errors := d.resultSet.GetErrorsByRuleCategory(cat.Id)
		warn := d.resultSet.GetWarningsByRuleCategory(cat.Id)
		info := d.resultSet.GetInfoByRuleCategory(cat.Id)
		c.Data[x] = []float64{float64(len(errors)), float64(len(warn)), float64(len(info))}
		c.BarWidth = 10
	}

	c.Labels = labels

	return CategoryChart{c: c, cats: cats}
}

// Render will render the dashboard.
func (d Dashboard) Render() {

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize CUI: %v", err)
	}

	d.run = true
	defer ui.Close()

	// extract categories and calculate coverage.
	var gauges []CategoryGauge
	var cats []*model.RuleCategory
	for l, cat := range model.RuleCategories {

		score := d.resultSet.CalculateCategoryHealth(l)
		gauges = append(gauges, d.GenerateGauge(cat.Name, score, cat))
		cats = append(cats, cat)
	}

	gridItems := d.ComposeGauges(gauges)

	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second * 5).C

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	p1 := widgets.NewParagraph()
	p1.Text = "A lovely dog"

	p2 := widgets.NewParagraph()
	p2.Text = "Nice boots in the winter"

	p3 := widgets.NewParagraph()
	p3.Text = "Warm apple pie and ice-cream"

	p4 := widgets.NewParagraph()
	p4.Text = "A lovely cold beer in the summer"

	p5 := widgets.NewParagraph()
	p5.Text = "Hugs and kisses from the family"

	p6 := widgets.NewParagraph()
	p6.Text = "A curry and a chat with your friends"

	p7 := widgets.NewParagraph()
	p7.Text = "A roaring fire in the winter"

	p8 := widgets.NewParagraph()
	p8.Text = "The goals we no longer seek due to a bigger vision"

	//chart := d.GenerateStackedBarChart(cats)
	tabs := d.GenerateTabbedView(cats)

	var para interface{}
	para = p1

	tabs.renderTab = func() {
		switch tabs.tv.ActiveTabIndex {
		case 0:
			para = p1
		case 1:
			para = p2
		case 2:
			para = p3
		case 3:
			para = p4
		case 4:
			para = p5
		case 5:
			para = p6
		case 6:
			para = p7
		case 7:
			para = p8

		}
	}

	var setGrid = func() {

		grid.Set(
			ui.NewRow(1.0,
				ui.NewCol(0.2,
					gridItems[0], gridItems[1], gridItems[2], gridItems[3],
					gridItems[4], gridItems[5], gridItems[6], gridItems[7],
				),
				ui.NewCol(0.8,
					ui.NewRow(0.1, tabs.tv),
					ui.NewRow(0.9, para),
				),
			),
		)
	}
	setGrid()
	ui.Render(grid)

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "[", "<Left>":
				tabs.tv.FocusLeft()
				ui.Clear()
				tabs.renderTab()
				setGrid()
				ui.Render(grid)
			case "]", "<Right>":
				tabs.tv.FocusRight()
				ui.Clear()
				tabs.renderTab()
				setGrid()
				ui.Render(grid)
			}
		case <-ticker:
			if d.run {
				ui.Render(grid)
			}
		}
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
