// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package languageserver

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/require"
)

func TestRepairCodeActionUsesVersionedBuffer(t *testing.T) {
	sets := rulesets.BuildDefaultRuleSets()
	state := NewServer("test", &utils.LintFileRequest{DefaultRuleSets: sets, SelectedRS: sets.GenerateOpenAPIRecommendedRuleSet(), TimeoutFlag: 5, LookupTimeoutFlag: 500})
	uri := "file:///tmp/vacuum-repair-test.json"
	content := `{"openapi":"3.1.0","info":{"title":"Test 🐈","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"enum":["a","a"]}}}}`
	doc := state.documentStore.addVersioned(uri, content, 7)
	action := state.repairCodeAction(context.Background(), uri)
	require.NotNil(t, action)
	edit := action.Edit.(repairWorkspaceEdit).DocumentChanges[0]
	require.Equal(t, protocol.Integer(7), edit.TextDocument.Version)
	require.Equal(t, uri, edit.TextDocument.URI)
	require.Equal(t, len(content), edit.Edits[0].Range.End.IndexIn(content))
	require.Equal(t, content, doc.Content)
	require.NotEqual(t, content, edit.Edits[0].NewText)
	require.NoError(t, applyDocumentChanges(doc, []any{protocol.TextDocumentContentChangeEventWhole{Text: content + "\n"}}, 6))
	require.Equal(t, content+"\n", doc.Content)
	require.NoError(t, applyDocumentChanges(doc, []any{protocol.TextDocumentContentChangeEventWhole{Text: edit.Edits[0].NewText}}, 8))
	require.Nil(t, state.repairCodeAction(context.Background(), uri))
}

func repairTestServer(t *testing.T, content string) (*ServerState, string, *atomic.Int32) {
	t.Helper()
	sets := rulesets.BuildDefaultRuleSets()
	state := NewServer("test", &utils.LintFileRequest{DefaultRuleSets: sets, SelectedRS: sets.GenerateOpenAPIRecommendedRuleSet(), TimeoutFlag: 5, LookupTimeoutFlag: 500})
	uri := "file:///tmp/vacuum-repair-cache.json"
	state.documentStore.addVersioned(uri, content, 1)
	calls := &atomic.Int32{}
	state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
		calls.Add(1)
		return sets.GenerateOpenAPIRecommendedRuleSet()
	}
	return state, uri, calls
}

const repairCacheSpec = `{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["a","a"]}}}}`

func TestRepairCodeActionRequestGates(t *testing.T) {
	state, uri, calls := repairTestServer(t, repairCacheSpec)
	source := "vacuum"
	params := &protocol.CodeActionParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}, Context: protocol.CodeActionContext{Diagnostics: []protocol.Diagnostic{{Source: &source, Code: &protocol.IntegerOrString{Value: "duplicated-entry-in-enum"}, CodeDescription: &protocol.CodeDescription{HRef: "https://quobix.com/vacuum/rules/duplicated-entry-in-enum"}}}}}
	params.Context.Only = []string{"source.organizeImports"}
	require.Empty(t, state.codeActions(context.Background(), params))
	require.Equal(t, int32(0), calls.Load())
	params.Context.Only = []string{"quickfix"}
	params.Context.Diagnostics[0].Source = nil
	actions := state.codeActions(context.Background(), params)
	require.Len(t, actions, 1)
	require.Equal(t, "View documentation", actions[0].Title)
	require.Equal(t, int32(0), calls.Load())
	params.Context.Diagnostics = nil
	require.Empty(t, state.codeActions(context.Background(), params))
	require.Equal(t, int32(0), calls.Load())
	params.Context.Diagnostics = []protocol.Diagnostic{{Source: &source, Code: &protocol.IntegerOrString{Value: "duplicated-entry-in-enum"}}}
	for i := 0; i < 10; i++ {
		require.Len(t, state.codeActions(context.Background(), params), 1)
	}
	require.Equal(t, int32(1), calls.Load())
}

func TestRepairCodeActionCacheInvalidation(t *testing.T) {
	state, uri, calls := repairTestServer(t, repairCacheSpec)
	first := state.repairCodeAction(context.Background(), uri)
	require.NotNil(t, first)
	for i := 0; i < 10; i++ {
		require.NotNil(t, state.repairCodeAction(context.Background(), uri))
	}
	require.Equal(t, int32(1), calls.Load())
	state.bumpConfigGeneration()
	require.NotNil(t, state.repairCodeAction(context.Background(), uri))
	require.Equal(t, int32(2), calls.Load())
	doc, _ := state.documentStore.Get(uri)
	require.NoError(t, applyDocumentChanges(doc, []any{protocol.TextDocumentContentChangeEventWhole{Text: repairCacheSpec + "\n"}}, 2))
	require.NotNil(t, state.repairCodeAction(context.Background(), uri))
	require.Equal(t, int32(3), calls.Load())
	// A reopened document with the same version must not inherit the closed cache.
	state.documentStore.Remove(uri)
	require.Nil(t, state.repairCodeAction(context.Background(), uri))
	state.documentStore.addVersioned(uri, repairCacheSpec, 2)
	require.NotNil(t, state.repairCodeAction(context.Background(), uri))
	require.Equal(t, int32(4), calls.Load())
}

func TestRepairCodeActionCachesNoFixAndConcurrentRequests(t *testing.T) {
	state, uri, calls := repairTestServer(t, repairCacheSpec)
	var wg sync.WaitGroup
	actions := make(chan *protocol.CodeAction, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); actions <- state.repairCodeAction(context.Background(), uri) }()
	}
	wg.Wait()
	close(actions)
	var candidate string
	for action := range actions {
		require.NotNil(t, action)
		candidate = action.Edit.(repairWorkspaceEdit).DocumentChanges[0].Edits[0].NewText
	}
	require.Equal(t, int32(1), calls.Load())
	doc, _ := state.documentStore.Get(uri)
	require.NoError(t, applyDocumentChanges(doc, []any{protocol.TextDocumentContentChangeEventWhole{Text: candidate}}, 2))
	for i := 0; i < 10; i++ {
		require.Nil(t, state.repairCodeAction(context.Background(), uri))
	}
	require.Equal(t, int32(2), calls.Load())
}

func TestRepairCodeActionDiscardsStaleWork(t *testing.T) {
	for _, change := range []string{"buffer", "configuration", "closed document"} {
		t.Run(change, func(t *testing.T) {
			state, uri, _ := repairTestServer(t, repairCacheSpec)
			started, release := make(chan struct{}), make(chan struct{})
			state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
				close(started)
				<-release
				return state.lintRequest.SelectedRS
			}
			finished := make(chan *protocol.CodeAction, 1)
			go func() { finished <- state.repairCodeAction(context.Background(), uri) }()
			<-started
			doc, _ := state.documentStore.Get(uri)
			switch change {
			case "buffer":
				require.NoError(t, applyDocumentChanges(doc, []any{protocol.TextDocumentContentChangeEventWhole{Text: repairCacheSpec + "\n"}}, 2))
			case "configuration":
				state.bumpConfigGeneration()
			case "closed document":
				state.documentStore.Remove(uri)
			}
			close(release)
			require.Nil(t, <-finished)
			require.Nil(t, doc.repairCache)
		})
	}
}

func TestRepairCodeActionCachesErrors(t *testing.T) {
	state, uri, calls := repairTestServer(t, "openapi: [")
	for i := 0; i < 3; i++ {
		require.Nil(t, state.repairCodeAction(context.Background(), uri))
	}
	require.Equal(t, int32(1), calls.Load())
}

func TestRepairCodeActionCancellationDoesNotBlockOrCache(t *testing.T) {
	state, uri, calls := repairTestServer(t, repairCacheSpec)
	started, release := make(chan struct{}), make(chan struct{})
	state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
		calls.Add(1)
		close(started)
		<-release
		return state.lintRequest.SelectedRS
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan *protocol.CodeAction, 1)
	go func() { finished <- state.repairCodeAction(ctx, uri) }()
	<-started
	waiting, cancelWaiting := context.WithCancel(context.Background())
	waiter := make(chan *protocol.CodeAction, 1)
	go func() { waiter <- state.repairCodeAction(waiting, uri) }()
	cancelWaiting()
	select {
	case action := <-waiter:
		require.Nil(t, action)
	case <-time.After(time.Second):
		t.Fatal("canceled request waited for another repair")
	}
	cancel()
	close(release)
	require.Nil(t, <-finished)
	doc, _ := state.documentStore.Get(uri)
	require.Nil(t, doc.repairCache)
	require.Equal(t, int32(1), calls.Load())
}

func TestRepairCodeActionRejectsRepeatedVersionDuringWork(t *testing.T) {
	state, uri, _ := repairTestServer(t, repairCacheSpec)
	started, release := make(chan struct{}), make(chan struct{})
	state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
		close(started)
		<-release
		return state.lintRequest.SelectedRS
	}
	finished := make(chan *protocol.CodeAction, 1)
	go func() { finished <- state.repairCodeAction(context.Background(), uri) }()
	<-started
	doc, _ := state.documentStore.Get(uri)
	require.NoError(t, applyDocumentChanges(doc, []any{protocol.TextDocumentContentChangeEventWhole{Text: repairCacheSpec}}, 1))
	close(release)
	require.Nil(t, <-finished)
}

func TestOpenLintDoesNotOfferNativeRepairEdits(t *testing.T) {
	sets := rulesets.BuildDefaultRuleSets()
	state := NewServerWithExecutionOptions("test", &utils.LintFileRequest{DefaultRuleSets: sets, SelectedRS: sets.GenerateOpenAPIRecommendedRuleSet()}, &motor.ExecutionOptions{Contract: model.OpenLintV1Draft})
	uri := "file:///tmp/openlint.json"
	doc := state.documentStore.addVersioned(uri, `{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"enum":["a","a"]}}}}`, 1)
	require.Nil(t, state.repairCodeAction(context.Background(), uri))
	require.Nil(t, doc.repairCache)
}
