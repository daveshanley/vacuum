// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/motor/autofix"

	tea "charm.land/bubbletea/v2"
	"github.com/pb33f/testify/require"
)

func TestRepairDiscoveryIsLazyAndRetriesOffUIThread(t *testing.T) {
	for _, key := range []rune{'f', 'v', 'a'} {
		t.Run(string(key), func(t *testing.T) {
			m, _ := repairTestModel(t)
			m.repair.close()
			calls := 0
			open := m.repair.config.Open
			m.repair.config.Open = func(source []byte) (*autofix.FixReview, error) {
				calls++
				if calls == 1 {
					return nil, fmt.Errorf("temporary failure")
				}
				return open(source)
			}
			m.Init()
			require.Nil(t, m.repair.discovery)
			m.Update(relintCompleteMsg{results: m.allResults, specContent: m.specContent})
			require.Nil(t, m.repair.discovery)
			require.Zero(t, calls)
			cmd := repairKey(m, key, string(key))
			require.NotNil(t, cmd)
			require.Zero(t, calls, "discovery must not execute in Update")
			m.Update(cmd())
			require.Contains(t, m.repair.status, "temporary failure")
			cmd = repairKey(m, key, string(key))
			require.NotNil(t, cmd)
			require.Equal(t, 1, calls, "retry must also run off the UI thread")
			m.Update(cmd())
			require.Equal(t, 2, calls)
			require.NotNil(t, m.repair.review)
			if key == 'f' {
				require.True(t, m.repair.onlyFixes)
			}
			if key == 'a' {
				require.Equal(t, ModalRepairSave, m.uiState.ActiveModal)
			}
		})
	}
}

func TestRepairUnusedKeysReachTableNavigation(t *testing.T) {
	m, _ := repairTestModel(t)
	m.repair.config.DisabledReason = "Saved report is read-only"
	for _, key := range []tea.KeyPressMsg{{Code: 'f', Text: "f"}, {Code: ' ', Text: " "}} {
		handled, cmd := m.handleRepairMessage(key)
		require.False(t, handled)
		require.Nil(t, cmd)
	}
	m.repair.config.DisabledReason = ""
	m.repair.options = nil
	m.repair.automatic, m.repair.suggested, m.repair.needsInput = 0, 0, 0
	handled, cmd := m.handleRepairMessage(tea.KeyPressMsg{Code: 'f', Text: "f"})
	require.False(t, handled)
	require.Nil(t, cmd)
	m.repair.close()
	m.repair.options = nil
	handled, cmd = m.handleRepairMessage(tea.KeyPressMsg{Code: ' ', Text: " "})
	require.False(t, handled)
	require.Nil(t, cmd)
}
