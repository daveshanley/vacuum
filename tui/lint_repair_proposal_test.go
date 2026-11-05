package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/pb33f/testify/require"
)

func TestDashboardRequestBodyProposalCanBeAcceptedWithoutEditing(t *testing.T) {
	for _, fixture := range []struct {
		name   string
		source string
	}{
		{"request.yaml", "openapi: 3.0.3\ninfo: {title: Images, version: '1'}\npaths:\n  /pets/{petId}/uploadImage:\n    post:\n      description: Upload an image for a pet.\n      requestBody:\n        content:\n          application/octet-stream:\n            schema:\n              type: string\n              format: binary\n      responses:\n        '200':\n          description: Image received.\n"},
		{"request.json", `{"openapi":"3.0.3","info":{"title":"Images","version":"1"},"paths":{"/pets/{petId}/uploadImage":{"post":{"description":"Upload an image for a pet.","requestBody":{"content":{"application/octet-stream":{"schema":{"type":"string","format":"binary"}}}},"responses":{"200":{"description":"Image received."}}}}}}`},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			m, disk := repairSourceTestModel(t, []byte(fixture.source), fixture.name, []string{"operation-description"})
			require.Len(t, m.repair.options, 1)
			repairKey(m, 'f', "f")
			repairKey(m, tea.KeyEnter, "")
			require.Equal(t, ModalRepair, m.uiState.ActiveModal)
			option := m.currentRepairOption()
			require.True(t, option.Generated)
			require.NotEmpty(t, option.Value)
			require.False(t, m.repair.draft.NeedsValue)
			require.Contains(t, m.repair.draft.Before, "requestBody")
			require.NotContains(t, m.repair.draft.Before, "description")
			require.Contains(t, m.repair.draft.After, "requestBody")
			require.Contains(t, m.repair.draft.After, "description")
			require.Contains(t, m.repair.draft.After, option.Value)
			view := ansi.Strip(m.buildRepairModal())
			require.Contains(t, view, "enter: accept")
			require.Contains(t, view, "description")
			require.NotContains(t, view, "VALUE REQUIRED")
			require.NotContains(t, view, "No value has been proposed")
			require.False(t, m.repair.selected[option.Index])
			require.Equal(t, fixture.source, string(*disk))

			// Enter accepts the supplied content; editing is not a prerequisite.
			repairKey(m, tea.KeyEnter, "")
			require.Equal(t, ModalRepair, m.uiState.ActiveModal)
			require.False(t, m.repair.editing)
			require.True(t, m.repair.selected[option.Index])
			require.Contains(t, ansi.Strip(m.buildRepairModal()), "enter: deselect")
			require.Equal(t, option.Value, m.repair.values[option.Index])
			preview := repairKey(m, 'v', "v")
			require.NotNil(t, preview)
			m.Update(preview())
			require.NotNil(t, m.repair.preview)
			require.Equal(t, 1, m.repair.preview.Fixed)
			require.Empty(t, m.repair.preview.Results)
			require.Equal(t, fixture.source, string(*disk))
			require.Contains(t, string(m.repair.preview.Candidate), option.Value)
			repairKey(m, 'a', "a")
			require.Contains(t, string(*disk), option.Value)
			require.Empty(t, m.allResults)
			require.Contains(t, m.repair.status, "APPLIED: 1 fixes")
		})
	}
}
