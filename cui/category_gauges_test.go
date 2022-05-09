// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cui

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewCategoryGauge(t *testing.T) {

	g := NewCategoryGauge("tipt0p says hi.", 23, model.RuleCategories[model.CategorySecurity])
	assert.Equal(t, 23, g.g.Percent)

}
