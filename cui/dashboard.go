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
	C   chan bool
	run bool
}

// CategoryGauge represents a percent bar visualizing how well spec did in a particular category
type CategoryGauge struct {
	g   *widgets.Gauge
	cat *model.RuleCategory
}

// GenerateGauge returns a CategoryGauge already populated with the title, color and percent.
func (d Dashboard) GenerateGauge(title string, percent int, cat *model.RuleCategory) CategoryGauge {
	g := widgets.NewGauge()
	g.Title = title
	g.Percent = percent
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

// Render will render the dashboard.
func (d Dashboard) Render() {

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize CUI: %v", err)
	}

	d.run = true
	defer ui.Close()

	// extract categories and calculate coverage.
	var gauges []CategoryGauge
	for _, cat := range model.RuleCategories {
		gauges = append(gauges, d.GenerateGauge(cat.Name, 75, cat))
	}

	gridItems := d.ComposeGauges(gauges)

	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second * 5).C

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	p := widgets.NewParagraph()
	p.Text = "Nice dash!"

	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(0.5, p),
			ui.NewCol(0.5,
				gridItems[0], gridItems[1], gridItems[2], gridItems[3],
				gridItems[4], gridItems[5], gridItems[6], gridItems[7],
			),
		),
	)

	ui.Render(grid)

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
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
	if percent > 30 && percent <= 50 {
		return ui.ColorYellow
	}
	if percent > 50 && percent <= 80 {
		return ui.ColorBlue
	}
	if percent > 80 {
		return ui.ColorGreen
	}
	return ui.ColorClear
}
