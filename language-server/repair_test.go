package languageserver

import (
	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestRepairCodeActionUsesVersionedBuffer(t *testing.T) {
	sets := rulesets.BuildDefaultRuleSets()
	state := NewServer("test", &utils.LintFileRequest{DefaultRuleSets: sets, SelectedRS: sets.GenerateOpenAPIRecommendedRuleSet(), TimeoutFlag: 5, LookupTimeoutFlag: 500})
	uri := "file:///tmp/vacuum-repair-test.json"
	content := `{"openapi":"3.1.0","info":{"title":"Test 🐈","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"enum":["a","a"]}}}}`
	doc := state.documentStore.addVersioned(uri, content, 7)
	action := state.repairCodeAction(uri)
	require.NotNil(t, action)
	edit := action.Edit.(repairWorkspaceEdit).DocumentChanges[0]
	require.Equal(t, protocol.Integer(7), edit.TextDocument.Version)
	require.Equal(t, uri, edit.TextDocument.URI)
	require.Equal(t, len(content), edit.Edits[0].Range.End.IndexIn(content))
	require.Equal(t, content, doc.Content)
	require.NotEqual(t, content, edit.Edits[0].NewText)
	require.ErrorContains(t, applyDocumentChanges(doc, []any{protocol.TextDocumentContentChangeEventWhole{Text: "stale"}}, 6), "stale")
	require.Equal(t, content, doc.Content)
	require.NoError(t, applyDocumentChanges(doc, []any{protocol.TextDocumentContentChangeEventWhole{Text: edit.Edits[0].NewText}}, 8))
	require.Nil(t, state.repairCodeAction(uri))
}
