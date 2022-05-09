// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cui

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/gizak/termui/v3/widgets"
)

// CategoryGauge represents a percent bar visualizing how well spec did in a particular category
type CategoryGauge struct {
	g   *widgets.Gauge
	cat *model.RuleCategory
}

// NewCategoryGauge returns a new gauge widget that is ready to render
func NewCategoryGauge(title string, percent int, cat *model.RuleCategory) CategoryGauge {
	g := widgets.NewGauge()
	g.Title = title
	g.Percent = percent
	g.BarColor = getColorForPercentage(percent)
	g.BorderLeft = false
	g.BorderRight = false
	g.BorderBottom = false
	g.BorderTop = false
	g.PaddingTop = 1
	return CategoryGauge{g: g, cat: cat}
}
