package tui

import (
	tea "charm.land/bubbletea/v2"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/testify/require"
)

func TestRepairVerifiedViewIncludesDistantChanges(t *testing.T) {
	source := []byte("openapi: 3.1.0\ninfo:\n  title: Review\n  version: '1'\npaths: {}\nx-padding:\n" +
		strings.Repeat("  - unchanged\n", 300) + "components:\n  schemas:\n    Pet:\n      type: string\n")
	m, disk := repairSourceTestModel(t, source, "review.yaml", []string{"info-description", "component-description"})
	for _, option := range m.repair.options {
		if option.Available {
			m.repair.selected[option.Index] = true
		}
	}
	cmd := repairKey(m, 'v', "v")
	require.NotNil(t, cmd)
	m.Update(cmd())
	require.NotNil(t, m.repair.preview)
	require.Equal(t, 2, m.repair.preview.Fixed)
	require.Contains(t, m.repair.diff, "Pet schema.")
	require.NotContains(t, m.repair.diff, "more lines omitted")
	require.Equal(t, source, *disk, "review must not write")
}

func TestRepairImpactRequiresSeparateAcceptance(t *testing.T) {
	for _, status := range []motor.RepairImpactStatus{motor.RepairImpactBreaking, motor.RepairImpactUnknown} {
		t.Run(string(status), func(t *testing.T) {
			m, disk := repairTestModel(t)
			original := string(*disk)
			selectRepairRule(t, m, "info-description")
			repairKey(m, ' ', " ")
			cmd := repairKey(m, 'v', "v")
			require.NotNil(t, cmd)
			m.Update(cmd())
			require.NotNil(t, m.repair.preview)
			prepareRepairSave(t, m)
			m.repair.preview.Impact.Status = status
			repairKey(m, tea.KeyEnter, "")
			require.Equal(t, original, string(*disk))
			require.True(t, m.repair.confirmImpact)
			repairKey(m, tea.KeyEnter, "")
			require.NotEqual(t, original, string(*disk))
		})
	}
}

func TestRepairEditInvalidatesImpactAcceptance(t *testing.T) {
	m, disk := repairTestModel(t)
	original := string(*disk)
	selectRepairRule(t, m, "info-description")
	repairKey(m, ' ', " ")
	cmd := repairKey(m, 'v', "v")
	require.NotNil(t, cmd)
	m.Update(cmd())
	prepareRepairSave(t, m)
	m.repair.preview.Impact.Status = motor.RepairImpactUnknown
	repairKey(m, tea.KeyEnter, "")
	require.True(t, m.repair.confirmImpact)
	repairKey(m, 'e', "e")
	require.False(t, m.repair.confirmImpact)
	require.Nil(t, m.repair.preview)
	require.Equal(t, original, string(*disk))
}
